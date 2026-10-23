package statementextract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

func rowScopedWire(t *testing.T, v any) []byte {
	t.Helper()
	raw, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var shape any
	_ = json.Unmarshal(raw, &shape)
	var fix func(any)
	fix = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, z := range x {
				if z == nil {
					if k == "labels" {
						x[k] = map[string]any{}
					} else {
						x[k] = []any{}
					}
				} else {
					fix(z)
				}
			}
		case []any:
			for _, z := range x {
				fix(z)
			}
		}
	}
	fix(shape)
	header := func(h map[string]any) {
		labels := h["labels"].(map[string]any)
		for _, k := range []string{"openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover"} {
			if _, ok := labels[k]; !ok {
				labels[k] = []any{}
			}
		}
	}
	if m, ok := shape.(map[string]any); ok {
		if h, ok := m["header"].(map[string]any); ok {
			header(h)
		}
	}
	raw, e = json.Marshal(shape)
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func rowScopedFixture(t *testing.T) (statementdata.Document, RowGrouping, statementdata.RowMapping) {
	t.Helper()
	d := testDocument("19.10.2026", "-1.00", "RUB", "TEXT", "21.10.2026", "+1.00", "OTHER")
	g := RowGrouping{Rows: []RowEnvelope{{[]string{"s0", "s1", "s3"}}, {[]string{"s4", "s5", "s6"}}}, Context: []string{"s2"}, Header: statementdata.HeaderMapping{Currency: []string{"s2"}}, Service: []statementdata.ServiceMapping{}, Unresolved: []string{}}
	decoded, e := DecodeRowGrouping(d, rowScopedWire(t, g))
	if e != nil {
		t.Fatal(e)
	}
	row := statementdata.RowMapping{OperationDate: []string{"s0"}, AccountAmount: []string{"s1"}, AccountCurrency: []string{"s2"}, Description: []string{"s3"}}
	r, e := DecodeRowFields(d, decoded, 1, rowScopedWire(t, row))
	if e != nil {
		t.Fatal(e)
	}
	return d, decoded, r
}
func TestRowScopedBuildersAndIsolation(t *testing.T) {
	d, g, r := rowScopedFixture(t)
	before := d.Hash()
	group, e := BuildRowGrouping(d)
	if e != nil {
		t.Fatal(e)
	}
	q, e := BuildRowFields(d, g, 1)
	if e != nil {
		t.Fatal(e)
	}
	if group.Version != RowScopedVersion || q.Stage != "row" || q.RowOrdinal != 1 || q.PromptSHA != RowFieldPromptSHA || group.PromptSHA != RowGroupingPromptSHA || q.Hash == group.Hash {
		t.Fatal("identity")
	}
	for _, v := range []RowScopedRequest{group, q} {
		raw, _ := json.Marshal(v.Messages)
		if len(raw)+len(v.Schema) != v.InputBytes || v.InputBytes > MaxInputBytes {
			t.Fatal("bytes")
		}
	}
	var body struct{ Context, Spans [][]any }
	if json.Unmarshal([]byte(q.Messages[1]["content"]), &body) != nil {
		t.Fatal("wire")
	}
	got := []string{}
	for _, s := range body.Spans {
		got = append(got, s[0].(string))
	}
	if !reflect.DeepEqual(got, []string{"s0", "s1", "s3"}) || len(body.Context) != 1 || body.Context[0][0] != "s2" {
		t.Fatal("neighbor delivered", body)
	}
	// Opaque hidden descriptions remain hidden, but their aliases are preserved.
	if strings.Contains(q.Messages[1]["content"], "OTHER") || strings.Contains(q.Messages[1]["content"], "TEXT") {
		t.Fatal("redaction changed")
	}
	var schema map[string]any
	_ = json.Unmarshal(q.Schema, &schema)
	defs := schema["$defs"].(map[string]any)
	if !reflect.DeepEqual(defs["ownedAlias"].(map[string]any)["enum"], []any{"s0", "s1", "s3"}) || !reflect.DeepEqual(defs["moneySelection"].(map[string]any)["items"].(map[string]any)["enum"], []any{"s1"}) {
		t.Fatal("schema outside owned", defs)
	}
	g2 := g
	r2 := r
	r2.OperationDate = []string{"s4"}
	r2.AccountAmount = []string{"s5"}
	r2.Description = []string{"s6"}
	out, e := ComposeRowScoped(d, g2, []statementdata.RowMapping{r, r2})
	if e != nil {
		t.Fatal(e)
	}
	out.Rows[0].Description[0] = "MUTATED"
	q.Schema[0] = 'X'
	q.Messages[1]["content"] = "MUTATED"
	again, e := BuildRowFields(d, g, 1)
	if e != nil || again.Hash == "" || again.Messages[1]["content"] == "MUTATED" || d.Hash() != before || r.Description[0] != "s3" {
		t.Fatal("shared memory")
	}
}
func TestRowScopedPartitionAndFieldRefusals(t *testing.T) {
	d, g, r := rowScopedFixture(t)
	for _, mode := range []string{"duplicate", "missing", "foreign", "row_order", "owned_order", "unresolved", "too_many", "empty_row", "header_outside", "label_unknown", "null"} {
		t.Run(mode, func(t *testing.T) {
			raw := rowScopedWire(t, g)
			var x RowGrouping
			_ = json.Unmarshal(raw, &x)
			switch mode {
			case "duplicate":
				x.Context = append(x.Context, "s0")
			case "missing":
				x.Rows[0].Spans = x.Rows[0].Spans[:2]
			case "foreign":
				x.Rows[0].Spans[0] = "forged"
			case "row_order":
				x.Rows[0], x.Rows[1] = x.Rows[1], x.Rows[0]
			case "owned_order":
				x.Rows[0].Spans[0], x.Rows[0].Spans[1] = x.Rows[0].Spans[1], x.Rows[0].Spans[0]
			case "unresolved":
				x.Unresolved = []string{"s3"}
				x.Rows[0].Spans = x.Rows[0].Spans[:2]
			case "too_many":
				x.Rows = append(x.Rows, x.Rows[0])
			case "empty_row":
				x.Rows[0].Spans = []string{}
			case "header_outside":
				x.Header.Currency = []string{"s0"}
			case "label_unknown":
				x.Header.Labels["invented"] = []string{}
			case "null":
				x.Context = nil
			}
			b, _ := json.Marshal(x)
			decoded, e := DecodeRowGrouping(d, b)
			if mode == "unresolved" {
				if e != nil {
					t.Fatal(e)
				}
				_, e = BuildRowFields(d, decoded, 1)
				if RowScopedCause(e) != "partition_unresolved" {
					t.Fatal(e)
				}
				return
			}
			if e == nil {
				t.Fatal("accepted", mode)
			}
		})
	}
	for _, mode := range []string{"missing_description", "neighbor", "context_description", "context_money", "two_currency", "duplicate_alias", "direction", "non_numeric"} {
		t.Run(mode, func(t *testing.T) {
			raw := rowScopedWire(t, r)
			var x statementdata.RowMapping
			_ = json.Unmarshal(raw, &x)
			switch mode {
			case "missing_description":
				x.Description = []string{}
			case "neighbor":
				x.Description = []string{"s6"}
			case "context_description":
				x.Description = append(x.Description, "s2")
			case "context_money":
				x.AccountAmount = append(x.AccountAmount, "s2")
			case "two_currency":
				x.AccountCurrency = append(x.AccountCurrency, "s0")
			case "duplicate_alias":
				x.Description = append(x.Description, "s3")
			case "direction":
				x.Direction = "income"
			case "non_numeric":
				x.AccountAmount = append(x.AccountAmount, "s3")
			}
			if _, e := DecodeRowFields(d, g, 1, rowScopedWire(t, x)); e == nil {
				t.Fatal("accepted", mode)
			}
		})
	}
	// Explicit context that no financial field uses is not automatic service.
	extra := testDocument("19.10.2026", "-1.00", "RUB", "TEXT", "21.10.2026", "+1.00", "OTHER", "unused")
	g.Context = append(g.Context, "s7")
	r2 := r
	r2.OperationDate = []string{"s4"}
	r2.AccountAmount = []string{"s5"}
	r2.Description = []string{"s6"}
	if _, e := ComposeRowScoped(extra, g, []statementdata.RowMapping{r, r2}); RowScopedCause(e) != "context_unused" {
		t.Fatal(e)
	}
}
func TestRowScopedStrictResponses(t *testing.T) {
	d, g, r := rowScopedFixture(t)
	raw := rowScopedWire(t, r)
	mutations := map[string][]byte{
		"missing":   bytes.Replace(raw, []byte(`"description":["s3"],`), nil, 1),
		"null":      bytes.Replace(raw, []byte(`"description":["s3"]`), []byte(`"description":null`), 1),
		"unknown":   append([]byte(`{"extra":[],`), raw[1:]...),
		"duplicate": append([]byte(`{"direction":"",`), raw[1:]...),
		"case":      bytes.Replace(raw, []byte(`"direction"`), []byte(`"Direction"`), 1),
		"BOM":       append([]byte{0xef, 0xbb, 0xbf}, raw...),
		"trailing":  append(append([]byte{}, raw...), []byte(`{}`)...),
		"oversize":  []byte(strings.Repeat(" ", RowScopedResponseLimit+1)),
		"utf8":      bytes.Replace(raw, []byte(`"direction":""`), []byte{'"', 'd', 'i', 'r', 'e', 'c', 't', 'i', 'o', 'n', '"', ':', '"', 0xff, '"'}, 1),
		"deep":      []byte(strings.Repeat("[", 34) + strings.Repeat("]", 34)),
	}
	for name, b := range mutations {
		t.Run(name, func(t *testing.T) {
			if _, e := DecodeRowFields(d, g, 1, b); RowScopedCause(e) != "response_invalid" {
				t.Fatal("not rejected", e)
			}
		})
	}
	group := rowScopedWire(t, g)
	for _, u := range []string{`\ud800`, `\udc00`, `\ud800x`, `\ud800\u1234`} {
		bad := bytes.Replace(group, []byte(`"service":[]`), []byte(fmt.Sprintf(`"service":[{"spans":[],"reason":"%s"}]`, u)), 1)
		if _, e := DecodeRowGrouping(d, bad); RowScopedCause(e) != "response_invalid" {
			t.Fatal("surrogate", e)
		}
	}
	if !rowScopedUnicode([]byte(`"\ud83d\ude00"`)) || !rowScopedUnicode([]byte(`"\\ud800"`)) {
		t.Fatal("valid Unicode escapes")
	}
	// Capture permits whitespace up to the exact limit, never silently truncates.
	padded := append(raw, bytes.Repeat([]byte(" "), RowScopedResponseLimit-len(raw))...)
	if _, e := DecodeRowFields(d, g, 1, padded); e != nil {
		t.Fatal(e)
	}
	if _, e := DecodeRowFields(d, g, 1, append(padded, ' ')); e == nil {
		t.Fatal("capture cap")
	}
}
func TestRowScopedRequestBounds(t *testing.T) {
	for _, n := range []int{2000, 2001} {
		xs := make([]string, n)
		for i := range xs {
			xs[i] = "RUB"
		}
		_, e := BuildRowGrouping(testDocument(xs...))
		if n == 2000 && e != nil || n == 2001 && e == nil {
			t.Fatal(n, e)
		}
	}
	c := Chunk{Index: 1, Context: []RedactedSpan{}, Spans: []RedactedSpan{{ID: "s0", Page: 1, X: 1, Y: 1, Text: "x", LexicalClass: ClassOther}}}
	schema := map[string]any{"type": "object"}
	p := ScreeningPolicy()
	q, e := rowScopedRequest("grouping", 0, p, c, "x", schema)
	if e != nil {
		t.Fatal(e)
	}
	c.Spans[0].Text = strings.Repeat("x", MaxInputBytes-q.InputBytes+1)
	q, e = rowScopedRequest("grouping", 0, p, c, "x", schema)
	if e != nil || q.InputBytes != MaxInputBytes {
		t.Fatal("exact input limit", q.InputBytes, e)
	}
	c.Spans[0].Text += "x"
	if _, e := rowScopedRequest("grouping", 0, p, c, "x", schema); RowScopedCause(e) != "request_limit_exceeded" {
		t.Fatal(e)
	}
}

