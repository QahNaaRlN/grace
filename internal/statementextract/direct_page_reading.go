package statementextract

// S1.22 is a research-only reading protocol. Nothing here writes financial state.
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"golang.org/x/text/unicode/norm"
)

const DirectProfile = "statement.direct-page-reading.v1"
const DirectModel = "gpt-6-astra"
const DirectOutputLimit = 2 << 20

var DirectRowFields = []string{"operationDate", "bookingDate", "accountAmount", "accountCurrency", "direction", "transactionAmount", "transactionCurrency", "description"}
var DirectHeaderFields = []string{"openingBookDate", "closingBookDate", "openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover", "currency"}

type PageQuote struct {
	Page  int    `json:"page"`
	Quote string `json:"quote"`
}
type DirectField struct {
	Value    string      `json:"value"`
	Evidence []PageQuote `json:"evidence"`
}
type DirectRow struct {
	Anchor []PageQuote             `json:"anchor"`
	Fields map[string]*DirectField `json:"fields"`
	Issues []string                `json:"issues"`
}
type DirectReading struct {
	Header map[string]*DirectField `json:"header"`
	Rows   []DirectRow             `json:"rows"`
}

func DirectNormalize(s string) string {
	return strings.Join(strings.FieldsFunc(norm.NFC.String(s), func(r rune) bool { return strings.ContainsRune(" \t\r\n\v\f\u00a0\u202f", r) }), " ")
}
func DirectHash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }

