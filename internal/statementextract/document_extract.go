package statementextract

import (
	"context"
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Service) extractDocument(ctx context.Context, id, key string, in ExtractionInput) (Attempt, error) {
	ctx, cancelRequest := context.WithTimeout(ctx, 210*time.Second)
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
	source, e := s.imports.Statement(ctx, id)
	if e != nil {
		return out, e
	}
	if source.Rendered.Validate() != nil {
		return out, fault("document_limit", 422)
	}
	docPolicy, e := s.document.CurrentDocumentPolicy(ctx, len(source.Rendered.Pages))
	policy := docPolicy.Policy()
	if e == nil {
		e = s.document.PrepareDocument(ctx, docPolicy, source.Rendered)
	}
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
	attemptCtx, cancel := context.WithTimeout(ctx, 210*time.Second)
	defer cancel()
	// Cross-process cancellation observes source revisions and tombstones. There
	// is no long-lived financial transaction across a provider request.
	watchDone := make(chan struct{})
	defer close(watchDone)
	go s.watchSource(attemptCtx, cancel, watchDone, id, in.ExpectedRevision)
	var reading statementdata.VisualReading
	actual := int64(0)
	unknown := false
	code := ""
	for index := range preview.Plan.Chunks {
		if e = s.claimCall(attemptCtx, id, out.ID, in.ExpectedRevision, index+1); e != nil {
			code = "statement_incomplete"
			break
		}
		callCtx, callCancel := context.WithTimeout(attemptCtx, 180*time.Second)
		resultReading, billing, callErr := s.document.CompleteDocument(callCtx, docPolicy, source.Rendered)
		callCancel()
		if callErr == nil {
			callErr = resultReading.Validate(source.PageCount)
		}
		if billing == nil || billing.GenerationID == "" || len(billing.GenerationID) > 200 || billing.CostMicros < 0 || billing.CostMicros > docPolicy.MaximumMicros || billing.InputTokens < 0 || billing.InputTokens > 30000 || billing.OutputTokens < 0 || billing.OutputTokens > 16000 {
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
		reading = resultReading
	}
	var proposal statementdata.Proposal
	if code == "" {
		if source.ReadingMode != "image_review" {
			if source.Layout == nil {
				e = fault("statement_original_gone", 410)
			} else {
				var binding DirectBinding
				binding, e = BindVisualReading(*source.Layout, reading)
				proposal = binding.Proposal
				if e == nil && !binding.Passed {
					e = fault("source_binding_failed", 422)
				}
			}
		}
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
				if source.ReadingMode == "image_review" {
					_, e = s.imports.MaterializeVisualInTx(persist, nested, id, in.ExpectedRevision, reading, "model", "Owner-authorized image reading", out.ID)
				} else {
					_, e = s.imports.MaterializeStatementInTx(persist, nested, id, in.ExpectedRevision, proposal, "model", "Owner-authorized statement extraction", out.ID)
				}
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
