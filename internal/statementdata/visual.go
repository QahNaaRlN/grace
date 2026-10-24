package statementdata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

type ImageQuote struct {
	Page  int        `json:"page"`
	Quote string     `json:"quote"`
	Box   [4]float64 `json:"box"`
}
type VisualField struct {
	Value    string       `json:"value"`
	Evidence []ImageQuote `json:"evidence"`
}
type VisualRow struct {
	Anchor []ImageQuote            `json:"anchor"`
	Fields map[string]*VisualField `json:"fields"`
	Issues []string                `json:"issues"`
}
type VisualReading struct {
	Header map[string]*VisualField `json:"header"`
	Rows   []VisualRow             `json:"rows"`
}
type VisualState struct {
	RowDigests    map[string]string `json:"rowDigests"`
	Recognition   string            `json:"recognition"`
	SourceBinding string            `json:"sourceBinding"`
	SourceSHA256  string            `json:"sourceSHA256"`
	Candidate     VisualReading     `json:"candidate"`
	Original      VisualReading     `json:"original"`
	ReviewedRows  map[string]string `json:"reviewedRows"`
}

var VisualRowFields = []string{"operationDate", "bookingDate", "accountAmount", "accountCurrency", "direction", "transactionAmount", "transactionCurrency", "description"}
var VisualHeaderFields = []string{"openingBookDate", "closingBookDate", "openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover", "currency"}

func VisualHash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func VisualRowID(i int) string { return fmt.Sprintf("row-%d", i+1) }
func (v VisualState) RowDigest(i int) string {
	if i < 0 || i >= len(v.Candidate.Rows) {
		return ""
	}
	return VisualHash(struct {
		Source string
		Row    VisualRow
	}{v.SourceSHA256, v.Candidate.Rows[i]})
}
func (v VisualReading) Validate(pages int) error {
	fail := errors.New("incomplete_reading")
	if pages < 1 || pages > 4 || len(v.Rows) < 1 || len(v.Rows) > 200 || len(v.Header) != len(VisualHeaderFields) {
		return fail
	}
	quote := func(q []ImageQuote) bool {
		if len(q) < 1 || len(q) > 8 {
			return false
		}
		for _, s := range q {
			b := s.Box
			if s.Page < 1 || s.Page > pages || strings.TrimSpace(s.Quote) == "" || len(s.Quote) > 8000 {
				return false
			}
			for _, n := range b {
				if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1 {
					return false
				}
			}
			if b[0] >= b[2] || b[1] >= b[3] {
				return false
			}
		}
		return true
	}
	fields := func(f map[string]*VisualField, keys []string) bool {
		if len(f) != len(keys) {
			return false
		}
		for _, k := range keys {
			value, ok := f[k]
			if !ok {
				return false
			}
			if value != nil && (len(value.Value) > 16000 || strings.TrimSpace(value.Value) == "" || !quote(value.Evidence)) {
				return false
			}
		}
		return true
	}
	if !fields(v.Header, VisualHeaderFields) {
		return fail
	}
	for _, r := range v.Rows {
		if !quote(r.Anchor) || !fields(r.Fields, VisualRowFields) || len(r.Issues) > 32 {
			return fail
		}
		for _, issue := range r.Issues {
			if issue == "" || len(issue) > 200 {
				return fail
			}
		}
	}
	return nil
}
func VisualSchema() json.RawMessage {
	quote := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"page", "quote", "box"}, "properties": map[string]any{"page": map[string]any{"type": "integer"}, "quote": map[string]any{"type": "string"}, "box": map[string]any{"type": "array", "items": map[string]any{"type": "number"}, "minItems": 4, "maxItems": 4}}}
	quotes := map[string]any{"type": "array", "items": quote}
	field := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"value", "evidence"}, "properties": map[string]any{"value": map[string]any{"type": "string"}, "evidence": quotes}}
	nullable := map[string]any{"anyOf": []any{map[string]any{"type": "null"}, field}}
	fields := func(keys []string) map[string]any {
		props := map[string]any{}
		for _, k := range keys {
			props[k] = nullable
		}
		return map[string]any{"type": "object", "additionalProperties": false, "required": keys, "properties": props}
	}
	row := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"anchor", "fields", "issues"}, "properties": map[string]any{"anchor": quotes, "fields": fields(VisualRowFields), "issues": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}}
	b, _ := json.Marshal(map[string]any{"type": "object", "additionalProperties": false, "required": []string{"header", "rows"}, "properties": map[string]any{"header": fields(VisualHeaderFields), "rows": map[string]any{"type": "array", "items": row}}})
	return b
}

