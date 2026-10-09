package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAIConflict = errors.New("ai_request_conflict")
var ErrAIInProgress = errors.New("ai_request_in_progress")
var ErrAINotFound = errors.New("not_found")

type ConversationRepository struct{ pool *database.TenantPool }

func NewConversationRepository(p *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{database.ForUsers(p)}
}
func aiHash(value any) string {
	b, _ := json.Marshal(value)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func ConsentHash(scope string, policy ModelPolicy) string {
	return aiHash(struct{ Scope, Model, Provider, Version string }{scope, policy.Model, policy.Provider, "grace.ai.20261008.zdr.confirmed"})
}
func (r *ConversationRepository) Consent(ctx context.Context, scope, hash string, grant bool) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if grant {
		_, err = tx.Exec(ctx, `INSERT INTO ai_consents(scope,policy_hash) VALUES($1,$2) ON CONFLICT(owner_id,scope) DO UPDATE SET policy_hash=excluded.policy_hash,granted_at=now()`, scope, hash)
	} else {
		_, err = tx.Exec(ctx, "DELETE FROM ai_consents WHERE scope=$1", scope)
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *ConversationRepository) HasConsent(ctx context.Context, scope, hash string) (bool, error) {
	var yes bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ai_consents WHERE scope=$1 AND policy_hash=$2)", scope, hash).Scan(&yes)
	return yes, err
}
func (r *ConversationRepository) Create(ctx context.Context) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, "INSERT INTO ai_conversations DEFAULT VALUES RETURNING id::text").Scan(&id)
	return id, err
}
func (r *ConversationRepository) Delete(ctx context.Context, id string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var deleted string
	err = tx.QueryRow(ctx, "DELETE FROM ai_conversations WHERE id::text=$1 RETURNING id::text", id).Scan(&deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAINotFound
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type ConversationMessage struct {
	ID        string          `json:"id"`
	Question  string          `json:"question"`
	Month     string          `json:"month"`
	Status    string          `json:"status"`
	Facts     json.RawMessage `json:"facts"`
	Response  json.RawMessage `json:"response"`
	ErrorCode *string         `json:"errorCode"`
}

func (r *ConversationRepository) Get(ctx context.Context, id string) ([]ConversationMessage, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ai_conversations WHERE id::text=$1)", id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrAINotFound
	}
	rows, err := r.pool.Query(ctx, `SELECT id::text,question,month,status,facts,response,error_code FROM ai_messages WHERE conversation_id::text=$1 ORDER BY created_at,id LIMIT 50`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ConversationMessage{}
	for rows.Next() {
		var m ConversationMessage
		if err = rows.Scan(&m.ID, &m.Question, &m.Month, &m.Status, &m.Facts, &m.Response, &m.ErrorCode); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *ConversationRepository) BeginMessage(ctx context.Context, conversation, key, hash, question, month string, facts any) (string, *ConversationMessage, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ai_conversations WHERE id::text=$1)", conversation).Scan(&exists); err != nil {
		return "", nil, err
	}
	if !exists {
		return "", nil, ErrAINotFound
	}
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended(grace_owner_id()::text || $1,0))", "ai-message:"+conversation); err != nil {
		return "", nil, err
	}
	var old ConversationMessage
	var oldHash string
	err = tx.QueryRow(ctx, `SELECT id::text,request_hash,question,month,status,facts,response,error_code FROM ai_messages WHERE conversation_id::text=$1 AND request_key=$2`, conversation, key).Scan(&old.ID, &oldHash, &old.Question, &old.Month, &old.Status, &old.Facts, &old.Response, &old.ErrorCode)
	if err == nil {
		if oldHash != hash {
			return "", nil, ErrAIConflict
		}
		if old.Status == "pending" {
			return "", nil, ErrAIInProgress
		}
		return old.ID, &old, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", nil, err
	}
	var count int
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM ai_messages WHERE conversation_id::text=$1", conversation).Scan(&count); err != nil {
		return "", nil, err
	}
	if count >= 50 {
		return "", nil, errors.New("conversation_limit")
	}
	raw, err := json.Marshal(facts)
	if err != nil {
		return "", nil, err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO ai_messages(conversation_id,request_key,request_hash,question,month,facts) VALUES($1::uuid,$2,$3,$4,$5,$6) RETURNING id::text`, conversation, key, hash, question, month, raw).Scan(&id)
	if err != nil {
		return "", nil, err
	}
	return id, nil, tx.Commit(ctx)
}
func (r *ConversationRepository) Finish(ctx context.Context, id string, response any, code string) error {
	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	status := "completed"
	var errCode any
	if code != "" {
		status = "failed"
		errCode = code
		raw = nil
	}
	_, err = tx.Exec(ctx, "UPDATE ai_messages SET status=$2,response=$3,error_code=$4 WHERE id::text=$1 AND status='pending'", id, status, raw, errCode)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
