package assistant

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestPlainProductRequestCeilingAdmissionAndWire(t *testing.T) {
	cases := []struct {
		name                string
		request             any
		omit, omitImage, ok bool
	}{
		{name: "absent_with_reserved_cap", omit: true, ok: true},
		{name: "explicit_zero", request: "0", ok: true},
		{name: "explicit_cap", request: "0.001", ok: true},
		{name: "null", request: nil},
		{name: "empty", request: ""},
		{name: "number", request: 0},
		{name: "negative", request: "-0.001"},
		{name: "malformed", request: "NaN"},
		{name: "over_cap", request: "0.001001"},
		{name: "missing_image", omit: true, omitImage: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := pricingMember()
			m.RequestFeeEvidence = PlainRequestFeeRouterCeilingOnly
			m.Qualification = "SPEC-0058 P offline fixture"
			m.QualifiedUntil = time.Now().Add(time.Hour)
			m.MaxImages = 5
			raw, _ := json.Marshal([]ModelGroup{{Revision: "plain-p-v1", Purpose: "statement", Profile: PlainDocumentReading, Members: []GroupMembership{m}}})
			f := &requestPricingFixture{omitRequest: tc.omit, request: tc.request, image: "0.000000275", omitImage: tc.omitImage}
			groups, err := NewModelGroups(string(raw), "offline", f)
			if err != nil {
				t.Fatal(err)
			}
			p, err := groups.Resolve(context.Background(), "statement", 5)
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
			if err != nil || reserve != 5605 {
				t.Fatal("positive cap not fully reserved", reserve, err)
			}
			req, err := DocumentRequest(p, groupSource(t), "Read", nil, 16000)
			if err != nil {
				t.Fatal(err)
			}
			provider, err := NewGroupedOpenRouterProvider("offline", groups, f)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = provider.Complete(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			route := f.inner.wire["provider"].(map[string]any)
			caps := route["max_price"].(map[string]any)
			if caps["request"] != 0.001 || caps["image"] != 0.000001 || route["allow_fallbacks"] != false || f.inner.posts != 1 || f.inner.gets != 3 || f.inner.wire["response_format"] != nil {
				t.Fatal("wire caps or call bounds", route)
			}
		})
	}
}

func TestPlainRequestCeilingCannotExpandOtherProfiles(t *testing.T) {
	for _, tc := range []struct {
		name, purpose, profile, mode string
		cap                          int64
	}{
		{"strict_statement", "statement", DocumentReading, PlainRequestFeeRouterCeilingOnly, 1000},
		{"receipt", "receipt", DocumentReading, PlainRequestFeeRouterCeilingOnly, 1000},
		{"analysis", "analysis", TextReasoning, PlainRequestFeeRouterCeilingOnly, 1000},
		{"zero", "statement", PlainDocumentReading, PlainRequestFeeRouterCeilingOnly, 0},
		{"too_large", "statement", PlainDocumentReading, PlainRequestFeeRouterCeilingOnly, 1001},
		{"experiment_only", "statement", PlainDocumentReading, RequestFeeRouterCeilingOnly, 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := pricingMember()
			m.RequestFeeEvidence = tc.mode
			m.Policy.RequestFeeMicros = tc.cap
			m.Qualification = "SPEC-0058 P fixture"
			m.QualifiedUntil = time.Now().Add(time.Hour)
			raw, _ := json.Marshal([]ModelGroup{{Revision: "test", Purpose: tc.purpose, Profile: tc.profile, Members: []GroupMembership{m}}})
			f := &requestPricingFixture{omitRequest: true, image: "0"}
			if _, err := NewModelGroups(string(raw), "offline", f); err == nil {
				t.Fatal("unsupported evidence admitted")
			}
			if f.inner.gets != 0 || f.inner.posts != 0 {
				t.Fatal("invalid configuration touched network")
			}
		})
	}
	m := pricingMember()
	m.RequestFeeEvidence = PlainRequestFeeRouterCeilingOnly
	if _, err := NewExperimentModelGroups(m, pricingGrant(), "offline", &requestPricingFixture{}); err == nil {
		t.Fatal("product-only mode entered research")
	}
}

func TestPlainMissingRequestPriceStillRequiresOperatorOptIn(t *testing.T) {
	m := pricingMember()
	m.RequestFeeEvidence = ""
	m.Qualification = "SPEC-0058 P fixture"
	m.QualifiedUntil = time.Now().Add(time.Hour)
	raw, _ := json.Marshal([]ModelGroup{{Revision: "test", Purpose: "statement", Profile: PlainDocumentReading, Members: []GroupMembership{m}}})
	f := &requestPricingFixture{omitRequest: true, image: "0"}
	groups, err := NewModelGroups(string(raw), "offline", f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = groups.Resolve(context.Background(), "statement", 1); err == nil || f.inner.posts != 0 {
		t.Fatal("missing price admitted without explicit opt-in", err)
	}
}
