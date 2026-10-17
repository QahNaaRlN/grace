package loans

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

const (
	AllocationOwner    = "owner"
	AllocationSchedule = "schedule"
	AllocationUnknown  = "unknown"
)

// ScheduleAllocation is the part of an operation that paid a schedule row.
type ScheduleAllocation struct {
	SettlementID       string `json:"settlementId"`
	ScheduledPaymentID string `json:"scheduledPaymentId"`
	DueOn              string `json:"dueOn"`
	AmountMinor        int64  `json:"amountMinor"`
	Parts              *Parts `json:"parts"`
	Source             string `json:"source"`
	Revision           int    `json:"revision"`
}

// AllocationView is how one debt payment splits for its loan.
type AllocationView struct {
	FinancialEventID string              `json:"financialEventId"`
	ObligationID     string              `json:"obligationId"`
	ObligationName   string              `json:"obligationName"`
	Currency         string              `json:"currency"`
	AmountMinor      int64               `json:"amountMinor"`
	Schedule         *ScheduleAllocation `json:"schedule"`
	Prepayment       *PrepaymentView     `json:"prepayment"`
	UnallocatedMinor int64               `json:"unallocatedMinor"`
}

// eventLoan finds the loan an operation is linked to.
func eventLoan(ctx context.Context, q Querier, eventID string) (string, error) {
	var loanID string
	err := each(ctx, q, `SELECT o.id::text FROM obligations o JOIN scheduled_payments sp ON sp.obligation_id=o.id
		JOIN scheduled_payment_settlements s ON s.scheduled_payment_id=sp.id WHERE o.kind='loan' AND s.financial_event_id::text=$1
		UNION SELECT obligation_id::text FROM loan_prepayments WHERE financial_event_id::text=$1`, []any{eventID}, func(rows pgx.Rows) error {
		return rows.Scan(&loanID)
	})
	if err == nil && loanID == "" {
		err = errNotLinked
	}
	return loanID, err
}

// Allocation reads how an operation splits for its loan.
func Allocation(ctx context.Context, q Querier, eventID string) (AllocationView, error) {
	loanID, err := eventLoan(ctx, q, eventID)
	if err != nil {
		return AllocationView{}, err
	}
	s, err := loadState(ctx, q, loanID)
	if err != nil {
		return AllocationView{}, err
	}
	view := AllocationView{FinancialEventID: eventID, ObligationID: loanID, ObligationName: s.Name, Currency: s.Currency}
	for _, row := range s.Rows {
		for _, st := range row.Settlements {
			if st.EventID != eventID {
				continue
			}
			a := &ScheduleAllocation{SettlementID: st.ID, ScheduledPaymentID: row.ID, DueOn: row.DueOn, AmountMinor: st.AmountMinor, Parts: st.Parts, Source: AllocationSchedule, Revision: st.OverrideRevision}
			switch {
			case st.Override != nil:
				a.Source = AllocationOwner
			case st.Parts == nil:
				a.Source = AllocationUnknown
				view.UnallocatedMinor += st.AmountMinor
			}
			view.Schedule = a
			view.AmountMinor += st.AmountMinor
		}
	}
	for _, p := range s.Prepayments {
		if p.EventID == eventID {
			view.Prepayment = &PrepaymentView{p.ID, p.EventID, FormatDay(p.On), p.AmountMinor, p.Parts, p.Revision}
			view.AmountMinor += p.AmountMinor
		}
	}
	return view, nil
}

func (h *Handler) GetAllocation(w http.ResponseWriter, r *http.Request) {
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	view, err := Allocation(r.Context(), tx, chi.URLParam(r, "eventId"))
	if err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 200, view)
}

// SaveAllocation replaces the default split of the schedule part with the owner's.
func (h *Handler) SaveAllocation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		optionalParts
		Revision int `json:"revision"`
	}
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	h.allocationWrite(w, r, func(ctx context.Context, tx pgx.Tx, settlementID string, amount int64, current *Parts, revision int) error {
		if body.Revision != revision {
			return pgx.ErrNoRows
		}
		if body.PrincipalMinor == nil || !ValidParts(body.PrincipalMinor, body.InterestMinor, body.OtherMinor, amount) {
			return errInvalid
		}
		parts := Parts{*body.PrincipalMinor, *body.InterestMinor, *body.OtherMinor}
		_, err := tx.Exec(ctx, `INSERT INTO loan_settlement_allocations(settlement_id,principal_minor,interest_minor,other_minor) VALUES($1::uuid,$2,$3,$4)
			ON CONFLICT(settlement_id) DO UPDATE SET principal_minor=EXCLUDED.principal_minor,interest_minor=EXCLUDED.interest_minor,
			other_minor=EXCLUDED.other_minor,revision=loan_settlement_allocations.revision+1`, settlementID, parts.PrincipalMinor, parts.InterestMinor, parts.OtherMinor)
		if err != nil {
			return err
		}
		return recordRevision(ctx, tx, chi.URLParam(r, "eventId"), "schedule", current, parts)
	})
}

// ResetAllocation returns the schedule part to the split derived from the row.
func (h *Handler) ResetAllocation(w http.ResponseWriter, r *http.Request) {
	expected, err := strconv.Atoi(r.URL.Query().Get("revision"))
	if err != nil {
		web.WriteError(w, 422, "validation_error", "Передайте revision распределения.")
		return
	}
	h.allocationWrite(w, r, func(ctx context.Context, tx pgx.Tx, settlementID string, _ int64, current *Parts, revision int) error {
		if current == nil || expected != revision {
			return pgx.ErrNoRows
		}
		if _, err := tx.Exec(ctx, `DELETE FROM loan_settlement_allocations WHERE settlement_id::text=$1`, settlementID); err != nil {
			return err
		}
		return recordRevision(ctx, tx, chi.URLParam(r, "eventId"), "schedule", current, nil)
	})
}

func (h *Handler) allocationWrite(w http.ResponseWriter, r *http.Request, change func(context.Context, pgx.Tx, string, int64, *Parts, int) error) {
	ctx := r.Context()
	eventID := chi.URLParam(r, "eventId")
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var loanID, settlementID string
	var amount int64
	var current *Parts
	var p, i, o *int64
	var revision int
	err = tx.QueryRow(ctx, `SELECT o.id::text,s.id::text,s.amount_minor,a.principal_minor,a.interest_minor,a.other_minor,COALESCE(a.revision,0)
		FROM scheduled_payment_settlements s JOIN scheduled_payments sp ON sp.id=s.scheduled_payment_id JOIN obligations o ON o.id=sp.obligation_id
		LEFT JOIN loan_settlement_allocations a ON a.settlement_id=s.id
		WHERE s.financial_event_id::text=$1 AND o.kind='loan'`, eventID).Scan(&loanID, &settlementID, &amount, &p, &i, &o, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		err = errNotLinked
	}
	if err != nil {
		failure(w, err)
		return
	}
	if _, err = lockLoan(ctx, tx, loanID); err != nil {
		failure(w, err)
		return
	}
	if p != nil {
		current = &Parts{*p, *i, *o}
	}
	if err = change(ctx, tx, settlementID, amount, current, revision); err != nil {
		failure(w, err)
		return
	}
	view, err := Allocation(ctx, tx, eventID)
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 200, view)
}
