package statementextract

import (
	"encoding/json"
)

var ProposalSchema = func() json.RawMessage {
	aliasRef := map[string]any{"$ref": "#/$defs/alias"}
	selection := map[string]any{"$ref": "#/$defs/selection"}
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
	currencySelection := map[string]any{"type": "array", "minItems": 0, "maxItems": 1, "items": aliasRef, "description": "Exactly ONE printed currency token, such as RUB or USD. Fill independently for each nonempty amount, even when the SAME header token ID is reused across every row and header. Header currency never implicitly populates row fields. Never select all occurrences. Use [] only if printed currency is unknown."}
	currencyRef := map[string]any{"$ref": "#/$defs/currencySelection"}
	row["accountCurrency"] = currencyRef
	row["transactionCurrency"] = currencyRef
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
	header["currency"] = currencyRef
	header["labels"] = object(labels, headerKeys)
	allHeader := append(append([]string{}, headerKeys...), "currency", "labels", "openingBookDate", "closingBookDate")
	schema := object(map[string]any{"rows": map[string]any{"type": "array", "items": object(row, rowKeys)}, "header": object(header, allHeader), "service": map[string]any{"type": "array", "items": object(map[string]any{"spans": selection, "reason": map[string]any{"type": "string"}}, []string{"spans", "reason"})}}, []string{"rows", "header", "service"})
	schema["$defs"] = map[string]any{
		"alias":             map[string]any{"type": "string"},
		"selection":         map[string]any{"type": "array", "minItems": 0, "items": aliasRef},
		"currencySelection": currencySelection,
		"moneySelection":    map[string]any{"type": "array", "minItems": 0, "items": map[string]any{"type": "string"}, "description": "Only the printed number/sign span IDs. Exclude all labels, headings and currencies. Use [] if absent. Hidden numeric fragments use number-class placeholders. Text-class fragments are never monetary candidates. Go enforces this eligibility and validates actual original values."}}
	raw, _ := json.Marshal(schema)
	return raw
}()

// RequestSchema restricts every selection to exact delivered aliases. Monetary
// selections additionally use server-owned lexical metadata before redaction.
// It never trusts placeholder-like source text. Numeric fragments remain hidden;
// original values still pass financial grounding, decimal and currency checks.
func RequestSchema(c Chunk) json.RawMessage {
	var schema map[string]any
	_ = json.Unmarshal(ProposalSchema, &schema)
	ids, numericIDs := []string{}, []string{}
	seen, numericSeen := map[string]bool{}, map[string]bool{}
	for _, span := range append(append([]RedactedSpan{}, c.Context...), c.Spans...) {
		if !seen[span.ID] {
			ids = append(ids, span.ID)
			seen[span.ID] = true
		}
		if !numericSeen[span.ID] && span.LexicalClass == ClassNumericFragment {
			numericIDs = append(numericIDs, span.ID)
			numericSeen[span.ID] = true
		}
	}
	defs := schema["$defs"].(map[string]any)
	if len(ids) > 0 {
		defs["alias"].(map[string]any)["enum"] = ids
	} else {
		defs["selection"].(map[string]any)["maxItems"] = 0
		defs["currencySelection"].(map[string]any)["maxItems"] = 0
	}
	money := defs["moneySelection"].(map[string]any)
	if len(numericIDs) > 0 {
		money["items"].(map[string]any)["enum"] = numericIDs
	} else {
		money["maxItems"] = 0
	}
	raw, _ := json.Marshal(schema)
	return raw
}