func TestRowScopedContextAmountNeverBecomesRowMoney(t *testing.T) {
	_, g, r := rowScopedFixture(t)
	d := testDocument("19.10.2026", "-1.00", "RUB", "TEXT", "21.10.2026", "+1.00", "OTHER", "20.00")
	g.Context = append(g.Context, "s7")
	g.Header.AvailableBalanceObserved = []string{"s7"}
	request, e := BuildRowFields(d, g, 1)
	if e != nil {
		t.Fatal(e)
	}
	var schema map[string]any
	_ = json.Unmarshal(request.Schema, &schema)
	money := schema["$defs"].(map[string]any)["moneySelection"].(map[string]any)["items"].(map[string]any)["enum"]
	if !reflect.DeepEqual(money, []any{"s1"}) {
		t.Fatal("context numeric eligible for row amount", money)
	}
	r.AccountAmount = append(r.AccountAmount, "s7")
	if _, e := DecodeRowFields(d, g, 1, rowScopedWire(t, r)); RowScopedCause(e) != "row_subset_invalid" {
		t.Fatal("numeric context escaped subset guard", e)
	}
}

func TestRowScopedUnboundLabelDoesNotUseContext(t *testing.T) {
	_, g, r := rowScopedFixture(t)
	d := testDocument("19.10.2026", "-1.00", "RUB", "TEXT", "21.10.2026", "+1.00", "OTHER", "Opening")
	g.Context = append(g.Context, "s7")
	g.Header.Labels["openingBookBalance"] = []string{"s7"}
	other := r
	other.OperationDate = []string{"s4"}
	other.AccountAmount = []string{"s5"}
	other.Description = []string{"s6"}
	if _, e := ComposeRowScoped(d, g, []statementdata.RowMapping{r, other}); RowScopedCause(e) != "context_unused" {
		t.Fatal("unbound context hint counted as coverage", e)
	}
}
