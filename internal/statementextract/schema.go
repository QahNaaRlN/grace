package statementextract

import (
	"encoding/json"
	"strings"
)

var ProposalSchema = func() json.RawMessage {
	selection := map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	object := func(props map[string]any, keys []string) map[string]any {
		return map[string]any{"type": "object", "additionalProperties": false, "properties": props, "required": keys}
	}
	rowKeys := []string{"operationDate", "bookingDate", "accountAmount", "accountCurrency", "transactionAmount", "transactionCurrency", "description", "direction", "directionEvidence"}
	row := map[string]any{}
	for _, key := range rowKeys {
		row[key] = selection
	}
	moneyRef := map[string]any{"$ref": "#/$defs/moneySelection"}
	row["accountAmount"] = moneyRef
	row["transactionAmount"] = moneyRef
	currencySelection := map[string]any{"type": "array", "minItems": 0, "maxItems": 1, "items": map[string]any{"type": "string"}, "description": "Exactly ONE printed currency token, such as RUB or USD. Fill independently for each nonempty amount, even when the SAME header token ID is reused across every row and header. Header currency never implicitly populates row fields. Never select all occurrences. Use [] only if printed currency is unknown."}
	row["accountCurrency"] = currencySelection
	row["transactionCurrency"] = currencySelection
	row["direction"] = map[string]any{"type": "string", "enum": []string{"", "debit", "credit"}}
	headerKeys := []string{"openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover"}
	labels := map[string]any{}
	header := map[string]any{}
	for _, key := range headerKeys {
		labels[key] = selection
		header[key] = moneyRef
	}
	header["openingBookDate"] = selection
	header["closingBookDate"] = selection
	header["currency"] = currencySelection
	header["labels"] = object(labels, headerKeys)
	allHeader := append(append([]string{}, headerKeys...), "currency", "labels", "openingBookDate", "closingBookDate")
	schema := object(map[string]any{"rows": map[string]any{"type": "array", "items": object(row, rowKeys)}, "header": object(header, allHeader), "service": map[string]any{"type": "array", "items": object(map[string]any{"spans": selection, "reason": map[string]any{"type": "string"}}, []string{"spans", "reason"})}}, []string{"rows", "header", "service"})
	schema["$defs"] = map[string]any{"moneySelection": map[string]any{"type": "array", "minItems": 0, "items": map[string]any{"type": "string"}, "description": "Only the printed number/sign span IDs. Exclude all labels, headings and currencies. Use [] if absent. Redacted integer fragments may be opaque placeholders; Go validates their actual source text."}}
	raw, _ := json.Marshal(schema)
	return raw
}()

// RequestSchema narrows monetary choices to numeric/sign or opaque redacted
// fragments. Visible headings, dates and currency tokens cannot become an amount.
// Opaque fragments stay eligible so minimization never loses grouped integers;
// all selected original fragments still pass Go grounding and decimal checks.
func RequestSchema(c Chunk) json.RawMessage {
	var schema map[string]any
	_ = json.Unmarshal(ProposalSchema, &schema)
	ids := []string{}
	seen := map[string]bool{}
	for _, span := range append(append([]RedactedSpan{}, c.Context...), c.Spans...) {
		if !seen[span.ID] && (amountToken.MatchString(span.Text) || span.Text == "+" || span.Text == "-" || span.Text == "−" || strings.HasPrefix(span.Text, "[text_")) {
			ids = append(ids, span.ID)
			seen[span.ID] = true
		}
	}
	money := schema["$defs"].(map[string]any)["moneySelection"].(map[string]any)
	if len(ids) > 0 {
		money["items"].(map[string]any)["enum"] = ids
	} else {
		money["maxItems"] = 0
	}
	raw, _ := json.Marshal(schema)
	return raw
}
