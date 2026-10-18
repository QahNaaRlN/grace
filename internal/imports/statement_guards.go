package imports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5"
)

func statementPostingsMatch(pdfAccount string, row statementdata.Row, posts []financialevents.Posting) bool {
	if row.AccountAmount == nil || row.Direction == "" {
		return false
	}
	count := 0
	for _, p := range posts {
		if p.AccountID == pdfAccount {
			if p.Currency != row.AccountAmount.Currency || p.Amount != row.AccountAmount.AmountMinor || p.Direction != row.Direction {
				return false
			}
			count++
		}
	}
	return count == 1
}
func (r *Repository) statementPlanGuards(ctx context.Context, tx pgx.Tx, batch string) ([]string, []string, int64, error) {
	var id string
	e := tx.QueryRow(ctx, "SELECT id::text FROM statement_sources WHERE batch_id=$1", batch).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil, 0, nil
	}
	if e != nil {
		return nil, nil, 0, e
	}
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return nil, nil, 0, e
	}
	if s.Review == nil {
		return []string{"statement_review_required"}, nil, s.Revision, nil
	}
	applyStatementFlags(&s)
	if e = statementAccount(ctx, tx, s.AccountID); e != nil {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "statement_debit_rub_account_required")
	}
	over, e := statementCandidateLimit(ctx, tx, batch)
	if e != nil {
		return nil, nil, 0, e
	}
	if over {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "statement_candidate_limit")
	}
	return s.Review.Reconciliation.Blockers, s.Review.Reconciliation.Warnings, s.Revision, nil
}
func (r *Repository) statementAfterConfirm(ctx context.Context, tx pgx.Tx, batch string) error {
	var id string
	e := tx.QueryRow(ctx, "SELECT id::text FROM statement_sources WHERE batch_id=$1", batch).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return e
	}
	if s.DeletedAt != nil || s.Retention == "keep" {
		return nil
	}
	var pending bool
	e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM import_records WHERE batch_id=$1 AND NOT published)", batch).Scan(&pending)
	if e != nil || pending {
		return e
	}
	return r.deleteStatementOriginal(ctx, tx, &s, "deleted")
}
func statementCandidates(ctx context.Context, tx pgx.Tx, batch string, rec Record) ([]Candidate, error) {
	out := []Candidate{}
	pdf := rec.Source.PDF
	row := pdf.Row
	if row.AccountAmount == nil || row.OperationDate == nil {
		return out, nil
	}
	dates := []string{row.OperationDate.Local[:10]}
	if row.BookingDate != nil {
		dates = append(dates, row.BookingDate.Local[:10])
	}
	min, max := dates[0], dates[0]
	for _, d := range dates {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}
	// Dates define a suggestion window, never proof of identity or permission to
	// change the existing event. A manual link is validated by Posting separately.
	rows, e := tx.Query(ctx, `SELECT DISTINCT e.id::text FROM financial_events e JOIN postings p ON p.financial_event_id=e.id AND p.owner_id=e.owner_id WHERE p.account_id=$1 AND p.direction::text=$2 AND p.currency=$3 AND p.amount_minor=$4 AND e.voided_at IS NULL AND (e.occurred_date BETWEEN $5::date-3 AND $6::date+3 OR e.occurred_at::date BETWEEN $5::date-3 AND $6::date+3) ORDER BY e.id::text LIMIT 1001`, pdf.AccountID, row.Direction, row.AccountAmount.Currency, row.AccountAmount.AmountMinor, min, max)
	if e != nil {
		return nil, e
	}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		out = append(out, Candidate{ID: "event:" + id, EventID: id, Rule: "statement_posting_dates", Differences: []string{"Compare operation and booking dates, description and source; candidate is not a duplicate decision"}})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	rows, e = tx.Query(ctx, `SELECT id::text,source FROM import_records WHERE batch_id=$1 AND id<>$2 AND source->'pdf'->>'accountId'=$3 AND source->'pdf'->'row'->>'direction'=$4 AND source->'normalized'->>'currency'=$5 AND (source->'normalized'->>'amountMinor')::bigint=$6 ORDER BY id LIMIT 1001`, batch, rec.ID, pdf.AccountID, row.Direction, row.AccountAmount.Currency, row.AccountAmount.AmountMinor)
	if e != nil {
		return nil, e
	}
	for rows.Next() {
		var id string
		var raw []byte
		if e = rows.Scan(&id, &raw); e != nil {
			rows.Close()
			return nil, e
		}
		var src struct {
			PDF struct {
				Row statementdata.Row `json:"row"`
			} `json:"pdf"`
		}
		if e = json.Unmarshal(raw, &src); e != nil {
			rows.Close()
			return nil, e
		}
		other := src.PDF.Row.OperationDate
		if other == nil {
			continue
		}
		a, _ := time.Parse("2006-01-02", other.Local[:10])
		near := false
		for _, date := range dates {
			b, _ := time.Parse("2006-01-02", date)
			delta := a.Sub(b)
			if delta >= -72*time.Hour && delta <= 72*time.Hour {
				near = true
			}
		}
		if near {
			out = append(out, Candidate{ID: "record:" + id, RecordID: id, Rule: "statement_batch_posting", Differences: []string{"Equal Posting values still require owner duplicate or separate-operation decision"}})
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if len(out) > 1000 {
		return nil, fault("review_limit", 422)
	}
	return out, nil
}

func statementCandidateLimit(ctx context.Context, tx pgx.Tx, batch string) (bool, error) {
	var n int
	err := tx.QueryRow(ctx, `WITH rows AS (
 SELECT id,source->'pdf'->>'accountId' AS account,
 source->'pdf'->'row'->>'direction' AS direction,
 source->'normalized'->>'currency' AS currency,
 (source->'normalized'->>'amountMinor')::bigint AS amount,
 left(source->'pdf'->'row'->'operationDate'->>'local',10)::date AS operation_date,
 left(source->'pdf'->'row'->'bookingDate'->>'local',10)::date AS booking_date
 FROM import_records WHERE batch_id=$1 AND source->'pdf' IS NOT NULL)
 SELECT count(*) FROM (
 SELECT r.id::text AS a,e.id::text AS b FROM rows r JOIN postings p ON p.account_id::text=r.account AND p.currency=r.currency AND p.direction::text=r.direction AND p.amount_minor=r.amount
 JOIN financial_events e ON e.id=p.financial_event_id AND e.owner_id=p.owner_id AND e.voided_at IS NULL
 WHERE COALESCE(e.occurred_date,e.occurred_at::date) BETWEEN LEAST(r.operation_date,COALESCE(r.booking_date,r.operation_date))-3 AND GREATEST(r.operation_date,COALESCE(r.booking_date,r.operation_date))+3
 UNION ALL
 SELECT a.id::text,b.id::text FROM rows a JOIN rows b ON a.id<b.id AND a.account=b.account AND a.currency=b.currency AND a.direction=b.direction AND a.amount=b.amount
 WHERE a.operation_date BETWEEN LEAST(b.operation_date,COALESCE(b.booking_date,b.operation_date))-3 AND GREATEST(b.operation_date,COALESCE(b.booking_date,b.operation_date))+3
 LIMIT 50001) candidates`, batch).Scan(&n)
	return n > 50000, err
}
