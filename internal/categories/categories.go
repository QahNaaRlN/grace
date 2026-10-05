package categories

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"createdAt"`
}

type Repository struct{ pool *database.TenantPool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: database.ForUsers(pool)} }

func (repo *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := repo.pool.Query(ctx, `
		SELECT id::text, name, kind::text, created_at
		FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()
	items := make([]Category, 0)
	for rows.Next() {
		var item Category
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := handler.repo.List(r.Context())
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить категории.")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