// VisualReview validates arithmetic and readability, never independent OCR
// grounding. Caller separately verifies source, row digests, pages and roles.
func (v VisualReading) Review(pages int) (Review, error) {
	if e := v.Validate(pages); e != nil {
		return Review{}, e
	}
	out := Review{Rows: []Row{}, Header: Header{Evidence: []Evidence{}}}
	value := func(m map[string]*VisualField, k string) string {
		if f := m[k]; f != nil {
			return f.Value
		}
		return ""
	}
	for i, src := range v.Rows {
		row := Row{Ordinal: i + 1, Description: value(src.Fields, "description"), Direction: value(src.Fields, "direction"), Issues: append([]string{}, src.Issues...), Evidence: []Evidence{}}
		for _, k := range VisualRowFields {
			if f := src.Fields[k]; f != nil {
				row.Evidence = append(row.Evidence, Evidence{Role: k, Literal: f.Value, Spans: []Span{}, Image: f.Evidence})
			}
		}
		date, e := ParseLocalDate(value(src.Fields, "operationDate"), "")
		if e != nil {
			row.Issues = append(row.Issues, "operation_date_required")
		} else {
			row.OperationDate = date
		}
		if raw := value(src.Fields, "bookingDate"); raw != "" {
			row.BookingDate, e = ParseLocalDate(raw, "")
			if e != nil {
				row.Issues = append(row.Issues, "booking_date_invalid")
			}
		}
		currency, e := Currency(value(src.Fields, "accountCurrency"))
		if e != nil || currency != "RUB" {
			row.Issues = append(row.Issues, "account_currency_required")
		}
		amount, dir, e := ParseMoney(value(src.Fields, "accountAmount"), currency)
		if e != nil || amount.AmountMinor <= 0 {
			row.Issues = append(row.Issues, "account_amount_required")
		} else {
			row.AccountAmount = &amount
			if dir != "" && dir != row.Direction {
				row.Issues = append(row.Issues, "direction_conflict")
			}
		}
		if row.Direction != "debit" && row.Direction != "credit" {
			row.Issues = append(row.Issues, "direction_required")
		}
		if row.Description == "" {
			row.Issues = append(row.Issues, "description_required")
		}
		if value(src.Fields, "transactionAmount") != "" || value(src.Fields, "transactionCurrency") != "" {
			c, e := Currency(value(src.Fields, "transactionCurrency"))
			if e != nil {
				row.Issues = append(row.Issues, "transaction_currency_required")
			}
			m, _, e := ParseMoney(value(src.Fields, "transactionAmount"), c)
			if e != nil || m.AmountMinor <= 0 {
				row.Issues = append(row.Issues, "transaction_amount_invalid")
			} else {
				row.TransactionAmount = &m
			}
		}
		out.Rows = append(out.Rows, row)
	}
	for _, k := range VisualHeaderFields {
		if f := v.Header[k]; f != nil {
			out.Header.Evidence = append(out.Header.Evidence, Evidence{Role: k, Literal: f.Value, Spans: []Span{}, Image: f.Evidence})
		}
	}
	headerIssues := []string{}
	currency, e := Currency(value(v.Header, "currency"))
	for _, f := range []struct {
		k      string
		target **Money
	}{{"openingBookBalance", &out.Header.OpeningBookBalance}, {"closingBookBalance", &out.Header.ClosingBookBalance}, {"availableBalanceObserved", &out.Header.AvailableBalanceObserved}, {"debitTurnover", &out.Header.DebitTurnover}, {"creditTurnover", &out.Header.CreditTurnover}} {
		raw := value(v.Header, f.k)
		if raw == "" {
			continue
		}
		if e != nil || currency != "RUB" {
			headerIssues = append(headerIssues, "header_currency_required")
			continue
		}
		m, dir, err := ParseMoney(raw, currency)
		if err != nil {
			headerIssues = append(headerIssues, f.k+":invalid_money")
			continue
		}
		if dir == "debit" {
			m.AmountMinor = -m.AmountMinor
			if f.k == "debitTurnover" || f.k == "creditTurnover" {
				headerIssues = append(headerIssues, "negative_turnover")
			}
		}
		*f.target = &m
	}
	for _, f := range []struct {
		k      string
		target **LocalDate
	}{{"openingBookDate", &out.Header.OpeningBookDate}, {"closingBookDate", &out.Header.ClosingBookDate}} {
		if raw := value(v.Header, f.k); raw != "" {
			d, err := ParseLocalDate(raw, "")
			if err != nil {
				headerIssues = append(headerIssues, f.k+":invalid_date")
			} else {
				*f.target = d
			}
		}
	}
	out.Reconciliation = Reconcile(out.Rows, out.Header)
	out.Reconciliation.Blockers = append(out.Reconciliation.Blockers, headerIssues...)
	out.Reconciliation.Warnings = append(out.Reconciliation.Warnings, "image_review_not_independent_ocr")
	return out, nil
}
