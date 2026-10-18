package statementextract

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool     *database.TenantPool
	imports  *imports.Repository
	reasoner Reasoner
	now      func() time.Time
}

func NewService(pool *pgxpool.Pool, repo *imports.Repository, reasoner Reasoner) *Service {
	return &Service{database.ForUsers(pool), repo, reasoner, time.Now}
}
func (s *Service) begin(ctx context.Context) (pgx.Tx, error) {
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return nil, e
	}
	if e = financialevents.LockOwner(ctx, tx); e != nil {
		_ = tx.Rollback(ctx)
		return nil, e
	}
	return tx, nil
}
func validKey(key string) bool { return strings.TrimSpace(key) != "" && len(key) <= 160 }
func (s *Service) policy(ctx context.Context) (Policy, error) {
	if s.reasoner == nil {
		return Policy{}, fault("ai_disabled", 503)
	}
	p, e := s.reasoner.CurrentPolicy(ctx)
	if e != nil {
		return p, e
	}
	if !p.Valid() {
		return p, fault("price_unknown", 503)
	}
	return p, nil
}
func sourceCurrent(ctx context.Context, tx pgx.Tx, id string, revision int64) error {
	var actual int64
	var active, published bool
	e := tx.QueryRow(ctx, "SELECT revision,deleted_at IS NULL AND (retention='keep' OR expires_at>now()),EXISTS(SELECT 1 FROM import_records r WHERE r.batch_id=statement_sources.batch_id AND r.published) FROM statement_sources WHERE id=$1 FOR UPDATE", id).Scan(&actual, &active, &published)
	if errors.Is(e, pgx.ErrNoRows) {
		return fault("not_found", 404)
	}
	if e != nil {
		return e
	}
	if !active {
		return fault("statement_original_gone", 410)
	}
	if published {
		return fault("statement_draft_published", 409)
	}
	if actual != revision {
		return fault("revision_conflict", 409)
	}
	return nil
}
func (s *Service) Preview(ctx context.Context, id string, in PreviewInput) (Preview, error) {
	var out Preview
	policy, e := s.policy(ctx)
	if e != nil {
		return out, e
	}
	source, e := s.imports.Statement(ctx, id)
	if e != nil {
		return out, e
	}
	if source.Layout == nil {
		return out, fault("statement_original_gone", 410)
	}
	if source.PublishedRows > 0 {
		return out, fault("statement_draft_published", 409)
	}
	plan, e := BuildPlan(*source.Layout, policy, in.VisibleSpanIDs)
	if e != nil {
		return out, fault(e.Error(), 422)
	}
	tx, e := s.begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(ctx)
	if e = sourceCurrent(ctx, tx, id, in.ExpectedRevision); e != nil {
		return out, e
	}
	var calls int
	if e = tx.QueryRow(ctx, "SELECT ai_calls FROM statement_sources WHERE id=$1", id).Scan(&calls); e != nil {
		return out, e
	}
	if calls+len(plan.Chunks) > MaxCalls {
		return out, fault("statement_call_limit", 422)
	}
	out = Preview{StatementID: id, SourceRevision: in.ExpectedRevision, Hash: Hash(plan), Plan: plan}
	encoded, _ := json.Marshal(plan)
	e = tx.QueryRow(ctx, "INSERT INTO statement_extraction_previews(statement_id,source_revision,state_hash,plan_hash,plan,maximum_micros,call_count,policy,prompt_version,schema_hash,redaction_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id::text,expires_at", id, in.ExpectedRevision, plan.DocumentHash, out.Hash, encoded, plan.MaximumMicros, len(plan.Chunks), mustPolicyJSON(plan.Policy), plan.PromptVersion, plan.SchemaHash, plan.RedactionVersion).Scan(&out.ID, &out.ExpiresAt)
	if e != nil {
		return out, e
	}
	return out, tx.Commit(ctx)
}
func loadPreview(ctx context.Context, tx pgx.Tx, id, statement string) (Preview, error) {
	var out Preview
	var encoded []byte
	e := tx.QueryRow(ctx, "SELECT id::text,statement_id::text,source_revision,plan_hash,plan,expires_at FROM statement_extraction_previews WHERE id=$1 AND statement_id=$2", id, statement).Scan(&out.ID, &out.StatementID, &out.SourceRevision, &out.Hash, &encoded, &out.ExpiresAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, fault("not_found", 404)
	}
	if e != nil {
		return out, e
	}
	if len(encoded) == 0 {
		return out, fault("statement_original_gone", 410)
	}
	if e = json.Unmarshal(encoded, &out.Plan); e != nil {
		return out, e
	}
	if Hash(out.Plan) != out.Hash {
		return out, fault("preview_changed", 409)
	}
	return out, nil
}
func (s *Service) Consent(ctx context.Context, id, key string, in ConsentInput) (Consent, error) {
	var out Consent
	if !validKey(key) {
		return out, fault("idempotency_key_required", 400)
	}
	tx, e := s.begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(ctx)
	var previousPreview, previousHash, previousSource string
	var previousRevision int64
	e = tx.QueryRow(ctx, "SELECT c.id::text,c.preview_id::text,c.preview_hash,c.statement_id::text,p.source_revision FROM statement_extraction_consents c JOIN statement_extraction_previews p ON p.id=c.preview_id AND p.owner_id=c.owner_id WHERE c.creation_key=$1", key).Scan(&out.ID, &previousPreview, &previousHash, &previousSource, &previousRevision)
	if e == nil {
		if previousPreview != in.PreviewID || previousHash != in.PreviewHash || previousSource != id || previousRevision != in.ExpectedRevision {
			return out, fault("idempotency_conflict", 409)
		}
		out.PreviewID = previousPreview
		return out, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return out, e
	}
	p, e := loadPreview(ctx, tx, in.PreviewID, id)
	if e != nil {
		return out, e
	}
	if p.Hash != in.PreviewHash || p.SourceRevision != in.ExpectedRevision {
		return out, fault("preview_changed", 409)
	}
	if !s.now().Before(p.ExpiresAt) {
		return out, fault("preview_expired", 409)
	}
	if e = sourceCurrent(ctx, tx, id, in.ExpectedRevision); e != nil {
		return out, e
	}
	out.PreviewID = p.ID
	e = tx.QueryRow(ctx, "INSERT INTO statement_extraction_consents(statement_id,preview_id,preview_hash,creation_key) VALUES($1,$2,$3,$4) RETURNING id::text", id, p.ID, p.Hash, key).Scan(&out.ID)
	if e != nil {
		return out, e
	}
	return out, tx.Commit(ctx)
}
func (s *Service) Attempt(ctx context.Context, id, attempt string) (Attempt, error) {
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if e != nil {
		return Attempt{}, e
	}
	defer tx.Rollback(ctx)
	return loadAttempt(ctx, tx, id, attempt)
}
func loadAttempt(ctx context.Context, tx pgx.Tx, id, attempt string) (Attempt, error) {
	var out Attempt
	var policy []byte
	e := tx.QueryRow(ctx, `SELECT a.id::text,a.statement_id::text,a.status,a.error_code,s.batch_id::text,p.policy,p.prompt_version,p.schema_hash,p.redaction_version FROM statement_extraction_attempts a JOIN statement_extraction_previews p ON p.id=a.preview_id AND p.owner_id=a.owner_id JOIN statement_sources s ON s.id=a.statement_id AND s.owner_id=a.owner_id WHERE a.id=$1 AND a.statement_id=$2`, attempt, id).Scan(&out.ID, &out.StatementID, &out.Status, &out.ErrorCode, &out.BatchID, &policy, &out.PromptVersion, &out.SchemaHash, &out.RedactionVersion)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, fault("not_found", 404)
	}
	if e == nil {
		e = json.Unmarshal(policy, &out.Policy)
	}
	return out, e
}

func mustPolicyJSON(p Policy) []byte { raw, _ := json.Marshal(p); return raw }