// DirectJSON rejects duplicate keys, excessive nesting and trailing JSON before
// typed decoding. Unlike encoding/json alone, case aliases never select fields.
func DirectJSON(raw []byte, target any) error {
	if len(raw) == 0 || len(raw) > DirectOutputLimit || !utf8.Valid(raw) {
		return errors.New("direct_json_size_or_encoding")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 32 {
			return errors.New("direct_json_depth")
		}
		t, e := d.Token()
		if e != nil {
			return e
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				key, ok := k.(string)
				if !ok || seen[strings.ToLower(key)] {
					return errors.New("direct_json_duplicate")
				}
				seen[strings.ToLower(key)] = true
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		default:
			return errors.New("direct_json_delimiter")
		}
		_, e = d.Token()
		return e
	}
	if e := walk(0); e != nil {
		return errors.New("direct_json_invalid")
	}
	if _, e := d.Token(); e != io.EOF {
		return errors.New("direct_json_trailing")
	}
	var shape any
	if json.Unmarshal(raw, &shape) != nil || directTypedKeys(shape, reflect.TypeOf(target)) != nil {
		return errors.New("direct_json_exact_shape")
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(target); e != nil {
		return errors.New("direct_json_shape")
	}
	return nil
}

// encoding/json accepts case aliases for Go structs. The research protocol does
// not: every declared JSON field is explicit, including false, null and zero.
func directTypedKeys(value any, typ reflect.Type) error {
	if typ == reflect.TypeOf(json.RawMessage{}) {
		return nil
	}
	if typ.Kind() == reflect.Pointer {
		if value == nil {
			return nil
		}
		return directTypedKeys(value, typ.Elem())
	}
	switch typ.Kind() {
	case reflect.Struct:
		m, ok := value.(map[string]any)
		if !ok {
			return errors.New("object_required")
		}
		keys := map[string]reflect.Type{}
		optional := map[string]bool{}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")[0]
			if tag == "-" {
				continue
			}
			if tag == "" {
				tag = f.Name
			}
			keys[tag] = f.Type
			optional[tag] = strings.Contains(f.Tag.Get("json"), ",omitempty")
		}
		for key := range m {
			if _, ok := keys[key]; !ok {
				return errors.New("unknown_exact_key")
			}
		}
		for key, t := range keys {
			v, ok := m[key]
			if !ok {
				if optional[key] {
					continue
				}
				return errors.New("exact_key_required")
			}
			if e := directTypedKeys(v, t); e != nil {
				return e
			}
		}
	case reflect.Slice:
		if value == nil {
			return nil
		}
		items, ok := value.([]any)
		if !ok {
			return errors.New("array_required")
		}
		for _, v := range items {
			if e := directTypedKeys(v, typ.Elem()); e != nil {
				return e
			}
		}
	case reflect.Bool:
		if _, ok := value.(bool); !ok {
			return errors.New("boolean_required")
		}
	case reflect.String:
		if _, ok := value.(string); !ok {
			return errors.New("string_required")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, ok := value.(float64)
		if !ok || math.Trunc(n) != n {
			return errors.New("integer_required")
		}
	case reflect.Float32, reflect.Float64:
		if _, ok := value.(float64); !ok {
			return errors.New("number_required")
		}
	case reflect.Map:
		if value == nil {
			return nil
		}
		m, ok := value.(map[string]any)
		if !ok {
			return errors.New("map_required")
		}
		for _, v := range m {
			if e := directTypedKeys(v, typ.Elem()); e != nil {
				return e
			}
		}
	}
	return nil
}

func directKeys(m map[string]json.RawMessage, keys []string) bool {
	if len(m) != len(keys) {
		return false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}
func directShape(raw json.RawMessage, keys []string) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || !directKeys(m, keys) {
		return nil, errors.New("direct_required_keys")
	}
	return m, nil
}
func DecodeDirectReading(raw []byte) (DirectReading, error) {
	var out DirectReading
	if e := DirectJSON(raw, &out); e != nil {
		return out, e
	}
	top, e := directShape(raw, []string{"header", "rows"})
	if e != nil {
		return out, e
	}
	header, e := directShape(top["header"], DirectHeaderFields)
	if e != nil {
		return out, e
	}
	var rows []json.RawMessage
	if json.Unmarshal(top["rows"], &rows) != nil || len(rows) < 1 || len(rows) > 64 {
		return out, errors.New("direct_rows_limit")
	}
	fields := []map[string]json.RawMessage{header}
	quoteRaw := []json.RawMessage{}
	for _, r := range rows {
		m, e := directShape(r, []string{"anchor", "fields", "issues"})
		if e != nil {
			return out, e
		}
		f, e := directShape(m["fields"], DirectRowFields)
		if e != nil {
			return out, e
		}
		fields = append(fields, f)
		var issues []string
		if string(m["issues"]) == "null" || json.Unmarshal(m["issues"], &issues) != nil || len(issues) > 32 {
			return out, errors.New("direct_issues")
		}
		for _, s := range issues {
			if len(s) > 256 {
				return out, errors.New("direct_issue_limit")
			}
		}
		quoteRaw = append(quoteRaw, m["anchor"])
	}
	for _, m := range fields {
		for _, raw := range m {
			if string(raw) == "null" {
				continue
			}
			f, e := directShape(raw, []string{"value", "evidence"})
			if e != nil {
				return out, e
			}
			var value string
			if json.Unmarshal(f["value"], &value) != nil || len(value) == 0 || len(value) > 4000 || DirectNormalize(value) == "" {
				return out, errors.New("direct_value_limit")
			}
			quoteRaw = append(quoteRaw, f["evidence"])
		}
	}
	for _, raw := range quoteRaw {
		var qs []json.RawMessage
		if json.Unmarshal(raw, &qs) != nil || len(qs) < 1 || len(qs) > 8 {
			return out, errors.New("direct_evidence_limit")
		}
		for _, q := range qs {
			m, e := directShape(q, []string{"page", "quote"})
			if e != nil {
				return out, e
			}
			var p int
			var text string
			if json.Unmarshal(m["page"], &p) != nil || p < 1 || p > 4 || json.Unmarshal(m["quote"], &text) != nil || len(text) > 4000 || DirectNormalize(text) == "" {
				return out, errors.New("direct_quote_invalid")
			}
		}
	}
	return out, nil
}

func DirectSchema() json.RawMessage {
	object := func(p map[string]any) map[string]any {
		keys := []string{}
		for k := range p {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return map[string]any{"type": "object", "properties": p, "required": keys, "additionalProperties": false}
	}
	quotes := map[string]any{"type": "array", "minItems": 1, "maxItems": 8, "items": object(map[string]any{"page": map[string]any{"type": "integer", "minimum": 1, "maximum": 4}, "quote": map[string]any{"type": "string", "minLength": 1, "maxLength": 4000}})}
	field := map[string]any{"anyOf": []any{object(map[string]any{"value": map[string]any{"type": "string", "minLength": 1, "maxLength": 4000}, "evidence": quotes}), map[string]any{"type": "null"}}}
	fields := func(keys []string) any {
		m := map[string]any{}
		for _, k := range keys {
			m[k] = map[string]any{"$ref": "#/$defs/field"}
		}
		return object(m)
	}
	schema := object(map[string]any{"header": fields(DirectHeaderFields), "rows": map[string]any{"type": "array", "minItems": 1, "maxItems": 64, "items": object(map[string]any{"anchor": quotes, "fields": fields(DirectRowFields), "issues": map[string]any{"type": "array", "maxItems": 32, "items": map[string]any{"type": "string", "maxLength": 256}}})}})
	schema["$defs"] = map[string]any{"field": field}
	raw, _ := json.Marshal(schema)
	return raw
}

type DirectBindingCheck struct {
	Field    string      `json:"field"`
	Row      int         `json:"row"`
	Passed   bool        `json:"passed"`
	Reason   string      `json:"reason"`
	Evidence []PageQuote `json:"evidence"`
}
type DirectBinding struct {
	Passed   bool                   `json:"sourceBindingPassed"`
	Proposal statementdata.Proposal `json:"proposal"`
	Review   statementdata.Review   `json:"review"`
	Checks   []DirectBindingCheck   `json:"checks"`
}

type directRange struct{ lo, hi int }

func directFind(d statementdata.Document, page int, text string, lo, hi int) []directRange {
	target := DirectNormalize(text)
	out := []directRange{}
	for i := lo; i < hi; i++ {
		if d.Spans[i].Page != page {
			continue
		}
		s := ""
		for j := i; j < hi && d.Spans[j].Page == page; j++ {
			if s != "" {
				s += " "
			}
			s += d.Spans[j].Text
			n := DirectNormalize(s)
			if n == target {
				out = append(out, directRange{i, j + 1})
				break
			}
			if len(n) >= len(target) {
				break
			}
		}
	}
	return out
}
func directQuote(d statementdata.Document, q PageQuote, lo, hi int) (directRange, error) {
	found := directFind(d, q.Page, q.Quote, lo, hi)
	if len(found) != 1 {
		return directRange{}, errors.New("quote_not_unique")
	}
	return found[0], nil
}

// BindDirectReading never sees a golden. Only literal, uniquely scoped evidence
// can select source IDs. Unclaimed spans remain unclaimed; no service repair.
func BindDirectReading(d statementdata.Document, reading DirectReading) (DirectBinding, error) {
	out := DirectBinding{Passed: true, Checks: []DirectBindingCheck{}, Proposal: statementdata.Proposal{Rows: []statementdata.RowMapping{}, Header: statementdata.HeaderMapping{Labels: map[string]statementdata.Selection{}}, Service: []statementdata.ServiceMapping{}}}
	if e := d.Validate(); e != nil {
		return out, e
	}
	raw, _ := json.Marshal(reading)
	if _, e := DecodeDirectReading(raw); e != nil {
		return out, e
	}
	add := func(row int, key string, qs []PageQuote, e error) {
		c := DirectBindingCheck{Row: row, Field: key, Passed: e == nil, Evidence: qs}
		if e != nil {
			c.Reason = e.Error()
			out.Passed = false
		}
		out.Checks = append(out.Checks, c)
	}
	anchors := make([]int, len(reading.Rows)+1)
	anchors[len(reading.Rows)] = len(d.Spans)
	for i, row := range reading.Rows {
		pos := -1
		end := -1
		var err error
		for _, q := range row.Anchor {
			r, e := directQuote(d, q, 0, len(d.Spans))
			if e != nil {
				err = e
				break
			}
			if end > r.lo {
				err = errors.New("anchor_order")
				break
			}
			if pos < 0 {
				pos = r.lo
			}
			end = r.hi
		}
		if pos < 0 || (i > 0 && pos <= anchors[i-1]) {
			err = errors.New("anchor_order")
		}
		anchors[i] = pos
		add(i+1, "anchor", row.Anchor, err)
	}
	// A bad anchor cannot define a permissive global row scope.
	for _, a := range anchors {
		if a < 0 {
			return out, nil
		}
	}
	bind := func(row int, key string, f *DirectField, lo, hi int) (statementdata.Selection, statementdata.Selection) {
		if f == nil {
			return nil, nil
		}
		selected := statementdata.Selection{}
		all := statementdata.Selection{}
		parts := []string{}
		last := -1
		var err error
		for _, q := range f.Evidence {
			r, e := directQuote(d, q, lo, hi)
			if e != nil && strings.HasSuffix(key, "Currency") && len(directFind(d, q.Page, q.Quote, lo, hi)) == 0 {
				r, e = directQuote(d, q, 0, len(d.Spans))
			}
			if e != nil {
				err = e
				break
			}
			if r.lo < last {
				err = errors.New("evidence_order")
				break
			}
			last = r.hi
			for _, s := range d.Spans[r.lo:r.hi] {
				all = append(all, s.ID)
			}
			if key == "description" {
				for _, s := range d.Spans[r.lo:r.hi] {
					selected = append(selected, s.ID)
				}
				parts = append(parts, DirectNormalize(q.Quote))
				continue
			}
			if key == "direction" {
				continue
			}
			found := directFind(d, q.Page, f.Value, r.lo, r.hi)
			if len(found) != 1 {
				err = errors.New("value_not_unique_in_quote")
				break
			}
			for _, s := range d.Spans[found[0].lo:found[0].hi] {
				selected = append(selected, s.ID)
			}
		}
		if err == nil && key == "description" && strings.Join(parts, " ") != DirectNormalize(f.Value) {
			err = errors.New("description_value_conflict")
		}
		if key != "description" && key != "direction" && len(f.Evidence) != 1 {
			err = errors.New("scalar_segments")
		}
		if key == "direction" && err == nil {
			// Direction is interpreted by Go from the printed signed account amount.
			if len(f.Evidence) != 1 {
				err = errors.New("direction_segments")
			} else {
				amount := reading.Rows[row-1].Fields["accountAmount"]
				literal := ""
				if amount != nil {
					literal = amount.Value
				}
				_, dir, e := statementdata.ParseMoney(literal, "RUB")
				if e != nil || dir != f.Value || dir == "" {
					err = errors.New("direction_conflict")
				}
			}
			selected = all
		}
		add(row, key, f.Evidence, err)
		if err != nil {
			return nil, nil
		}
		return selected, all
	}
	for i, row := range reading.Rows {
		m := map[string]statementdata.Selection{}
		for _, k := range DirectRowFields {
			lo, hi := anchors[i], anchors[i+1]

			m[k], _ = bind(i+1, k, row.Fields[k], lo, hi)
		}
		if row.Fields["direction"] != nil {
			selected := map[string]bool{}
			for _, id := range m["direction"] {
				selected[id] = true
			}
			covered := len(m["accountAmount"]) > 0
			for _, id := range m["accountAmount"] {
				covered = covered && selected[id]
			}
			if !covered {
				add(i+1, "direction", row.Fields["direction"].Evidence, errors.New("direction_not_account_evidence"))
			}
		}
		// Nulls are literal absence claims, assessed separately by recognition.
		out.Proposal.Rows = append(out.Proposal.Rows, statementdata.RowMapping{OperationDate: m["operationDate"], BookingDate: m["bookingDate"], AccountAmount: m["accountAmount"], AccountCurrency: m["accountCurrency"], TransactionAmount: m["transactionAmount"], TransactionCurrency: m["transactionCurrency"], Description: m["description"]})
	}
	h := map[string]statementdata.Selection{}
	for _, k := range DirectHeaderFields {
		ids, all := bind(0, k, reading.Header[k], 0, len(d.Spans))
		h[k] = ids
		if strings.Contains(k, "Balance") || strings.HasSuffix(k, "Turnover") {
			values := map[string]bool{}
			for _, id := range ids {
				values[id] = true
			}
			for _, id := range all {
				if !values[id] {
					for _, s := range d.Spans {
						if s.ID == id {
							if _, e := statementdata.Currency(s.Text); e != nil {
								out.Proposal.Header.Labels[k] = append(out.Proposal.Header.Labels[k], id)
							}
							break
						}
					}
				}
			}
		}
	}
	out.Proposal.Header.OpeningBookDate = h["openingBookDate"]
	out.Proposal.Header.ClosingBookDate = h["closingBookDate"]
	out.Proposal.Header.OpeningBookBalance = h["openingBookBalance"]
	out.Proposal.Header.ClosingBookBalance = h["closingBookBalance"]
	out.Proposal.Header.AvailableBalanceObserved = h["availableBalanceObserved"]
	out.Proposal.Header.DebitTurnover = h["debitTurnover"]
	out.Proposal.Header.CreditTurnover = h["creditTurnover"]
	out.Proposal.Header.Currency = h["currency"]
	review, e := statementdata.ValidateProposal(d, out.Proposal, nil, false, "")
	if e != nil {
		return out, e
	}
	out.Review = review
	for i, row := range review.Rows {
		for _, issue := range row.Issues {
			add(i+1, "existing_guard", nil, fmt.Errorf("%s", issue))
		}
		if f := reading.Rows[i].Fields["direction"]; f != nil && f.Value != row.Direction {
			add(i+1, "direction", f.Evidence, errors.New("source_direction_conflict"))
		}
	}
	// Existing role/grounding contradictions fail binding; owner/coverage blocks
	// remain visible without being reinterpreted as recognition failures.
	for _, b := range review.Reconciliation.Blockers {
		if b == "owner_page_review_required" || b == "owner_field_role_review_required" || b == "source_coverage_incomplete" {
			continue
		}
		if strings.Contains(b, ":") || strings.Contains(b, "book") || strings.Contains(b, "turnover_role") || strings.Contains(b, "ground") || strings.Contains(b, "Label") || strings.Contains(b, "field_label") || strings.Contains(b, "available_is_not") {
			add(0, "existing_guard", nil, fmt.Errorf("%s", b))
		}
	}
	return out, nil
}
