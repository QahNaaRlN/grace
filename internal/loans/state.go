package loans

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// localDay is the calendar day of an operation in the application time zone.
const localDay = `COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)`

var ErrNotLoan = errors.New("obligation is not a loan")

// Querier is a transaction or a pool.
type Querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type settlementState struct {
	ID               string
	EventID          string
	On               time.Time
	AmountMinor      int64
	Override         *Parts
	OverrideRevision int
	Parts            *Parts // computed: the override or the row's default split
}

type rowState struct {
	ID          string
	DueOn       string
	AmountMinor int64
	Parts       *Parts
	Source      *string
	Settlements []*settlementState
}

func (r *rowState) paid() int64 {
	var total int64
	for _, s := range r.Settlements {
		total += s.AmountMinor
	}
	return total
}

type prepaymentState struct {
	ID          string
	EventID     string
	On          time.Time
	AmountMinor int64
	Parts       Parts
	Revision    int
}

// SnapshotView is the outstanding debt the owner copied from the bank.
type SnapshotView struct {
	ID                   string `json:"id"`
	TakenOn              string `json:"takenOn"`
	PrincipalMinor       int64  `json:"principalMinor"`
	AccruedInterestMinor *int64 `json:"accruedInterestMinor"`
	PayoffMinor          *int64 `json:"payoffMinor"`
	Note                 string `json:"note"`
	Revision             int    `json:"revision"`
}

type loanState struct {
	ID            string
	Name          string
	Currency      string
	Issued        *int64
	RepaymentType string
	RateBP        *int
	AgreementOn   *string
	Rows          []*rowState
	Prepayments   []*prepaymentState
	Snapshots     []SnapshotView // newest first
}

