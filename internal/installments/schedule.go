package installments

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"net/http"
)

func loadDraft(ctx context.Context, tx pgx.Tx, id, planID string) (Draft, error) {
	var d Draft
	e := tx.QueryRow(ctx, `SELECT id::text,revision,plan_revision,payments,applied FROM installment_schedule_drafts WHERE id::text=$1 AND plan_id::text=$2`, id, planID).Scan(&d.ID, &d.Revision, &d.PlanRevision, &d.Payments, &d.Applied)
	return d, e
}
func (h *Handler) CreateDraft(w http.ResponseWriter, r *http.Request) {
	var c DraftRequest
	if !web.DecodeJSON(w, r, &c) {
		return
	}
	h.write(w, r, c, func(ctx context.Context, tx pgx.Tx) (any, error) {
		p, e := readPlan(ctx, tx, chi.URLParam(r, "planId"), Today())
		if e != nil {
			return nil, e
		}
		if c.Revision != p.Revision {
			return nil, ErrConflict
		}
		rows := []Row{}
		for _, v := range p.Payments {
			rows = append(rows, Row{v.DueOn, v.AmountMinor})
		}
		var id string
		e = tx.QueryRow(ctx, `INSERT INTO installment_schedule_drafts(plan_id,plan_revision,payments) VALUES($1::uuid,$2,$3) RETURNING id::text`, p.ID, p.Revision, rows).Scan(&id)
		if e != nil {
			return nil, e
		}
		return loadDraft(ctx, tx, id, p.ID)
	})
}
func (h *Handler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	var c DraftRequest
	if !web.DecodeJSON(w, r, &c) {
		return
	}
	h.write(w, r, c, func(ctx context.Context, tx pgx.Tx) (any, error) {
		id, planID := chi.URLParam(r, "draftId"), chi.URLParam(r, "planId")
		d, e := loadDraft(ctx, tx, id, planID)
		if e != nil {
			return nil, e
		}
		p, e := readPlan(ctx, tx, planID, Today())
		if e != nil {
			return nil, e
		}
		if d.Applied || d.Revision != c.Revision || d.PlanRevision != p.Revision {
			return nil, ErrConflict
		}
		if !validRows(c.Payments, p.InitialRemainingMinor, p.ObservedOn) {
			return nil, ErrInvalid
		}
		_, e = tx.Exec(ctx, `UPDATE installment_schedule_drafts SET payments=$2,revision=revision+1 WHERE id::text=$1`, id, c.Payments)
		if e != nil {
			return nil, e
		}
		return loadDraft(ctx, tx, id, planID)
	})
}
func (h *Handler) ApplyDraft(w http.ResponseWriter, r *http.Request) {
	var c struct {
		Revision     int `json:"revision"`
		PlanRevision int `json:"planRevision"`
	}
	if !web.DecodeJSON(w, r, &c) {
		return
	}
	h.write(w, r, c, func(ctx context.Context, tx pgx.Tx) (any, error) {
		id, planID := chi.URLParam(r, "draftId"), chi.URLParam(r, "planId")
		d, e := loadDraft(ctx, tx, id, planID)
		if e != nil {
			return nil, e
		}
		p, e := readPlan(ctx, tx, planID, Today())
		if e != nil {
			return nil, e
		}
		if d.Applied || d.Revision != c.Revision || p.Revision != c.PlanRevision || d.PlanRevision != p.Revision {
			return nil, ErrConflict
		}
		if !validRows(d.Payments, p.InitialRemainingMinor, p.ObservedOn) {
			return nil, ErrInvalid
		}
		// Paid rows retain their IDs, amount, position and date. Unpaid replaced rows
		// remain archived so old drafts and audit history keep their provenance.
		for n, row := range p.Payments {
			var allocated bool
			e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM installment_payment_allocations WHERE payment_id::text=$1 AND active)`, row.ID).Scan(&allocated)
			if e != nil {
				return nil, e
			}
			if allocated {
				if n >= len(d.Payments) || d.Payments[n].DueOn != row.DueOn || d.Payments[n].AmountMinor != row.AmountMinor {
					return nil, ErrConflict
				}
			} else {
				if _, e = tx.Exec(ctx, `UPDATE scheduled_payments SET plan_active=false WHERE id::text=$1`, row.ID); e != nil {
					return nil, e
				}
			}
		}
		for n, row := range d.Payments {
			var exists bool
			e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM scheduled_payments WHERE obligation_id=$1::uuid AND plan_active AND position=$2)`, p.ObligationID, n+1).Scan(&exists)
			if e != nil {
				return nil, e
			}
			if !exists {
				if _, e = tx.Exec(ctx, `INSERT INTO scheduled_payments(obligation_id,position,due_on,amount_minor) VALUES($1::uuid,$2,$3::date,$4)`, p.ObligationID, n+1, row.DueOn, row.AmountMinor); e != nil {
					return nil, e
				}
			}
		}
		_, e = tx.Exec(ctx, `UPDATE installment_plans SET revision=revision+1,updated_at=now() WHERE id::text=$1`, p.ID)
		if e != nil {
			return nil, e
		}
		_, e = tx.Exec(ctx, `UPDATE installment_schedule_drafts SET applied=true WHERE id::text=$1`, id)
		if e != nil {
			return nil, e
		}
		return readPlan(ctx, tx, p.ID, Today())
	})
}
