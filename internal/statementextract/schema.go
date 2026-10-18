package statementextract

import "encoding/json"

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
	row["direction"] = map[string]any{"type": "string", "enum": []string{"", "debit", "credit"}}
	headerKeys := []string{"openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover"}
	labels := map[string]any{}
	header := map[string]any{}
	for _, key := range headerKeys {
		labels[key] = selection
		header[key] = selection
	}
	header["openingBookDate"] = selection
	header["closingBookDate"] = selection
	header["currency"] = selection
	header["labels"] = object(labels, headerKeys)
	allHeader := append(append([]string{}, headerKeys...), "currency", "labels", "openingBookDate", "closingBookDate")
	schema := object(map[string]any{"rows": map[string]any{"type": "array", "items": object(row, rowKeys)}, "header": object(header, allHeader), "service": map[string]any{"type": "array", "items": object(map[string]any{"spans": selection, "reason": map[string]any{"type": "string"}}, []string{"spans", "reason"})}}, []string{"rows", "header", "service"})
	raw, _ := json.Marshal(schema)
	return raw
}()
