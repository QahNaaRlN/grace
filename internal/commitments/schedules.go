package commitments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/loans"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"net/http"
	"time"
)

type DraftPayment struct {
	SourceID    string `json:"sourceId"`
	DueOn       string `json:"dueOn"`
	AmountMinor int64  `json:"amountMinor"`
	PaidMinor   int64  `json:"paidMinor"`
	// SPEC-0053: optional split of the row; omitted when unknown so the base hash
	// of rows without parts stays the same.
	PrincipalMinor *int64  `json:"principalMinor,omitempty"`
	InterestMinor  *int64  `json:"interestMinor,omitempty"`
	OtherMinor     *int64  `json:"otherMinor,omitempty"`
	PartsSource    *string `json:"partsSource,omitempty"`
}
type ScheduleDraft struct {
	ID        string         `json:"id"`
	Payments  []DraftPayment `json:"payments"`
	Revision  int            `json:"revision"`
	AppliedAt *time.Time     `json:"appliedAt"`
}

func snapshot(ctx context.Context, tx pgx.Tx, id string) ([]DraftPayment, string, error) {
	rows, err := tx.Query(ctx, `SELECT sp.id::text,sp.due_on::text,sp.amount_minor,COALESCE((SELECT sum(s.amount_minor) FROM scheduled_payment_settlements s WHERE s.scheduled_payment_id=sp.id),0),
		sp.principal_minor,sp.interest_minor,sp.other_minor,sp.parts_source FROM scheduled_payments sp WHERE sp.obligation_id::text=$1 AND sp.plan_active ORDER BY sp.position FOR UPDATE OF sp`, id)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	payments := []DraftPayment{}
	for rows.Next() {
		var p DraftPayment
		if err = rows.Scan(&p.SourceID, &p.DueOn, &p.AmountMinor, &p.PaidMinor, &p.PrincipalMinor, &p.InterestMinor, &p.OtherMinor, &p.PartsSource); err != nil {
			return nil, "", err
		}
		payments = append(payments, p)
	}
	if err = rows.Err(); err != nil {
		return nil, "", err
	}
	data, _ := json.Marshal(payments)
	hash := sha256.Sum256(data)
	return payments, hex.EncodeToString(hash[:]), nil
}
func (h *Handler) CopySchedule(w http.ResponseWriter, r *http.Request) {
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	id := chi.URLParam(r, "obligationId")
	if err = lockLoan(r.Context(), tx, id); err != nil {
		failure(w, err)
		return
	}
	payments, hash, err := snapshot(r.Context(), tx, id)
	if err != nil {
		failure(w, err)
		return
	}
	draft := ScheduleDraft{Payments: payments, Revision: 1}
	err = tx.QueryRow(r.Context(), `INSERT INTO payment_schedule_drafts(obligation_id,base_hash,payments) VALUES($1,$2,$3) RETURNING id::text`, id, hash, payments).Scan(&draft.ID)
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 201, draft)
}

// normalizeParts keeps a complete split with its source ("bank" unless the owner
// kept Grace's calculation) and reports whether the split is valid.
func normalizeParts(p *DraftPayment) bool {
	if p.PrincipalMinor == nil && p.InterestMinor == nil && p.OtherMinor == nil {
		p.PartsSource = nil
		return true
	}
	if !loans.ValidParts(p.PrincipalMinor, p.InterestMinor, p.OtherMinor, p.AmountMinor) {
		return false
	}
	if p.PartsSource == nil || *p.PartsSource != loans.SourceCalculated {
		source := loans.SourceBank
		p.PartsSource = &source
	}
	return true
}

