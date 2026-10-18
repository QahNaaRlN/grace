package statementdata

import "time"

func sourceDay(d *LocalDate) string {
	if d == nil || len(d.Local) < 10 {
		return ""
	}
	day := d.Local[:10]
	if _, e := time.Parse("2006-01-02", day); e != nil {
		return ""
	}
	return day
}

// Reconcile compares source rows and source header fields only. It does not
// compare Grace's historical balance or infer missing book balances.
func Reconcile(rows []Row, h Header) Reconciliation {
	out := Reconciliation{FinancialCoverage: Check{"not_evaluated", []string{"no_financial_confirmation"}}, Turnover: Check{"not_provided", []string{}}, BookBalance: Check{"not_provided", []string{}}, RowCoverage: Check{"not_reviewed", []string{"owner_page_review_required"}}, LedgerBalance: Check{"not_evaluated", []string{"historical_anchor_not_proven"}}, Blockers: []string{}, Warnings: []string{}}
	debit, credit := int64(0), int64(0)
	valid := len(rows) > 0
	for _, r := range rows {
		if r.AccountAmount == nil || r.AccountAmount.Currency != "RUB" || r.AccountAmount.AmountMinor <= 0 || len(r.Issues) > 0 {
			valid = false
			continue
		}
		var ok bool
		switch r.Direction {
		case "debit":
			debit, ok = checkedAdd(debit, r.AccountAmount.AmountMinor)
		case "credit":
			credit, ok = checkedAdd(credit, r.AccountAmount.AmountMinor)
		default:
			valid = false
			continue
		}
		if !ok {
			valid = false
			out.Blockers = append(out.Blockers, "aggregate_overflow")
		}
	}
	if valid {
		out.DebitMinor = &debit
		out.CreditMinor = &credit
	} else {
		out.Blockers = append(out.Blockers, "invalid_source_rows")
	}
	if h.DebitTurnover != nil || h.CreditTurnover != nil {
		out.Turnover.Status = "matched"
		if !valid {
			out.Turnover.Status = "incomplete"
		} else {
			for _, v := range []struct {
				actual   int64
				expected *Money
			}{{debit, h.DebitTurnover}, {credit, h.CreditTurnover}} {
				if v.expected == nil {
					if out.Turnover.Status != "mismatch" {
						out.Turnover.Status = "incomplete"
					}
					continue
				}
				if v.expected.Currency != "RUB" || v.expected.AmountMinor != v.actual {
					out.Turnover.Status = "mismatch"
					out.Turnover.Reasons = append(out.Turnover.Reasons, "printed_turnover_mismatch")
					out.Blockers = append(out.Blockers, "printed_turnover_mismatch")
				}
			}
		}
	}
	if h.OpeningBookBalance != nil || h.ClosingBookBalance != nil {
		out.BookBalance.Status = "incomplete"
		periodValid := sourceDay(h.OpeningBookDate) != "" && sourceDay(h.ClosingBookDate) != "" && sourceDay(h.OpeningBookDate) <= sourceDay(h.ClosingBookDate)
		if periodValid {
			for _, row := range rows {
				date := row.BookingDate
				if date == nil {
					date = row.OperationDate
				}
				if sourceDay(date) == "" || sourceDay(date) < sourceDay(h.OpeningBookDate) || sourceDay(date) > sourceDay(h.ClosingBookDate) {
					periodValid = false
				}
			}
		}
		if !periodValid {
			out.BookBalance.Reasons = append(out.BookBalance.Reasons, "book_balance_period_not_proven")
		}
		if valid && periodValid && h.OpeningBookBalance != nil && h.ClosingBookBalance != nil {
			balance, ok := checkedAdd(h.OpeningBookBalance.AmountMinor, credit)
			if ok {
				balance, ok = checkedAdd(balance, -debit)
			}
			if !ok {
				out.Blockers = append(out.Blockers, "aggregate_overflow")
			} else if h.OpeningBookBalance.Currency != "RUB" || h.ClosingBookBalance.Currency != "RUB" || balance != h.ClosingBookBalance.AmountMinor {
				out.BookBalance.Status = "mismatch"
				out.BookBalance.Reasons = append(out.BookBalance.Reasons, "book_balance_mismatch")
				out.Blockers = append(out.Blockers, "book_balance_mismatch")
			} else {
				out.BookBalance.Status = "matched"
			}
		}
	}
	if out.Turnover.Status == "not_provided" || out.Turnover.Status == "incomplete" {
		out.Warnings = append(out.Warnings, "turnover_not_checked")
	}
	if out.BookBalance.Status == "not_provided" || out.BookBalance.Status == "incomplete" {
		out.Warnings = append(out.Warnings, "book_balance_not_checked")
	}
	return out
}
