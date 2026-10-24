package documentqualification

import (
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"os"
	"path/filepath"
)

func DecodeCandidate(raw []byte) (*assistant.GroupMembership, error) {
	var p *assistant.GroupMembership
	if len(raw) > 16384 || artifactJSON(raw, &p) != nil {
		return nil, errors.New("candidate_invalid")
	}
	return p, nil
}
func WriteBundle(dir string, c Corpus, m Manifest) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || m.Validate(c, m.Source, false) != nil {
		return errors.New("bundle_invalid")
	}
	if e := os.Mkdir(dir, 0755); e != nil {
		return errors.New("bundle_exists_or_invalid")
	}
	for name, v := range map[string]any{"corpus.json": c, "MANIFEST.json": m} {
		raw, e := json.Marshal(v)
		if e != nil {
			return errors.New("bundle_invalid")
		}
		f, e := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if e != nil {
			return errors.New("bundle_invalid")
		}
		_, e = f.Write(raw)
		closeErr := f.Close()
		if e != nil || closeErr != nil {
			return errors.New("bundle_invalid")
		}
	}
	return nil
}
func LoadBundle(dir string, source string) (Corpus, Manifest, error) {
	fail := errors.New("bundle_invalid")
	raw, e := readBounded(filepath.Join(dir, "corpus.json"), 8<<20)
	if e != nil {
		return Corpus{}, Manifest{}, fail
	}
	c, e := DecodeCorpus(raw)
	if e != nil {
		return c, Manifest{}, fail
	}
	raw, e = readBounded(filepath.Join(dir, "MANIFEST.json"), 65536)
	var m Manifest
	if e != nil || artifactJSON(raw, &m) != nil || m.Validate(c, source, false) != nil {
		return c, m, fail
	}
	return c, m, nil
}
func ValidateSummary(raw []byte, m Manifest) (Summary, error) {
	var s Summary
	fail := errors.New("summary_invalid")
	if len(raw) > 65536 || artifactJSON(raw, &s) != nil || s.Version != Version || s.Source != m.Source || s.ManifestSHA256 != m.Hash() || !hash64(s.ReportSHA256) || s.ImportReady || s.ActivationAuthorized || s.POST < 0 || s.POST > 8 || s.MetadataGET < 0 || s.MetadataGET > 6 || s.KnownCostMicros < 0 || s.KnownCostMicros > 8*MaximumObservedMicros || s.UnknownBills < 0 || s.UnknownBills > 8 || s.DurationMS < 0 || s.DurationMS > 1800000 || len(s.Cases) != 8 {
		return s, fail
	}
	enum := func(s string, allowed ...string) bool {
		for _, v := range allowed {
			if s == v {
				return true
			}
		}
		return false
	}
	if !enum(s.Decision, "PASS_SYNTHETIC", "REJECT", "INCONCLUSIVE") {
		return s, fail
	}
	if len(s.ProofSHA256) > 2 {
		return s, fail
	}
	for purpose, hash := range s.ProofSHA256 {
		if (purpose != "statement" && purpose != "receipt") || !hash64(hash) {
			return s, fail
		}
	}
	completed, observed, unknown, rejected := 0, 0, 0, 0
	overCap := false
	var total int64
	for i, v := range s.Cases {
		if v.Ordinal != i+1 || v.Case != IDs()[i] || !enum(v.State, "not_sent", "observed", "rejected", "completed") || !enum(v.Reason, "", "preflight", "pre_post_refusal", "billing_requires_review", "protocol_refusal", "quality_reject", "decisive_quality_reject", "reader_refusal") || !enum(v.Settlement, "not_reserved", "reserved", "unknown", "failed", "settled") || !enum(v.NativeBinding, "", "not_applicable", "pass", "fail", "image_review_required") || v.Checks < 0 || v.Checks > 400 || v.FailedChecks < 0 || v.FailedChecks > v.Checks || v.ActualMicros != nil && (*v.ActualMicros < 0 || *v.ActualMicros > MaximumObservedMicros) {
			return s, fail
		}
		if (v.InputTokens == nil) != (v.OutputTokens == nil) || v.InputTokens != nil && (*v.InputTokens < 0 || *v.InputTokens > 1000000000 || *v.OutputTokens < 0 || *v.OutputTokens > 1000000000) || v.DurationMS != nil && (*v.DurationMS < 0 || *v.DurationMS > 185000) {
			return s, fail
		}
		if (v.State != "not_sent") != (v.DurationMS != nil) {
			return s, fail
		}
		if len(v.CheckResults) != v.Checks || len(v.LegacyCheckResults) > 32 {
			return s, fail
		}
		failed := 0
		for _, check := range append(append([]Check{}, v.CheckResults...), v.LegacyCheckResults...) {
			if check.Row < 0 || check.Row > 200 || !knownCheckCode(check.Code) {
				return s, fail
			}
		}
		for _, check := range v.CheckResults {
			if !check.Passed {
				failed++
			}
		}
		if failed != v.FailedChecks {
			return s, fail
		}
		if v.ActualMicros != nil {
			total += *v.ActualMicros
			if *v.ActualMicros > MaximumCallMicros {
				overCap = true
			}
		} else if v.State != "not_sent" {
			unknown++
		}
		if v.State == "rejected" {
			rejected++
		}
		if v.State != "not_sent" {
			observed++
		}

		if v.State == "completed" {
			completed++
			if v.NativeBinding == "fail" || v.InputTokens == nil || *v.InputTokens < 1 || *v.InputTokens > 30000 || *v.OutputTokens < 1 || *v.OutputTokens > 16000 || !v.Recognition || v.Settlement != "settled" || v.ActualMicros == nil || v.FailedChecks != 0 {
				return s, fail
			}
		}
	}
	if total > MaximumMicros {
		overCap = true
	}
	if overCap != s.OverCap || s.Decision == "REJECT" && rejected != 1 || s.Decision == "INCONCLUSIVE" && completed == 8 {
		return s, fail
	}
	if observed != s.POST || unknown != s.UnknownBills || total != s.KnownCostMicros || s.Decision == "PASS_SYNTHETIC" && (s.OverCap || completed != 8 || s.POST != 8 || s.UnknownBills != 0 || s.MetadataGET != 6) {
		return s, fail
	}
	return s, nil
}

func knownCheckCode(code string) bool {
	for _, v := range []string{"operationDate", "bookingDate", "accountAmount", "accountCurrency", "direction", "transactionAmount", "transactionCurrency", "description", "openingBookDate", "closingBookDate", "openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover", "currency", "rowCount", "issues", "protocol", "measurement", "golden", "receipt_complete", "receipt_unknown", "seller", "issuedDate", "issuedAt", "total", "fiscalDocumentNumber", "item_count", "name", "quantity", "unitPrice", "lineTotal", "discount", "brand", "package", "broadGroup", "row_count", "account_amount_currency", "operation_clock", "booking_clock", "record_description", "row_issues", "fx_column", "turnover", "coverage", "blockers", "absent_book_balances", "available_balance", "book_balances", "opening_book_value", "closing_book_value", "absent_booking", "absent_fx", "printed_debit_turnover", "printed_credit_turnover", "absent_book_dates"} {
		if v == code {
			return true
		}
	}
	return false
}