func validateSchedule(payments, base []DraftPayment) (int64, bool) {
	if len(payments) == 0 || len(payments) > 600 {
		return 0, false
	}
	locked := map[string]DraftPayment{}
	sources := map[string]bool{}
	for _, p := range base {
		if p.PaidMinor > 0 {
			locked[p.SourceID] = p
		}
		sources[p.SourceID] = true
	}
	seen := map[string]bool{}
	total := int64(0)
	last := ""
	for i := range payments {
		if !normalizeParts(&payments[i]) {
			return 0, false
		}
		p := payments[i]
		if !validDate(p.DueOn) || p.DueOn <= last || p.AmountMinor < 1 || p.AmountMinor > 9007199254740991-total || seen[p.SourceID] && p.SourceID != "" {
			return 0, false
		}
		if p.SourceID != "" && !sources[p.SourceID] {
			return 0, false
		}
		if old, ok := locked[p.SourceID]; ok {
			// Paid rows keep their date and amount; their parts may be added or fixed.
			if old.DueOn != p.DueOn || old.AmountMinor != p.AmountMinor || old.PaidMinor != p.PaidMinor {
				return 0, false
			}
			delete(locked, p.SourceID)
		} else if p.PaidMinor != 0 {
			return 0, false
		}
		seen[p.SourceID] = true
		total += p.AmountMinor
		last = p.DueOn
	}
	return total, len(locked) == 0
}
func (h *Handler) EditSchedule(w http.ResponseWriter, r *http.Request)  { h.changeSchedule(w, r, false) }
func (h *Handler) ApplySchedule(w http.ResponseWriter, r *http.Request) { h.changeSchedule(w, r, true) }
func (h *Handler) changeSchedule(w http.ResponseWriter, r *http.Request, apply bool) {
	var input struct {
		Payments []DraftPayment `json:"payments"`
		Revision int            `json:"revision"`
		Confirm  bool           `json:"confirm"`
	}
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	if apply && !input.Confirm {
		web.WriteError(w, 422, "confirmation_required", "Подтвердите замену неоплаченной части графика.")
		return
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	id := chi.URLParam(r, "obligationId")
	if err = lockLoan(r.Context(), tx, id); err != nil {
		failure(w, err)
		return
	}
	var hash string
	d := ScheduleDraft{ID: chi.URLParam(r, "draftId")}
	err = tx.QueryRow(r.Context(), `SELECT base_hash,payments,revision,applied_at FROM payment_schedule_drafts WHERE id::text=$1 AND obligation_id::text=$2 FOR UPDATE`, d.ID, id).Scan(&hash, &d.Payments, &d.Revision, &d.AppliedAt)
	if err != nil {
		failure(w, err)
		return
	}
	if apply && d.AppliedAt != nil {
		web.WriteJSON(w, 200, d)
		return
	}
	if d.AppliedAt != nil || d.Revision != input.Revision {
		failure(w, pgx.ErrNoRows)
		return
	}
	base, currentHash, err := snapshot(r.Context(), tx, id)
	if err != nil {
		failure(w, err)
		return
	}
	if hash != currentHash {
		web.WriteError(w, 409, "stale_schedule", "График или оплаты изменились. Создайте новую копию.")
		return
	}
	if !apply {
		d.Payments = input.Payments
	}
	total, ok := validateSchedule(d.Payments, base)
	if !ok {
		web.WriteError(w, 422, "invalid_schedule", "Проверьте даты и суммы. Оплаченные строки должны остаться без изменений.")
		return
	}
	if apply {
		_, err = tx.Exec(r.Context(), `UPDATE scheduled_payments sp SET plan_active=false WHERE obligation_id::text=$1 AND plan_active AND NOT EXISTS(SELECT 1 FROM scheduled_payment_settlements s WHERE s.scheduled_payment_id=sp.id)`, id)
		if err != nil {
			failure(w, err)
			return
		}
		// Move retained rows out of the position range before placing the new plan.
		_, err = tx.Exec(r.Context(), `UPDATE scheduled_payments SET position=position+1000000 WHERE obligation_id::text=$1 AND plan_active`, id)
		if err != nil {
			failure(w, err)
			return
		}
		for i, p := range d.Payments {
			if p.PaidMinor > 0 {
				_, err = tx.Exec(r.Context(), `UPDATE scheduled_payments SET position=$2,principal_minor=$3,interest_minor=$4,other_minor=$5,parts_source=$6 WHERE id::text=$1 AND plan_active`,
					p.SourceID, i+1, p.PrincipalMinor, p.InterestMinor, p.OtherMinor, p.PartsSource)
			} else {
				err = tx.QueryRow(r.Context(), `INSERT INTO scheduled_payments(obligation_id,position,due_on,amount_minor,principal_minor,interest_minor,other_minor,parts_source) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id::text`,
					id, i+1, p.DueOn, p.AmountMinor, p.PrincipalMinor, p.InterestMinor, p.OtherMinor, p.PartsSource).Scan(&d.Payments[i].SourceID)
			}
			if err != nil {
				failure(w, err)
				return
			}
		}
		_, err = tx.Exec(r.Context(), `UPDATE obligations SET total_minor=$2 WHERE id::text=$1`, id, total)
		if err != nil {
			failure(w, err)
			return
		}
		err = tx.QueryRow(r.Context(), `UPDATE payment_schedule_drafts SET applied_at=now(),revision=revision+1,payments=$2 WHERE id::text=$1 RETURNING applied_at,revision`, d.ID, d.Payments).Scan(&d.AppliedAt, &d.Revision)
	} else {
		err = tx.QueryRow(r.Context(), `UPDATE payment_schedule_drafts SET payments=$2,revision=revision+1 WHERE id::text=$1 RETURNING revision`, d.ID, d.Payments).Scan(&d.Revision)
	}
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 200, d)
}

