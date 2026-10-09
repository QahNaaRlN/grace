package assistant

import (
	"context"
	"errors"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageRepository uses owner-scoped transactions; only audited metering
// functions may serialize the global experimental allowance across owners.
type UsageRepository struct{ pool *database.TenantPool }

func NewUsageRepository(p *pgxpool.Pool) *UsageRepository {
	return &UsageRepository{database.ForUsers(p)}
}
func (r *UsageRepository) Reserve(ctx context.Context, policy ModelPolicy, inputBound, outputBound int, purpose string) (string, error) {
	amount, err := policy.ReservationMicros(inputBound, outputBound)
	if err != nil {
		return "", err
	}
	if purpose != "analysis" && purpose != "receipt" && purpose != "jev_category" {
		return "", errors.New("invalid_ai_purpose")
	}
	var id string
	err = r.pool.QueryRow(ctx, "SELECT grace_ai_reserve($1,$2,$3,$4)::text", amount, policy.Model, policy.Provider, purpose).Scan(&id)
	return id, err
}
func (r *UsageRepository) Settle(ctx context.Context, id string, response *LLMResponse) error {
	// A cancelled browser request does not cancel the persistence of upstream
	// billing evidence. The original principal remains in this bounded context.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var actual any
	var generation any
	if response != nil {
		actual = response.CostMicros
		generation = response.ID
	}
	var ignored string
	return r.pool.QueryRow(ctx, "SELECT grace_ai_settle($1::uuid,$2::bigint,$3::text)::text", id, actual, generation).Scan(&ignored)
}
func (r *UsageRepository) Remaining(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, "SELECT grace_ai_remaining()").Scan(&n)
	return n, err
}

type UsageMeter interface {
	Reserve(context.Context, ModelPolicy, int, int, string) (string, error)
	Settle(context.Context, string, *LLMResponse) error
}

func MeteredCompletion(ctx context.Context, provider ReasoningProvider, meter UsageMeter, in LLMRequest, purpose string) (LLMResponse, error) {
	id, err := meter.Reserve(ctx, in.Policy, in.InputTokenUpperBound, in.MaxOutputTokens, purpose)
	if err != nil {
		return LLMResponse{}, err
	}
	out, err := provider.Complete(ctx, in)
	if err != nil {
		if settleErr := meter.Settle(ctx, id, nil); settleErr != nil {
			return LLMResponse{}, errors.New("ai_usage_requires_review")
		}
		return LLMResponse{}, err
	}
	if err = meter.Settle(ctx, id, &out); err != nil {
		return LLMResponse{}, errors.New("ai_usage_requires_review")
	}
	return out, nil
}
