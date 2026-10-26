package assistant

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func receiptPricingMember() GroupMembership {
	m := pricingMember()
	m.RequestFeeEvidence = ReceiptRequestFeeRouterCeilingOnly
	m.Qualification = "SPEC-0059 offline receipt fixture"
	m.QualifiedUntil = time.Now().Add(time.Hour)
	m.MaxImages = 1
	return m
}

func TestReceiptProductRequestCeilingAdmissionAndWire(t *testing.T) {
	for _, tc := range []struct {
		name                string
		request             any
		omit, omitImage, ok bool
		fault               string
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
		{name: "missing_zdr", omit: true, fault: "zdr"},
		{name: "token_price_over_cap", omit: true, fault: "price"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := receiptPricingMember()
			raw, _ := json.Marshal([]ModelGroup{{Revision: "receipt-0059-v1", Purpose: "receipt", Profile: DocumentReading, Members: []GroupMembership{m}}})
			f := &requestPricingFixture{omitRequest: tc.omit, request: tc.request, image: "0.000000275", omitImage: tc.omitImage}
			f.inner.fault = tc.fault
			groups, err := NewModelGroups(string(raw), "offline", f)
			if err != nil {
				t.Fatal(err)
			}
			p, err := groups.Resolve(context.Background(), "receipt", 1)
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
			if err != nil || reserve != 5601 {
				t.Fatal("positive request and image ceilings not reserved", reserve, err)
			}
			request, err := DocumentRequest(p, groupSource(t), receiptDocumentPrompt, receiptDocumentSchema(), 16000)
			if err != nil {
				t.Fatal(err)
			}
			if f.inner.posts != 0 {
				t.Fatal("preparation sent POST")
			}
			provider, err := NewGroupedOpenRouterProvider("offline", groups, f)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = provider.Complete(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			route := f.inner.wire["provider"].(map[string]any)
			caps := route["max_price"].(map[string]any)
			only := route["only"].([]any)
			if caps["request"] != 0.001 || caps["image"] != 0.000001 || route["allow_fallbacks"] != false || len(only) != 1 || only[0] != "fixture/eu" || f.inner.posts != 1 || f.inner.gets != 3 || f.inner.wire["response_format"] == nil {
				t.Fatal("wire bounds or receipt schema changed", route)
			}
		})
	}
}

func TestReceiptRequestCeilingCannotExpandOtherProfiles(t *testing.T) {
	for _, tc := range []struct {
		name, purpose, profile, mode string
		cap                          int64
		images                       int
	}{
		{"strict_statement", "statement", DocumentReading, ReceiptRequestFeeRouterCeilingOnly, 1000, 1},
		{"plain_statement", "statement", PlainDocumentReading, ReceiptRequestFeeRouterCeilingOnly, 1000, 1},
		{"analysis", "analysis", TextReasoning, ReceiptRequestFeeRouterCeilingOnly, 1000, 1},
		{"zero", "receipt", DocumentReading, ReceiptRequestFeeRouterCeilingOnly, 0, 1},
		{"too_large", "receipt", DocumentReading, ReceiptRequestFeeRouterCeilingOnly, 1001, 1},
		{"multiple_images", "receipt", DocumentReading, ReceiptRequestFeeRouterCeilingOnly, 1000, 2},
		{"plain_mode", "receipt", DocumentReading, PlainRequestFeeRouterCeilingOnly, 1000, 1},
		{"experiment_mode", "receipt", DocumentReading, RequestFeeRouterCeilingOnly, 1000, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := receiptPricingMember()
			m.RequestFeeEvidence = tc.mode
			m.Policy.RequestFeeMicros = tc.cap
			m.MaxImages = tc.images
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
	if _, err := NewExperimentModelGroups(receiptPricingMember(), pricingGrant(), "offline", &requestPricingFixture{}); err == nil {
		t.Fatal("receipt product mode entered research")
	}
}

func TestReceiptMissingRequestPriceRequiresOperatorOptIn(t *testing.T) {
	m := receiptPricingMember()
	m.RequestFeeEvidence = ""
	raw, _ := json.Marshal([]ModelGroup{{Revision: "test", Purpose: "receipt", Profile: DocumentReading, Members: []GroupMembership{m}}})
	f := &requestPricingFixture{omitRequest: true, image: "0"}
	groups, err := NewModelGroups(string(raw), "offline", f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = groups.Resolve(context.Background(), "receipt", 1); err == nil || f.inner.posts != 0 {
		t.Fatal("missing price admitted without operator opt-in", err)
	}
}
