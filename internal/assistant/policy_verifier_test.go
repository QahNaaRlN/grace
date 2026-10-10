package assistant

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPolicyRefreshRetainsApprovedBoundary(t *testing.T) {
	cases := []struct {
		name, model, provider, price string
		want                         bool
	}{
		{"same", "google/gemini-2.5-flash", "google-vertex", "0.0000003", true},
		{"cheaper", "google/gemini-2.5-flash", "google-vertex", "0.0000002", true},
		{"more_expensive", "google/gemini-2.5-flash", "google-vertex", "0.0000004", false},
		{"another_model", "google/gemini-2.5-pro", "google-vertex", "0.0000003", false},
		{"another_provider", "google/gemini-2.5-flash", "google-ai-studio", "0.0000003", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "" {
					t.Fatal("metadata forwarded a secret")
				}
				fmt.Fprintf(w, `{"data":{"endpoints":[{"model_id":%q,"tag":%q,"provider_name":"Google","status":0,"supported_parameters":["structured_outputs","response_format","max_tokens"],"pricing":{"prompt":%q,"completion":"0.0000025"}}]}}`, c.model, c.provider, c.price)
			}))
			defer server.Close()
			v := NewPolicyVerifier()
			v.endpoint = server.URL + "/"
			p := ModelPolicy{Model: "google/gemini-2.5-flash", Provider: "google-vertex", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000, VerifiedAt: time.Now().Add(-25 * time.Hour)}
			fresh, err := v.Current(context.Background(), p)
			if (err == nil) != c.want {
				t.Fatalf("policy %+v error %v", fresh, err)
			}
			if c.want && (fresh.Model != p.Model || fresh.Provider != p.Provider || fresh.InputMicrosPerMillion != p.InputMicrosPerMillion || !fresh.Valid(time.Now())) {
				t.Fatal("approval changed")
			}
			v.Current(context.Background(), p)
			if calls != 1 {
				t.Fatal("unbounded metadata retries")
			}
		})
	}
}

func TestVisionRefreshRequiresKnownImagePrice(t *testing.T) {
	for _, imagePrice := range []string{"0.0000003", "0.0000004", ""} {
		t.Run(imagePrice, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"data":{"endpoints":[{"model_id":"google/gemini-2.5-flash","tag":"google-vertex","provider_name":"Google","status":0,"supported_parameters":["structured_outputs","response_format","max_tokens"],"pricing":{"prompt":"0.0000003","completion":"0.0000025","image":%q}}]}}`, imagePrice)
			}))
			defer server.Close()
			v := NewPolicyVerifier()
			v.endpoint = server.URL + "/"
			p := ModelPolicy{Model: "google/gemini-2.5-flash", Provider: "google-vertex", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000, Vision: true, ImageTokenUpperBound: 4200, VerifiedAt: time.Now().Add(-25 * time.Hour)}
			_, err := v.Current(context.Background(), p)
			if (err == nil) != (imagePrice == "0.0000003") {
				t.Fatalf("image price %q error %v", imagePrice, err)
			}
		})
	}
}
