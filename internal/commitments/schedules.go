package commitments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
}
type ScheduleDraft struct {
	ID        string         `json:"id"`
	Payments  []DraftPayment `json:"payments"`
	Revision  int            `json:"revision"`
	AppliedAt *time.Time     `json:"appliedAt"`
}

func snapshot(ctx context.Context, tx pgx.Tx, id string) ([]DraftPayment, string, error) {
	rows, err := tx.Query(ctx, `SELECT sp.id::text,sp.due_on::text,sp.amount_minor,COALESCE((SELECT sum(s.amount_minor) FROM scheduled_payment_settlements s WHERE s.scheduled_payment_id=sp.id),0) FROM scheduled_payments sp WHERE sp.obligation_id::text=$1 AND sp.plan_active ORDER BY sp.position FOR UPDATE OF sp`, id)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	payments := []DraftPayment{}
	for rows.Next() {
		var p DraftPayment
		if err = rows.Scan(&p.SourceID, &p.DueOn, &p.AmountMinor, &p.PaidMinor); err != nil {
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
	for _, p := range payments {
		if !validDate(p.DueOn) || p.DueOn <= last || p.AmountMinor < 1 || p.AmountMinor > 9007199254740991-total || seen[p.SourceID] && p.SourceID != "" {
			return 0, false
		}
		if p.SourceID != "" && !sources[p.SourceID] {
			return 0, false
		}
		if old, ok := locked[p.SourceID]; ok {
			if old != p {
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
				_, err = tx.Exec(r.Context(), `UPDATE scheduled_payments SET position=$2 WHERE id::text=$1 AND plan_active`, p.SourceID, i+1)
			} else {
				err = tx.QueryRow(r.Context(), `INSERT INTO scheduled_payments(obligation_id,position,due_on,amount_minor) VALUES($1,$2,$3,$4) RETURNING id::text`, id, i+1, p.DueOn, p.AmountMinor).Scan(&d.Payments[i].SourceID)
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
