package database

import (
	"context"
	"errors"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrMissingPrincipal = errors.New("financial access requires a principal")

// TenantPool scopes every financial operation, including tools, to a transaction.
// SET LOCAL prevents an identity from surviving reuse of a pooled connection.
type TenantPool struct{ pool *pgxpool.Pool }

func ForUsers(pool *pgxpool.Pool) *TenantPool { return &TenantPool{pool: pool} }

func (p *TenantPool) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	principal, ok := access.Current(ctx)
	if !ok {
		return nil, ErrMissingPrincipal
	}
	tx, err := p.pool.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, "SET LOCAL ROLE grace_financial"); err == nil {
		_, err = tx.Exec(ctx, "SELECT set_config('grace.owner_id', $1, true)", principal.ID)
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

func (p *TenantPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	tx, err := p.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return &tenantRows{Rows: rows, tx: tx}, nil
}

type tenantRows struct {
	pgx.Rows
	tx pgx.Tx
}

func (r *tenantRows) Close() { r.Rows.Close(); _ = r.tx.Rollback(context.Background()) }

func (p *TenantPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	tx, err := p.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return &tenantRow{err: err}
	}
	return &tenantRow{row: tx.QueryRow(ctx, sql, args...), tx: tx, ctx: ctx}
}

type tenantRow struct {
	row pgx.Row
	tx  pgx.Tx
	ctx context.Context
	err error
}

func (r *tenantRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	defer func() { _ = r.tx.Rollback(context.Background()) }()
	if err := r.row.Scan(dest...); err != nil {
		return err
	}
	return r.tx.Commit(r.ctx)
}
