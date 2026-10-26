package receiptdrafts

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestExtractedReceiptClockProjection(t *testing.T) {
	for _, tc := range []struct {
		date, clock string
		normalized  bool
	}{
		{"2026-08-14", "00:00", true}, {"2026-08-14", "23:59", true},
		{"2024-02-29", "07:08:09", true}, {"2026-08-14", "23:59:59", true},
		{"2026-02-29", "07:08", false}, {"2026-02-30", "07:08", false},
		{"", "07:08", false}, {"2026-08-14", "24:00", false},
		{"2026-08-14", "07:60", false}, {"2026-08-14", "07:08:60", false},
		{"2026-08-14", "7:08", false}, {"2026-08-14", "07:8", false},
		{"2026-08-14", " 07:08", false}, {"2026-08-14", "07:08 ", false},
		{"2026-08-14", "07:08Z", false}, {"2026-08-14", "2026-08-13T07:08:09Z", false},
	} {
		t.Run(tc.date+"/"+tc.clock, func(t *testing.T) {
			raw := fixtureFields("account")
			raw.IssuedDate, raw.IssuedAt = ptr(tc.date), ptr(tc.clock)
			before, _ := json.Marshal(raw)
			candidate := normalizeExtractedFields(raw)
			after, _ := json.Marshal(raw)
			if string(before) != string(after) {
				t.Fatal("original changed")
			}
			if (candidate.IssuedAt == nil) != tc.normalized || !reflect.DeepEqual(candidate.Items, raw.Items) || !reflect.DeepEqual(candidate.IssuedDate, raw.IssuedDate) {
				t.Fatal("incorrect projection")
			}
			if _, err := Validate(candidate); (err == nil) != tc.normalized {
				t.Fatal("unsafe date accepted or valid projection rejected", err)
			}
			if _, err := Validate(raw); err == nil {
				t.Fatal("manual clock normalized by financial validator")
			}
		})
	}
	raw := fixtureFields("account")
	raw.IssuedDate = nil
	raw.IssuedAt = ptr("2026-08-14T07:08:09+05:00")
	if !reflect.DeepEqual(normalizeExtractedFields(raw), raw) {
		t.Fatal("timestamp changed")
	}
	if _, err := Validate(raw); err != nil {
		t.Fatal(err)
	}
	raw.IssuedAt = ptr("07:08")
	if !reflect.DeepEqual(normalizeExtractedFields(raw), raw) {
		t.Fatal("missing date guessed")
	}
}

func TestReceiptProjectionOnlyForUntouchedExtractedCandidate(t *testing.T) {
	raw := fixtureFields("account")
	raw.IssuedAt = ptr("07:08")
	for _, status := range []string{"uploaded", "extracting", "reviewed", "confirmed", "cancelled", "expired", "extracted"} {
		d := Draft{Status: status, Revision: 2, Fields: raw, Extractions: []Extracted{{Fields: raw, SourceText: "Original clock 07:08"}}}
		history := string(marshal(d.Extractions))
		projectExtractedCandidate(&d)
		if (d.Fields.IssuedAt == nil) != (status == "extracted") || d.Revision != 2 || string(marshal(d.Extractions)) != history {
			t.Fatal(status, "candidate or original incorrectly changed")
		}
	}
	d := Draft{Status: "extracted", Fields: raw, Extractions: []Extracted{{Fields: raw}}}
	d.Fields.Seller = ptr("Owner edit")
	projectExtractedCandidate(&d)
	if d.Fields.IssuedAt == nil {
		t.Fatal("owner edit projected")
	}
}

func TestReceiptFreeItemsAndPositivePurchase(t *testing.T) {
	f := fixtureFields("account")
	f.Items = append(f.Items, Item{Name: ptr("Complimentary item"), Quantity: ptr("2"), UnitPrice: ptr("0"), LineTotal: ptr("0")})
	v, err := Validate(f)
	if err != nil || v.Event.AmountMinor != 18000 || len(v.Receipt.Items) != 2 || v.Receipt.Items[1].QuantityMilli != 2000 || v.Receipt.Items[1].LineTotalMinor != 0 {
		t.Fatal(v, err)
	}
	for name, change := range map[string]func(*Fields){
		"all free":        func(f *Fields) { f.Items = f.Items[1:]; f.Total = ptr("0") },
		"negative price":  func(f *Fields) { f.Items[1].UnitPrice = ptr("-1") },
		"negative line":   func(f *Fields) { f.Items[1].LineTotal = ptr("-1") },
		"zero quantity":   func(f *Fields) { f.Items[1].Quantity = ptr("0") },
		"mismatch":        func(f *Fields) { f.Items[1].LineTotal = ptr("1") },
		"excess discount": func(f *Fields) { f.Items[1].Discount = ptr("1") },
		"overflow": func(f *Fields) {
			f.Items[0].Quantity = ptr("9223372036854775")
			f.Items[0].UnitPrice = ptr("92233720368547758.07")
		},
		"sum overflow": func(f *Fields) {
			f.Total = ptr("92233720368547758.07")
			f.Items[0].Quantity = ptr("1")
			f.Items[0].UnitPrice = f.Total
			f.Items[0].LineTotal = f.Total
			f.Items[0].Discount = nil
			f.Items[1].UnitPrice = ptr("1")
			f.Items[1].LineTotal = ptr("2")
		},
	} {
		t.Run(name, func(t *testing.T) {
			var bad Fields
			if err := json.Unmarshal(marshal(f), &bad); err != nil {
				t.Fatal(err)
			}
			change(&bad)
			if _, err := Validate(bad); err == nil {
				t.Fatal("invalid free-item purchase accepted")
			}
		})
	}
}
