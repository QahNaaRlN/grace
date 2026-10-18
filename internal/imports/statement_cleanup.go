package imports

import (
	"context"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunStatementCleanup is an internal worker, not an admin HTTP capability. Its
// privileged query enumerates only expiry identifiers; deletion uses ordinary
// owner-scoped transactions and the same locks as confirmation/extraction.
func RunStatementCleanup(ctx context.Context, pool *pgxpool.Pool) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			_ = CleanupExpiredStatements(cleanupCtx, pool)
			cancel()
		}
	}
}
func CleanupExpiredStatements(ctx context.Context, pool *pgxpool.Pool) error {
	rows, e := pool.Query(ctx, "SELECT id::text,owner_id::text FROM statement_sources WHERE deleted_at IS NULL AND retention='delete_after_confirmation' AND expires_at<=now() ORDER BY expires_at LIMIT 100")
	if e != nil {
		return e
	}
	type item struct{ id, owner string }
	items := []item{}
	for rows.Next() {
		var it item
		if e = rows.Scan(&it.id, &it.owner); e != nil {
			rows.Close()
			return e
		}
		items = append(items, it)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	repo := NewRepository(pool)
	for _, it := range items {
		ownerCtx := access.WithPrincipal(ctx, access.Principal{ID: it.owner, Role: "user"})
		if _, e = repo.Statement(ownerCtx, it.id); e != nil {
			return e
		}
	}
	return nil
}
