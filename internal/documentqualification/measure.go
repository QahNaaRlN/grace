package documentqualification

import (
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/receiptdrafts"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"reflect"
	"strings"
)

type Check struct {
	Code   string `json:"code"`
	Row    int    `json:"row"`
	Passed bool   `json:"passed"`
}
type Measurement struct {
	Protocol              bool                                  `json:"protocol"`
	Recognition           bool                                  `json:"recognition"`
	NativeBinding         string                                `json:"nativeBinding"`
	ImageReviewRequired   bool                                  `json:"imageReviewRequired"`
	ImportReady           bool                                  `json:"importReady"`
	Checks                []Check                               `json:"checks"`
	LegacyChecks          []statementqualification.ScoringCheck `json:"legacyChecks"`
	LegacyReadinessPassed bool                                  `json:"legacyReadinessPassed"`
	Blockers              []string                              `json:"blockers"`
}

func Measure(c Case, out assistant.LLMResponse) Measurement {
	m := Measurement{NativeBinding: "not_applicable", ImageReviewRequired: true, Checks: []Check{}, Blockers: []string{"owner_review_required"}}
	if c.Golden != nil {
		var v statementdata.VisualReading
		if statementextract.DirectJSON(out.Content, &v) != nil || v.Validate(len(c.Document.Pages)) != nil {
			m.Checks = append(m.Checks, Check{"protocol", 0, false})
			return m
		}
		m.Protocol = true
		raw, _ := json.Marshal(v)
		var d statementextract.DirectReading
		_ = json.Unmarshal(raw, &d)
		name := c.ID
		if name == "cards-photo" {
			name = "cards-image-only"
		}
		if strings.HasPrefix(name, "full32") {
			name = "full32"
		}
		measured, e := statementqualification.MeasureDirect(statementqualification.DirectCase{Name: name, Document: c.Oracle, Golden: *c.Golden}, d)
		if e != nil {
			m.Checks = append(m.Checks, Check{"measurement", 0, false})
			return m
		}
		m.Recognition = measured.RecognitionPassed
		for _, check := range measured.Checks {
			m.Checks = append(m.Checks, Check{check.Field, check.Row, check.Passed})
		}
		m.LegacyChecks = measured.LegacyChecks
		if measured.LegacyResult != nil {
			m.LegacyReadinessPassed = measured.LegacyResult.Passed
		}
		if c.Native != nil {
			binding, e := statementextract.BindVisualReading(*c.Native, v)
			m.NativeBinding = "fail"
			if e == nil && binding.Passed {
				m.NativeBinding = "pass"
			}
			if e == nil {
				m.Blockers = append(m.Blockers, binding.Review.Reconciliation.Blockers...)
			}
		} else {
			m.NativeBinding = "image_review_required"
		}
		return m
	}
	receipt, e := assistant.DecodeDocumentReceipt(out)
	if e != nil {
		m.Checks = append(m.Checks, Check{"protocol", 0, false})
		return m
	}
	m.Protocol = true
	m.Recognition = true
	add := func(code string, row int, ok bool) {
		m.Checks = append(m.Checks, Check{code, row, ok})
		m.Recognition = m.Recognition && ok
	}
	var expected receiptdrafts.Fields
	if json.Unmarshal(c.ReceiptGolden, &expected) != nil {
		add("golden", 0, false)
		return m
	}
	got := receipt.Extracted.Fields
	if c.Readable {
		add("receipt_complete", 0, receipt.Complete && receipt.ReceiptCount == 1 && len(receipt.Issues) == 0)
	} else {
		add("receipt_unknown", 0, !receipt.Complete && len(receipt.Issues) > 0 && receipt.ReceiptCount <= 1)
	}
	compare := func(code string, row int, w, g *string, scale int) {
		ok := reflect.DeepEqual(w, g)
		if w != nil && g != nil {
			if scale >= 0 {
				a, e := receiptdrafts.ParseScaled(*w, scale)
				b, f := receiptdrafts.ParseScaled(*g, scale)
				ok = e == nil && f == nil && a == b
			} else {
				ok = strings.TrimSpace(*w) == strings.TrimSpace(*g)
			}
		}
		add(code, row, ok)
	}
	for _, field := range []struct {
		code  string
		w, g  *string
		scale int
	}{{"seller", expected.Seller, got.Seller, -1}, {"issuedDate", expected.IssuedDate, got.IssuedDate, -1}, {"issuedAt", expected.IssuedAt, got.IssuedAt, -1}, {"currency", expected.Currency, got.Currency, -1}, {"total", expected.Total, got.Total, 2}, {"fiscalDocumentNumber", expected.FiscalDocumentNumber, got.FiscalDocumentNumber, -1}} {
		compare(field.code, 0, field.w, field.g, field.scale)
	}
	add("item_count", 0, len(expected.Items) == len(got.Items))
	n := len(expected.Items)
	if len(got.Items) > n {
		n = len(got.Items)
	}
	for i := 0; i < n; i++ {
		var w, g receiptdrafts.Item
		if i < len(expected.Items) {
			w = expected.Items[i]
		}
		if i < len(got.Items) {
			g = got.Items[i]
		}
		for _, f := range []struct {
			code  string
			w, g  *string
			scale int
		}{{"name", w.Name, g.Name, -1}, {"quantity", w.Quantity, g.Quantity, 3}, {"unitPrice", w.UnitPrice, g.UnitPrice, 2}, {"lineTotal", w.LineTotal, g.LineTotal, 2}, {"discount", w.Discount, g.Discount, 2}, {"brand", w.Brand, g.Brand, -1}, {"package", w.Package, g.Package, -1}, {"broadGroup", w.BroadGroup, g.BroadGroup, -1}} {
			compare(f.code, i+1, f.w, f.g, f.scale)
		}
	}
	return m
}
