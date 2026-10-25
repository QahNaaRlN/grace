package assistant

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Mutate only the real endpoint metadata wire; catalog/ZDR and the grouped
// generation transport remain the existing fixtures.
type requestPricingFixture struct {
	inner                  groupFixture
	omitRequest, omitImage bool
	request, image         any
}

func (f *requestPricingFixture) Do(r *http.Request) (*http.Response, error) {
	response, err := f.inner.Do(r)
	if err != nil || r.Method != "GET" || !strings.HasSuffix(r.URL.Path, "/endpoints") {
		return response, err
	}
	raw, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	var v map[string]any
	if err = json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	for _, item := range v["data"].(map[string]any)["endpoints"].([]any) {
		price := item.(map[string]any)["pricing"].(map[string]any)
		if f.omitRequest {
			delete(price, "request")
		} else {
			price["request"] = f.request
		}
		if f.omitImage {
			delete(price, "image")
		} else {
			price["image"] = f.image
		}
	}
	return groupJSON(v), nil
}
func pricingMember() GroupMembership {
	return GroupMembership{RequestFeeEvidence: RequestFeeRouterCeilingOnly, Policy: ModelPolicy{Model: "fixture/vision-a", Provider: "fixture/eu", ProviderName: "Fixture", InputMicrosPerMillion: 100000, OutputMicrosPerMillion: 100000, RequestFeeMicros: 1000, ImageFeeMicros: 1, Vision: true, ImageTokenUpperBound: 1000}, Canonical: "fixture/vision-a-v1", MaxImages: 4, MaxImageSide: 2048, Region: "eu"}
}
func pricingGrant() ExperimentGrant {
	return ExperimentGrant{Source: strings.Repeat("a", 40), Approval: strings.Repeat("b", 40), ManifestSHA256: strings.Repeat("c", 64), ExpiresAt: time.Now().Add(35 * time.Minute)}
}
func TestRRequestFeeEvidenceAndWireCaps(t *testing.T) {
	tests := []struct {
		name                  string
		omit                  bool
		request, image        any
		omitImage, strict, ok bool
	}{
		{name: "omitted_research", omit: true, image: "0.000000275", ok: true},
		{name: "explicit_zero", request: "0", image: "0", ok: true},
		{name: "explicit_upper_cap", request: "0.001", image: "0.000001", ok: true},
		{name: "strict_omission", omit: true, image: "0", strict: true},
		{name: "null", request: nil, image: "0"},
		{name: "empty", request: "", image: "0"},
		{name: "number", request: 0, image: "0"},
		{name: "bool", request: false, image: "0"},
		{name: "negative", request: "-0.001", image: "0"},
		{name: "nonfinite", request: "NaN", image: "0"},
		{name: "request_over_cap", request: "0.001001", image: "0"},
		{name: "missing_image", omit: true, omitImage: true},
		{name: "null_image", omit: true, image: nil},
		{name: "invalid_image", omit: true, image: "NaN"},
		{name: "image_over_cap", omit: true, image: "0.0000011"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := pricingMember()
			if tc.strict {
				m.RequestFeeEvidence = ""
			}
			f := &requestPricingFixture{omitRequest: tc.omit, request: tc.request, image: tc.image, omitImage: tc.omitImage}
			groups, err := NewExperimentModelGroups(m, pricingGrant(), "offline-group-key", f)
			if err != nil {
				t.Fatal(err)
			}
			p, err := groups.Resolve(context.Background(), "statement", 1)
			if (err == nil) != tc.ok {
				t.Fatal("admission", err)
			}
			if !tc.ok {
				if f.inner.posts != 0 {
					t.Fatal("refusal sent POST")
				}
				return
			}
			reserve, err := p.ReservationMicros(30000, 16000)
			if err != nil || reserve != 5604 {
				t.Fatal("cap not reserved", reserve, err)
			}
			request, err := DocumentRequest(p, groupSource(t), "Read", json.RawMessage(`{"type":"object"}`), 16000)
			if err != nil {
				t.Fatal(err)
			}
			provider, err := NewGroupedOpenRouterProvider("offline-group-key", groups, f)
			if err != nil {
				t.Fatal(err)
			}
			_, err = provider.Complete(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			wire := f.inner.wire["provider"].(map[string]any)
			caps := wire["max_price"].(map[string]any)
			if f.inner.posts != 1 || caps["request"] != 0.001 || caps["image"] != 0.000001 || wire["allow_fallbacks"] != false || len(wire["only"].([]any)) != 1 || wire["only"].([]any)[0] != "fixture/eu" {
				t.Fatal("wire caps/routing", wire)
			}
		})
	}
}
func TestRFeeEvidenceCannotEnterProductionOrLosePositiveCap(t *testing.T) {
	m := pricingMember()
	m.Qualification = "ordinary qualified member"
	m.QualifiedUntil = time.Now().Add(time.Hour)
	raw, _ := json.Marshal([]ModelGroup{{Revision: "test", Purpose: "statement", Profile: DocumentReading, Members: []GroupMembership{m}}})
	f := &requestPricingFixture{omitRequest: true, image: "0"}
	if _, err := NewModelGroups(string(raw), "offline", f); err == nil {
		t.Fatal("research fee mode entered production")
	}
	for _, tc := range []struct {
		mode string
		cap  int64
	}{{RequestFeeRouterCeilingOnly, 0}, {RequestFeeRouterCeilingOnly, -1}, {"unknown", 1000}} {
		m = pricingMember()
		m.RequestFeeEvidence = tc.mode
		m.Policy.RequestFeeMicros = tc.cap
		if _, err := NewExperimentModelGroups(m, pricingGrant(), "offline", f); err == nil {
			t.Fatal("invalid evidence accepted", tc)
		}
	}
	if f.inner.gets != 0 || f.inner.posts != 0 {
		t.Fatal("invalid config touched network")
	}
}
