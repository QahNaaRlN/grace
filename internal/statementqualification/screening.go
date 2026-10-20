package statementqualification

import (
	"context"
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"time"
)

const ScreeningReportVersion = "statement.screening.v1.gemini31-ru-roles"
const ScreeningScorerVersion = "statement-screening.v1.19-20-23"
const ScreeningMaxMicros int64 = 161700

func ScreeningNames() []string { return []string{"columns", "cards", "turnover-roles-ru"} }
func ScreeningProfile() Profile {
	p := CurrentProfile()
	p.Scorer = ScreeningScorerVersion
	p.Calls = 3
	p.CeilingMicros = ScreeningMaxMicros
	p.RunSeconds = 420
	return p
}
func RunScreening(ctx context.Context, samples []Sample, r statementextract.Reasoner, m Meter) (Report, error) {
	return run(ctx, samples, r, m, true)
}

// ValidateScreeningReport cannot authorize production activation. It accepts a
// complete screening result only, never an incomplete campaign or unknown bill.
func ValidateScreeningReport(r Report, now time.Time) error {
	fail := errors.New("invalid_screening_report")
	if r.Version != ScreeningReportVersion || r.Profile != ScreeningProfile() || !statementextract.ScreeningPolicyValid(r.Policy) || r.Digest != reportDigest(r) || !r.Passed || r.Error != "" || r.Time.After(now) || now.Sub(r.Time) > 15*time.Minute || len(r.Corpus) != 3 || len(r.Results) != 3 || len(r.Traces) != 3 || len(r.Calls) != 3 || r.MaximumMicros != ScreeningMaxMicros || r.ActualMicros == nil {
		return fail
	}
	total := int64(0)
	ids := map[string]bool{}
	for _, c := range r.Calls {
		if !validBill(&statementextract.Billing{GenerationID: c.GenerationID, CostMicros: c.CostMicros, InputTokens: c.InputTokens, OutputTokens: c.OutputTokens}, r.Policy) || c.OutputTokens == 0 || ids[c.GenerationID] {
			return fail
		}
		ids[c.GenerationID] = true
		total += c.CostMicros
	}
	if total != *r.ActualMicros || total > ScreeningMaxMicros {
		return fail
	}
	for i, name := range ScreeningNames() {
		c, s, t := r.Corpus[i], r.Results[i], r.Traces[i]
		checks := []int{19, 20, 23}[i]
		if c.Name != name || c.FixtureHash != FixtureHash(name) || !validHash(c.DocumentHash) || !validHash(c.PlanHash) || c.InputBytes <= 0 || c.InputBytes > statementextract.MaxInputBytes || s.Name != name || s.Rows != 2 || !s.Passed || s.Checks != checks || s.Correct != checks || len(s.Codes) != 0 || len(s.Validation) != 0 || len(s.Coverage) != 0 || t.Name != name || len(t.Parts) != 1 {
			return fail
		}
	}
	return nil
}
func ScoreScreening(name string, r statementdata.Review) Result {
	if name != "turnover-roles-ru" {
		return Score(name, r)
	}
	out := Result{Name: name, Rows: len(r.Rows), Codes: []string{}, Validation: r.Reconciliation.Blockers, Coverage: r.Reconciliation.RowCoverage.Reasons}
	check := func(ok bool, code string) {
		out.Checks++
		if ok {
			out.Correct++
		} else {
			out.Codes = append(out.Codes, code)
		}
	}
	money := func(m *statementdata.Money, n int64) bool {
		return m != nil && m.AmountMinor == n && m.Currency == "RUB"
	}
	check(len(r.Rows) == 2, "row_count")
	for i, row := range r.Rows {
		if i >= 2 {
			break
		}
		check(money(row.AccountAmount, []int64{248600, 310500}[i]), "account_amount_currency")
		check(row.Direction == []string{"debit", "credit"}[i], "direction")
		check(row.OperationDate != nil && row.OperationDate.Local == []string{"2026-10-12", "2026-10-13"}[i] && row.OperationDate.UTC == "", "operation_clock")
		check(row.BookingDate == nil, "absent_booking")
		check(row.TransactionAmount == nil, "absent_fx")
		check(row.Description == []string{"ТЕСТОВЫЙ МАГАЗИН - ПОКУПКА", "ТЕСТОВЫЙ ПЕРЕВОД - ПРОВЕРИТЬ СМЫСЛ"}[i], "record_description")
		check(len(row.Issues) == 0, "row_issues")
	}
	check(money(r.Header.DebitTurnover, 248600), "printed_debit_turnover")
	check(money(r.Header.CreditTurnover, 310500), "printed_credit_turnover")
	check(money(r.Header.AvailableBalanceObserved, 0), "available_balance")
	check(r.Header.OpeningBookBalance == nil && r.Header.ClosingBookBalance == nil && r.Reconciliation.BookBalance.Status == "not_provided", "absent_book_balances")
	check(r.Header.OpeningBookDate == nil && r.Header.ClosingBookDate == nil, "absent_book_dates")
	check(r.Reconciliation.Turnover.Status == "matched", "turnover")
	check(r.Reconciliation.RowCoverage.Status == "matched", "coverage")
	check(len(r.Reconciliation.Blockers) == 0, "blockers")
	out.Passed = out.Checks == 23 && out.Checks == out.Correct
	return out
}
