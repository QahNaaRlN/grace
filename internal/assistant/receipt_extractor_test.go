package assistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReceiptDecoderRetainsLimitsOutsideProviderSchema(t *testing.T) {
	fixture := func() map[string]any {
		return map[string]any{"seller": "Синтетический магазин", "issuedAt": nil, "issuedDate": "2026-09-10", "currency": "RUB", "total": "90.00", "fiscalDocumentNumber": nil, "sourceText": "Искусственный чек", "items": []any{map[string]any{"name": "Молоко", "quantity": "1", "unitPrice": "90.00", "lineTotal": "90.00", "discount": nil, "brand": nil, "package": nil, "broadGroup": nil}}}
	}
	for _, name := range []string{"valid", "seller_too_long", "item_too_long", "source_too_long", "too_many_items", "owner_choice", "suggestion_injection"} {
		t.Run(name, func(t *testing.T) {
			input := fixture()
			switch name {
			case "seller_too_long":
				input["seller"] = strings.Repeat("я", 301)
			case "item_too_long":
				input["items"].([]any)[0].(map[string]any)["name"] = strings.Repeat("я", 301)
			case "source_too_long":
				input["sourceText"] = strings.Repeat("я", 12001)
			case "too_many_items":
				items := make([]any, 51)
				for i := range items {
					items[i] = input["items"].([]any)[0]
				}
				input["items"] = items
			case "owner_choice":
				input["accountId"] = "invented"
			case "suggestion_injection":
				input["items"].([]any)[0].(map[string]any)["groupSuggestion"] = map[string]any{"group": "invented"}
			}
			raw, _ := json.Marshal(input)
			out, err := decodeReceipt(LLMResponse{Content: raw, CostMicros: 123})
			if name != "valid" {
				if err == nil {
					t.Fatal("unsafe or over-limit receipt accepted")
				}
				return
			}
			if err != nil || out.Fields.IssuedAt != nil || out.Fields.IssuedDate == nil || *out.Fields.Total != "90.00" || out.CostMicros != "123" {
				t.Fatal(out, err)
			}
		})
	}
}