// CalculationRow is the warning of one draft row after Grace's calculation.
type CalculationRow struct {
	Warning *string `json:"warning"`
}

// PartsCalculation is a draft with principal and interest calculated by the loan's
// repayment type (SPEC-0053). The schedule itself is not applied.
type PartsCalculation struct {
	Draft                    ScheduleDraft    `json:"draft"`
	Rows                     []CalculationRow `json:"rows"`
	PrincipalTotalMinor      int64            `json:"principalTotalMinor"`
	PrincipalDifferenceMinor int64            `json:"principalDifferenceMinor"`
}

func (h *Handler) CalculateParts(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Revision int `json:"revision"`
	}
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	id := chi.URLParam(r, "obligationId")
	if err = lockLoan(r.Context(), tx, id); err != nil {
		failure(w, err)
		return
	}
	var repaymentType string
	var principal *int64
	var rate *int
	var agreementOn *string
	err = tx.QueryRow(r.Context(), `SELECT repayment_type,principal_minor,annual_rate_basis_points,agreement_on::text FROM loan_details WHERE obligation_id::text=$1`, id).
		Scan(&repaymentType, &principal, &rate, &agreementOn)
	if err != nil && err != pgx.ErrNoRows {
		failure(w, err)
		return
	}
	missing := []string{}
	if repaymentType != loans.Annuity && repaymentType != loans.Differentiated {
		missing = append(missing, "repaymentType")
	}
	if principal == nil {
		missing = append(missing, "principalMinor")
	}
	if rate == nil {
		missing = append(missing, "annualRateBasisPoints")
	}
	if agreementOn == nil {
		missing = append(missing, "agreementOn")
	}
	if len(missing) > 0 {
		web.WriteJSON(w, 422, map[string]any{"code": "loan_details_required", "message": "Укажите вид погашения, сумму выдачи, ставку и дату договора.", "missing": missing})
		return
	}
	d := ScheduleDraft{ID: chi.URLParam(r, "draftId")}
	err = tx.QueryRow(r.Context(), `SELECT payments,revision,applied_at FROM payment_schedule_drafts WHERE id::text=$1 AND obligation_id::text=$2 FOR UPDATE`, d.ID, id).Scan(&d.Payments, &d.Revision, &d.AppliedAt)
	if err != nil {
		failure(w, err)
		return
	}
	if d.AppliedAt != nil || d.Revision != input.Revision {
		failure(w, pgx.ErrNoRows)
		return
	}
	start, _ := loans.ParseDay(*agreementOn)
	in := loans.CalcInput{Type: repaymentType, PrincipalMinor: *principal, RateBP: *rate, AgreementOn: start, Rows: make([]loans.CalcRow, len(d.Payments))}
	for i, p := range d.Payments {
		due, err := loans.ParseDay(p.DueOn)
		if err != nil {
			invalid(w)
			return
		}
		in.Rows[i] = loans.CalcRow{DueOn: due, AmountMinor: p.AmountMinor}
	}
	result := loans.CalculateParts(in)
	out := PartsCalculation{Rows: make([]CalculationRow, len(d.Payments)), PrincipalTotalMinor: result.PrincipalTotalMinor, PrincipalDifferenceMinor: result.PrincipalDifferenceMinor}
	for i, row := range result.Rows {
		if row.Warning != "" {
			warning := row.Warning
			out.Rows[i].Warning = &warning
		}
		if row.Parts == nil {
			continue
		}
		source := loans.SourceCalculated
		parts := *row.Parts
		d.Payments[i].PrincipalMinor, d.Payments[i].InterestMinor, d.Payments[i].OtherMinor = &parts.PrincipalMinor, &parts.InterestMinor, &parts.OtherMinor
		d.Payments[i].PartsSource = &source
	}
	if err = tx.QueryRow(r.Context(), `UPDATE payment_schedule_drafts SET payments=$2,revision=revision+1 WHERE id::text=$1 RETURNING revision`, d.ID, d.Payments).Scan(&d.Revision); err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		failure(w, err)
		return
	}
	out.Draft = d
	web.WriteJSON(w, 200, out)
}
