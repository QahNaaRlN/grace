package statementextract

import (
	"context"
	"errors"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5"
)

// Extract claims and reserves the entire plan once. A replay only reads the
// durable attempt: no automatic retry or recovery ever invokes the provider.
func (s *Service) Extract(ctx context.Context, id, key string, in ExtractionInput) (Attempt, error) {
	if s.document != nil {
		return s.extractDocument(ctx, id, key, in)
	}
	ctx, cancelRequest := context.WithTimeout(ctx, AttemptTimeoutSeconds*time.Second)
	defer cancelRequest()
	var out Attempt
	if !validKey(key) {
		return out, fault("idempotency_key_required", 400)
	}
	tx, e := s.begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(ctx)
	var prior, requestHash string
	e = tx.QueryRow(ctx, "SELECT id::text,request_hash FROM statement_extraction_attempts WHERE statement_id=$1 AND idempotency_key=$2", id, key).Scan(&prior, &requestHash)
	if e == nil {
		if requestHash != Hash(in) {
			return out, fault("idempotency_conflict", 409)
		}
		return loadAttempt(ctx, tx, id, prior)
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return out, e
	}
	// Public policy verification is outside the SQL transaction below. No user
	// fragments are involved. The durable preview policy must still match it.
	_ = tx.Rollback(ctx)
	policy, e := s.policy(ctx)
	if e != nil {
		return out, e
	}
	tx, e = s.begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(ctx)
	e = tx.QueryRow(ctx, "SELECT id::text,request_hash FROM statement_extraction_attempts WHERE statement_id=$1 AND idempotency_key=$2", id, key).Scan(&prior, &requestHash)
	if e == nil {
		if requestHash != Hash(in) {
			return out, fault("idempotency_conflict", 409)
		}
		return loadAttempt(ctx, tx, id, prior)
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return out, e
	}
	preview, e := loadPreview(ctx, tx, in.PreviewID, id)
	if e != nil {
		return out, e
	}
	if preview.Hash != in.PreviewHash || preview.SourceRevision != in.ExpectedRevision || Hash(preview.Plan.Policy) != Hash(policy) {
		return out, fault("preview_changed", 409)
	}
	if !s.now().Before(preview.ExpiresAt) {
		return out, fault("preview_expired", 409)
	}
	if e = sourceCurrent(ctx, tx, id, in.ExpectedRevision); e != nil {
		return out, e
	}
	var yes bool
	e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM statement_extraction_consents WHERE id=$1 AND statement_id=$2 AND preview_id=$3 AND preview_hash=$4)", in.ConsentID, id, preview.ID, preview.Hash).Scan(&yes)
	if e != nil {
		return out, e
	}
	if !yes {
		return out, fault("consent_required", 422)
	}
	var usedPreview bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM statement_extraction_attempts WHERE preview_id=$1 AND statement_id=$2)", preview.ID, id).Scan(&usedPreview); e != nil {
		return out, e
	}
	if usedPreview {
		return out, fault("preview_changed", 409)
	}
	var calls int
	var running bool
	e = tx.QueryRow(ctx, "SELECT ai_calls,EXISTS(SELECT 1 FROM statement_extraction_attempts WHERE statement_id=$1 AND status='running') FROM statement_sources WHERE id=$1", id).Scan(&calls, &running)
	if e != nil {
		return out, e
	}
	if running {
		return out, fault("extraction_busy", 429)
	}
	if calls+len(preview.Plan.Chunks) > MaxCalls {
		return out, fault("statement_call_limit", 422)
	}
	var reservation string
	e = tx.QueryRow(ctx, "SELECT grace_ai_reserve($1,$2,$3,'statement')::text", preview.Plan.MaximumMicros, policy.Model, policy.Provider).Scan(&reservation)
	if e != nil {
		return out, e
	}
	e = tx.QueryRow(ctx, "INSERT INTO statement_extraction_attempts(statement_id,preview_id,consent_id,source_revision,reservation_id,idempotency_key,request_hash) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id::text", id, preview.ID, in.ConsentID, in.ExpectedRevision, reservation, key, Hash(in)).Scan(&out.ID)
	if e != nil {
		return out, e
	}
	if e = tx.Commit(ctx); e != nil {
		return out, e
	}
	attemptCtx, cancel := context.WithTimeout(ctx, AttemptTimeoutSeconds*time.Second)
	defer cancel()
	// Cross-process cancellation observes source revisions and tombstones. There
	// is no long-lived financial transaction across a provider request.
	watchDone := make(chan struct{})
	defer close(watchDone)
	go s.watchSource(attemptCtx, cancel, watchDone, id, in.ExpectedRevision)
	parts := []statementdata.Proposal{}
	actual := int64(0)
	unknown := false
	code := ""
	for index, chunk := range preview.Plan.Chunks {
		if e = s.claimCall(attemptCtx, id, out.ID, in.ExpectedRevision, index+1); e != nil {
			code = "statement_incomplete"
			break
		}
		callCtx, callCancel := context.WithTimeout(attemptCtx, CallTimeoutSeconds*time.Second)
		proposal, billing, callErr := s.reasoner.Complete(callCtx, policy, chunk)
		callCancel()
		if callErr == nil {
			proposal, callErr = CanonicalProposal(preview.Plan, proposal)
		}
		if billing == nil || billing.GenerationID == "" || len(billing.GenerationID) > 200 || billing.CostMicros < 0 || billing.CostMicros > policy.CallMicros() || billing.InputTokens < 0 || billing.InputTokens > InputBound || billing.OutputTokens < 0 || billing.OutputTokens > OutputBound {
			unknown = true
			code = "usage_requires_review"
		} else {
			actual += billing.CostMicros
		}
		if e = s.recordCall(ctx, id, out.ID, index+1, billing, unknown); e != nil {
			unknown = true
			code = "usage_requires_review"
		}
		if unknown {
			break
		}
		if callErr != nil {
			code = "provider_protocol"
			break
		}
		parts = append(parts, proposal)
	}
	var proposal statementdata.Proposal
	if code == "" {
		proposal, e = MergeProposals(preview.Plan, parts)
		if e != nil {
			code = e.Error()
		}
	}
	// Persist billing even after browser cancellation. An unknown response keeps
	// the full reservation uncertain and freezes further global spending.
	persist, cancelPersist := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancelPersist()
	tx, e = s.begin(persist)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(persist)
	var cost any
	if !unknown {
		cost = actual
	}
	if _, e = tx.Exec(persist, "SELECT grace_ai_settle($1,$2::bigint,$3)", reservation, cost, "statement:"+out.ID); e != nil {
		return out, e
	}
	status := "failed"
	if unknown {
		status = "uncertain"
	}
	if code == "" {
		if e = sourceCurrent(persist, tx, id, in.ExpectedRevision); e != nil {
			code = "stale_extraction"
		} else {
			var stillRunning bool
			e = tx.QueryRow(persist, "SELECT status='running' FROM statement_extraction_attempts WHERE id=$1", out.ID).Scan(&stillRunning)
			if e != nil {
				return out, e
			}
			if !stillRunning {
				code = "stale_extraction"
			} else {
				// A savepoint retains billing evidence if materialization is rejected.
				nested, e := tx.Begin(persist)
				if e != nil {
					return out, e
				}
				_, e = s.imports.MaterializeStatementInTx(persist, nested, id, in.ExpectedRevision, proposal, "model", "Owner-authorized statement extraction", out.ID)
				if e != nil {
					_ = nested.Rollback(persist)
					code = "proposal_rejected"
				} else if e = nested.Commit(persist); e != nil {
					return out, e
				} else {
					status = "completed"
				}
			}
		}
	}
	var errorCode any
	if code != "" {
		errorCode = code
	}
	_, e = tx.Exec(persist, "UPDATE statement_extraction_attempts SET status=$2,error_code=$3,finished_at=now() WHERE id=$1", out.ID, status, errorCode)
	if e != nil {
		return out, e
	}
	out, e = loadAttempt(persist, tx, id, out.ID)
	if e != nil {
		return out, e
	}
	return out, tx.Commit(persist)
}
func (s *Service) watchSource(ctx context.Context, cancel context.CancelFunc, done <-chan struct{}, id string, revision int64) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			var valid bool
			e := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM statement_sources s WHERE id=$1 AND revision=$2 AND deleted_at IS NULL AND (retention='keep' OR expires_at>now()) AND NOT EXISTS(SELECT 1 FROM import_records r WHERE r.batch_id=s.batch_id AND r.published))", id, revision).Scan(&valid)
			if e != nil || !valid {
				cancel()
				return
			}
		}
	}
}
func (s *Service) claimCall(ctx context.Context, id, attempt string, revision int64, index int) error {
	tx, e := s.begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = sourceCurrent(ctx, tx, id, revision); e != nil {
		return e
	}
	var running bool
	e = tx.QueryRow(ctx, "SELECT status='running' AND grace_statement_call_allowed(reservation_id) FROM statement_extraction_attempts WHERE id=$1 AND statement_id=$2", attempt, id).Scan(&running)
	if e != nil {
		return e
	}
	if !running {
		return fault("stale_extraction", 409)
	}
	result, e := tx.Exec(ctx, "UPDATE statement_sources SET ai_calls=ai_calls+1 WHERE id=$1 AND ai_calls<32", id)
	if e != nil {
		return e
	}
	if result.RowsAffected() != 1 {
		return fault("statement_call_limit", 422)
	}
	_, e = tx.Exec(ctx, "INSERT INTO statement_extraction_calls(attempt_id,statement_id,call_index) VALUES($1,$2,$3)", attempt, id, index)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Service) recordCall(ctx context.Context, id, attempt string, index int, b *Billing, unknown bool) error {
	persist, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	tx, e := s.begin(persist)
	if e != nil {
		return e
	}
	defer tx.Rollback(persist)
	var generation, cost, input, output any
	status := "uncertain"
	if !unknown && b != nil {
		generation = b.GenerationID
		cost = b.CostMicros
		input = b.InputTokens
		output = b.OutputTokens
		status = "completed"
	}
	_, e = tx.Exec(persist, "UPDATE statement_extraction_calls SET status=$4,generation_id=$5,cost_micros=$6,input_tokens=$7,output_tokens=$8,finished_at=now() WHERE attempt_id=$1 AND statement_id=$2 AND call_index=$3", attempt, id, index, status, generation, cost, input, output)
	if e != nil {
		return e
	}
	return tx.Commit(persist)
}
