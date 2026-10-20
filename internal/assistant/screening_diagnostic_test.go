package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type diagnosticBrokenReader struct{}

func (diagnosticBrokenReader) Read([]byte) (int, error) { return 0, errors.New("PRIVATE-read-key") }
func (diagnosticBrokenReader) Close() error             { return nil }

func TestScreeningDiagnosticsIOAndProductCompatibility(t *testing.T) {
	tests := []struct {
		name, code, base, ctype string
		status                  int
		body                    io.ReadCloser
		transport               error
		cancel                  bool
	}{
		{name: "request", code: "request", base: "provider_request"},
		{name: "transport", code: "transport", base: "provider_unavailable", transport: errors.New("PRIVATE-transport-key")},
		{name: "timeout", code: "timeout", base: "provider_timeout", transport: context.DeadlineExceeded, cancel: true},
		{name: "http400", code: "http_4xx", base: "provider_request", status: 400, ctype: "application/json; private=PRIVATE-header-key"},
		{name: "http422", code: "http_4xx", base: "provider_request", status: 422},
		{name: "http429", code: "http_4xx", base: "provider_unavailable", status: 429},
		{name: "http503", code: "http_5xx", base: "provider_unavailable", status: 503},
		{name: "redirect", code: "http_other", base: "provider_unavailable", status: 302},
		{name: "read", code: "read", base: "provider_protocol", status: 200, body: diagnosticBrokenReader{}},
		{name: "oversize", code: "oversize", base: "provider_protocol", status: 200, body: io.NopCloser(strings.NewReader(strings.Repeat("x", (1<<20)+1)))},
		{name: "sse", code: "envelope", base: "provider_protocol", status: 200, ctype: "text/event-stream; charset=utf-8"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// The same failure stays rejected with identical Error/BillingKnown in the
			// normal product path; screening only attaches the closed diagnostic.
			for _, screening := range []bool{false, true} {
				calls := 0
				client := doerFunc(func(*http.Request) (*http.Response, error) {
					calls++
					if test.transport != nil {
						return nil, test.transport
					}
					body := test.body
					if test.name == "oversize" {
						body = io.NopCloser(strings.NewReader(strings.Repeat("x", (1<<20)+1)))
					}
					if body == nil {
						body = io.NopCloser(strings.NewReader("PRIVATE-body-key"))
					}
					return &http.Response{StatusCode: test.status, Header: http.Header{"Content-Type": []string{test.ctype}}, Body: body}, nil
				})
				p, _ := NewStatementOpenRouterProvider("synthetic", client)
				in := LLMRequest{Policy: policyFixture(), Messages: []LLMMessage{{Role: "user", Content: "PRIVATE-request-key"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 1000, MaxOutputTokens: 100}
				in.Policy = ModelPolicy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000, VerifiedAt: time.Now().Add(-time.Minute)}
				in.DisableReasoning = true
				in.InputTokenUpperBound = 100000
				in.MaxOutputTokens = 16000
				in.Messages = []LLMMessage{{Role: "system", Content: "PRIVATE-request-key"}, {Role: "user", Content: "Synthetic"}}
				if screening {
					p.statementDocument = true
					p.screeningDocument = true
					in.screeningProof = syntheticScreeningProof()
					pin := statementextract.ScreeningPolicy()
					in.Policy = ModelPolicy{Model: pin.Model, Provider: pin.Provider, ProviderName: pin.ProviderName, InputMicrosPerMillion: pin.InputMicrosPerMillion, OutputMicrosPerMillion: pin.OutputMicrosPerMillion, VerifiedAt: time.Now().Add(-time.Minute)}
					in.DisableReasoning = true
					in.InputTokenUpperBound = 100000
					in.MaxOutputTokens = 16000
					in.Messages = []LLMMessage{{Role: "system", Content: "PRIVATE-request-key"}, {Role: "user", Content: "Synthetic"}}
				}
				if test.name == "request" {
					in.Schema = json.RawMessage(`invalid-schema`)
				}
				ctx, cancel := context.WithCancel(context.Background())
				if test.cancel {
					cancel()
				}
				defer cancel()
				out, err := p.Complete(ctx, in)
				expectedCalls := 1
				if test.name == "request" {
					expectedCalls = 0
				}
				if err == nil || err.Error() != test.base || out.BillingKnown || calls != expectedCalls {
					t.Fatalf("screening=%v calls=%d known=%v err=%v", screening, calls, out.BillingKnown, err)
				}
				if screening {
					var detail interface {
						ScreeningDiagnostic() statementextract.ScreeningDiagnostic
					}
					if !errors.As(err, &detail) {
						t.Fatal("lost diagnostic")
					}
					d := detail.ScreeningDiagnostic()
					raw, _ := json.Marshal(d)
					if d.Code != test.code || strings.Contains(string(raw), "PRIVATE") || d.CorrelationID != "" {
						t.Fatal(string(raw))
					}
					if test.status != 0 && d.HTTPStatus != test.status {
						t.Fatal(d)
					}
				} else if out.ScreeningDiagnostic != nil {
					t.Fatal("product diagnostic changed")
				}
			}
		})
	}
	for value, want := range map[string]string{"": "missing", "application/json; charset=utf-8": "json", "application/problem+json": "json", "text/event-stream": "event_stream", "PRIVATE-header": "other"} {
		if got := screeningContentType(value); got != want {
			t.Fatalf("content type %q != %q", got, want)
		}
	}
}
