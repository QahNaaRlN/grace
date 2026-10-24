package documentqualification

import (
	"context"
	"errors"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

// SQLBudget calls only the existing shared allowance functions. No financial
// query, user lookup, allowance creation or budget increase is available here.
type SQLBudget struct{ tenant *database.TenantPool }

func NewSQLBudget(pool *pgxpool.Pool) *SQLBudget { return &SQLBudget{database.ForUsers(pool)} }
func (b *SQLBudget) Remaining(ctx context.Context) (int64, error) {
	var n int64
	e := b.tenant.QueryRow(ctx, "SELECT grace_ai_remaining()").Scan(&n)
	return n, e
}
func (b *SQLBudget) Reserve(ctx context.Context, p assistant.ModelPolicy, input, output int, purpose string) (string, error) {
	n, e := p.ReservationMicros(input, output)
	if e != nil || n <= 0 || n > MaximumCallMicros || (purpose != "statement" && purpose != "receipt") {
		return "", errors.New("budget_refused")
	}
	var id string
	e = b.tenant.QueryRow(ctx, "SELECT grace_ai_reserve($1,$2,$3,$4)::text", n, p.Model, p.Provider, purpose).Scan(&id)
	return id, e
}
func (b *SQLBudget) Allowed(ctx context.Context, id string) (bool, error) {
	var yes bool
	e := b.tenant.QueryRow(ctx, "SELECT grace_statement_call_allowed($1::uuid)", id).Scan(&yes)
	return yes, e
}
func (b *SQLBudget) Settle(ctx context.Context, id string, out *assistant.LLMResponse) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var n any
	var generation any
	if out != nil {
		n = out.CostMicros
		generation = out.ID
	}
	var result string
	return b.tenant.QueryRow(ctx, "SELECT grace_ai_settle($1,$2::bigint,$3)::text", id, n, generation).Scan(&result)
}