// loadState reads a loan with its active schedule, settlements, prepayments and
// snapshots, and splits every settlement.
func loadState(ctx context.Context, q Querier, obligationID string) (*loanState, error) {
	s := &loanState{ID: obligationID}
	found := false
	err := each(ctx, q, `SELECT o.name,o.currency,o.kind,d.principal_minor,COALESCE(d.repayment_type,'unknown'),d.annual_rate_basis_points,d.agreement_on::text
		FROM obligations o LEFT JOIN loan_details d ON d.obligation_id=o.id WHERE o.id::text=$1`, []any{obligationID}, func(rows pgx.Rows) error {
		var kind string
		found = true
		if err := rows.Scan(&s.Name, &s.Currency, &kind, &s.Issued, &s.RepaymentType, &s.RateBP, &s.AgreementOn); err != nil {
			return err
		}
		if kind != "loan" {
			return ErrNotLoan
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, pgx.ErrNoRows
	}
	byID := map[string]*rowState{}
	err = each(ctx, q, `SELECT sp.id::text,sp.due_on::text,sp.amount_minor,sp.principal_minor,sp.interest_minor,sp.other_minor,sp.parts_source,
			s.id::text,s.financial_event_id::text,s.amount_minor,`+localDay+`::text,a.principal_minor,a.interest_minor,a.other_minor,COALESCE(a.revision,0)
		FROM scheduled_payments sp
		LEFT JOIN scheduled_payment_settlements s ON s.scheduled_payment_id=sp.id
		LEFT JOIN financial_events e ON e.id=s.financial_event_id
		LEFT JOIN loan_settlement_allocations a ON a.settlement_id=s.id
		WHERE sp.obligation_id::text=$1 AND sp.plan_active
		ORDER BY sp.position,`+localDay+`,s.created_at,s.id`, []any{obligationID}, func(rows pgx.Rows) error {
		var row rowState
		var p, i, o *int64
		var sid, eid, day *string
		var amount, ap, ai, ao *int64
		var revision int
		if err := rows.Scan(&row.ID, &row.DueOn, &row.AmountMinor, &p, &i, &o, &row.Source, &sid, &eid, &amount, &day, &ap, &ai, &ao, &revision); err != nil {
			return err
		}
		current, ok := byID[row.ID]
		if !ok {
			if p != nil && i != nil && o != nil {
				row.Parts = &Parts{*p, *i, *o}
			}
			current = &row
			byID[row.ID] = current
			s.Rows = append(s.Rows, current)
		}
		if sid == nil {
			return nil
		}
		on, err := ParseDay(*day)
		if err != nil {
			return err
		}
		st := &settlementState{ID: *sid, EventID: *eid, On: on, AmountMinor: *amount, OverrideRevision: revision}
		if ap != nil {
			st.Override = &Parts{*ap, *ai, *ao}
		}
		current.Settlements = append(current.Settlements, st)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, row := range s.Rows {
		payments := make([]RowPayment, len(row.Settlements))
		for i, st := range row.Settlements {
			payments[i] = RowPayment{AmountMinor: st.AmountMinor, Override: st.Override}
		}
		for i, parts := range AllocateRow(row.Parts, payments) {
			row.Settlements[i].Parts = parts
		}
	}
	err = each(ctx, q, `SELECT p.id::text,p.financial_event_id::text,`+localDay+`::text,p.amount_minor,p.principal_minor,p.interest_minor,p.other_minor,p.revision
		FROM loan_prepayments p JOIN financial_events e ON e.id=p.financial_event_id
		WHERE p.obligation_id::text=$1 ORDER BY 3,p.created_at`, []any{obligationID}, func(rows pgx.Rows) error {
		var p prepaymentState
		var day string
		if err := rows.Scan(&p.ID, &p.EventID, &day, &p.AmountMinor, &p.Parts.PrincipalMinor, &p.Parts.InterestMinor, &p.Parts.OtherMinor, &p.Revision); err != nil {
			return err
		}
		on, err := ParseDay(day)
		if err != nil {
			return err
		}
		p.On = on
		s.Prepayments = append(s.Prepayments, &p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.Snapshots = []SnapshotView{}
	err = each(ctx, q, `SELECT id::text,taken_on::text,principal_minor,accrued_interest_minor,payoff_minor,note,revision
		FROM loan_debt_snapshots WHERE obligation_id::text=$1 ORDER BY taken_on DESC`, []any{obligationID}, func(rows pgx.Rows) error {
		var v SnapshotView
		if err := rows.Scan(&v.ID, &v.TakenOn, &v.PrincipalMinor, &v.AccruedInterestMinor, &v.PayoffMinor, &v.Note, &v.Revision); err != nil {
			return err
		}
		s.Snapshots = append(s.Snapshots, v)
		return nil
	})
	return s, err
}

func each(ctx context.Context, q Querier, sql string, args []any, scan func(pgx.Rows) error) error {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Totals splits money by part; UnknownMinor is money whose split is not known.
type Totals struct {
	PrincipalMinor int64 `json:"principalMinor"`
	InterestMinor  int64 `json:"interestMinor"`
	OtherMinor     int64 `json:"otherMinor"`
	UnknownMinor   int64 `json:"unknownMinor"`
}

func (t *Totals) add(parts *Parts, amount int64) {
	if parts == nil {
		t.UnknownMinor += amount
		return
	}
	t.PrincipalMinor += parts.PrincipalMinor
	t.InterestMinor += parts.InterestMinor
	t.OtherMinor += parts.OtherMinor
}

// PrepaymentView is one early repayment of the loan.
type PrepaymentView struct {
	ID               string `json:"id"`
	FinancialEventID string `json:"financialEventId"`
	PaidOn           string `json:"paidOn"`
	AmountMinor      int64  `json:"amountMinor"`
	Parts
	Revision int `json:"revision"`
}

// DebtStatus is the loan debt for a day: outstanding principal by Grace's data and
// by the bank, what was paid and what the schedule still expects.
type DebtStatus struct {
	On                        string           `json:"on"`
	IssuedMinor               *int64           `json:"issuedMinor"`
	PrincipalOutstandingMinor *int64           `json:"principalOutstandingMinor"`
	Basis                     string           `json:"basis"`
	BasisOn                   *string          `json:"basisOn"`
	UnallocatedPayments       int              `json:"unallocatedPayments"`
	LatestSnapshot            *SnapshotView    `json:"latestSnapshot"`
	GracePrincipalOnSnapshot  *int64           `json:"gracePrincipalOnSnapshotMinor"`
	DiscrepancyMinor          *int64           `json:"discrepancyMinor"`
	Paid                      Totals           `json:"paid"`
	Upcoming                  Totals           `json:"upcoming"`
	ScheduleOutdated          bool             `json:"scheduleOutdated"`
	Snapshots                 []SnapshotView   `json:"snapshots"`
	Prepayments               []PrepaymentView `json:"prepayments"`
}

// Debt computes the loan's debt status at the end of day `on`.
func Debt(ctx context.Context, q Querier, obligationID string, on time.Time) (DebtStatus, error) {
	s, err := loadState(ctx, q, obligationID)
	if err != nil {
		return DebtStatus{}, err
	}
	return s.debt(on), nil
}

func (s *loanState) movements() []Movement {
	var out []Movement
	for _, row := range s.Rows {
		for _, st := range row.Settlements {
			m := Movement{On: st.On}
			if st.Parts != nil {
				v := st.Parts.PrincipalMinor
				m.PrincipalMinor = &v
			}
			out = append(out, m)
		}
	}
	for _, p := range s.Prepayments {
		v := p.Parts.PrincipalMinor
		out = append(out, Movement{On: p.On, PrincipalMinor: &v})
	}
	return out
}

func (s *loanState) snapshots() []Snapshot {
	out := make([]Snapshot, 0, len(s.Snapshots))
	for _, v := range s.Snapshots {
		if on, err := ParseDay(v.TakenOn); err == nil {
			out = append(out, Snapshot{on, v.PrincipalMinor})
		}
	}
	return out
}

func (s *loanState) debt(on time.Time) DebtStatus {
	movements, snapshots := s.movements(), s.snapshots()
	out := OutstandingOn(s.Issued, snapshots, movements, on)
	status := DebtStatus{
		On: FormatDay(on), IssuedMinor: s.Issued, PrincipalOutstandingMinor: out.PrincipalMinor,
		Basis: out.Basis, UnallocatedPayments: out.Unallocated, Snapshots: s.Snapshots, Prepayments: []PrepaymentView{},
	}
	if out.BasisOn != nil {
		v := FormatDay(*out.BasisOn)
		status.BasisOn = &v
	}
	if latest, grace, diff := Discrepancy(s.Issued, snapshots, movements, on); latest != nil {
		for i := range s.Snapshots {
			if s.Snapshots[i].TakenOn == FormatDay(latest.On) {
				status.LatestSnapshot = &s.Snapshots[i]
			}
		}
		status.GracePrincipalOnSnapshot, status.DiscrepancyMinor = grace, diff
	}
	upcomingKnown := true
	for _, row := range s.Rows {
		left := row.Parts
		if left != nil {
			parts := *left
			left = &parts
		}
		for _, st := range row.Settlements {
			if !st.On.After(on) {
				status.Paid.add(st.Parts, st.AmountMinor)
			}
			if left != nil && st.Parts != nil {
				*left = Parts{left.PrincipalMinor - st.Parts.PrincipalMinor, left.InterestMinor - st.Parts.InterestMinor, left.OtherMinor - st.Parts.OtherMinor}
			} else {
				left = nil
			}
		}
		remaining := row.AmountMinor - row.paid()
		if remaining <= 0 {
			continue
		}
		if left == nil || !left.valid() || left.Total() != remaining {
			upcomingKnown = false
			status.Upcoming.add(nil, remaining)
			continue
		}
		status.Upcoming.add(left, remaining)
	}
	for _, p := range s.Prepayments {
		if !p.On.After(on) {
			status.Paid.add(&p.Parts, p.AmountMinor)
		}
		status.Prepayments = append(status.Prepayments, PrepaymentView{p.ID, p.EventID, FormatDay(p.On), p.AmountMinor, p.Parts, p.Revision})
	}
	// The schedule is outdated when the principal it still expects differs from
	// the principal left (Grace's value, else the latest bank snapshot).
	current := status.PrincipalOutstandingMinor
	if current == nil && status.LatestSnapshot != nil {
		v := status.LatestSnapshot.PrincipalMinor
		current = &v
	}
	status.ScheduleOutdated = upcomingKnown && current != nil && *current != status.Upcoming.PrincipalMinor
	return status
}

// EventSplits returns the known loan split of each of the given debt payments:
// the part that paid a schedule row and the prepayment part together. Events
// without a loan link, or whose split is unknown, are absent or partial.
func EventSplits(ctx context.Context, q Querier, eventIDs []string) (map[string]Parts, error) {
	out := map[string]Parts{}
	if len(eventIDs) == 0 {
		return out, nil
	}
	var loanIDs []string
	err := each(ctx, q, `SELECT DISTINCT o.id::text FROM obligations o
		JOIN scheduled_payments sp ON sp.obligation_id=o.id
		JOIN scheduled_payment_settlements s ON s.scheduled_payment_id=sp.id
		WHERE o.kind='loan' AND s.financial_event_id::text=ANY($1)
		UNION SELECT obligation_id::text FROM loan_prepayments WHERE financial_event_id::text=ANY($1)`, []any{eventIDs}, func(rows pgx.Rows) error {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		loanIDs = append(loanIDs, id)
		return nil
	})
	if err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	for _, id := range eventIDs {
		wanted[id] = true
	}
	for _, id := range loanIDs {
		s, err := loadState(ctx, q, id)
		if err != nil {
			return nil, err
		}
		for _, row := range s.Rows {
			for _, st := range row.Settlements {
				if wanted[st.EventID] && st.Parts != nil {
					out[st.EventID] = out[st.EventID].add(*st.Parts)
				}
			}
		}
		for _, p := range s.Prepayments {
			if wanted[p.EventID] {
				out[p.EventID] = out[p.EventID].add(p.Parts)
			}
		}
	}
	return out, nil
}
