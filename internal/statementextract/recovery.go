package statementextract

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Recover never resumes paid work. It settles only already persisted complete
// call evidence; a started call with no answer freezes the shared allowance.
func (s *Service) Recover(ctx context.Context, id, attempt string) error {
	tx, e := s.begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var status, reservation, usageStatus string
	var started time.Time
	e = tx.QueryRow(ctx, `SELECT a.status,a.reservation_id::text,a.created_at,u.status FROM statement_extraction_attempts a JOIN ai_usage u ON u.id=a.reservation_id AND u.owner_id=a.owner_id WHERE a.id=$1 AND a.statement_id=$2 FOR UPDATE OF a`, attempt, id).Scan(&status, &reservation, &started, &usageStatus)
	if errors.Is(e, pgx.ErrNoRows) {
		return fault("not_found", 404)
	}
	if e != nil {
		return e
	}
	if (status != "running" && status != "cancelled") || usageStatus == "settled" || s.now().Sub(started) <= 130*time.Second {
		return nil
	}
	var incomplete int
	var known int64
	e = tx.QueryRow(ctx, "SELECT count(*) FILTER(WHERE status<>'completed'),COALESCE(sum(cost_micros),0) FROM statement_extraction_calls WHERE attempt_id=$1", attempt).Scan(&incomplete, &known)
	if e != nil {
		return e
	}
	var actual any
	status = "failed"
	if incomplete == 0 {
		actual = known
	} else {
		status = "uncertain"
	}
	if _, e = tx.Exec(ctx, "SELECT grace_ai_settle($1,$2::bigint,$3)", reservation, actual, "statement:"+attempt); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "UPDATE statement_extraction_attempts SET status=$2,error_code='interrupted_attempt',finished_at=now() WHERE id=$1", attempt, status)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) Attempts(ctx context.Context, id string) ([]Attempt, error) {
	var exists bool
	if e := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM statement_sources WHERE id=$1)", id).Scan(&exists); e != nil {
		return nil, e
	}
	if !exists {
		return nil, fault("not_found", 404)
	}
	rows, e := s.pool.Query(ctx, "SELECT id::text FROM statement_extraction_attempts WHERE statement_id=$1 ORDER BY created_at DESC,id DESC LIMIT 20", id)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var attempt string
		if e = rows.Scan(&attempt); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, attempt)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	out := []Attempt{}
	for _, attempt := range ids {
		if e = s.Recover(ctx, id, attempt); e != nil {
			return nil, e
		}
		item, e := s.Attempt(ctx, id, attempt)
		if e != nil {
			return nil, e
		}
		out = append(out, item)
	}
	return out, nil
}
