package categories

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"createdAt"`
	Editable  bool      `json:"editable"`
}

type Repository struct{ pool *database.TenantPool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: database.ForUsers(pool)} }

func (repo *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := repo.pool.Query(ctx, `
		SELECT id::text, name, kind::text, created_at, owner_id IS NOT NULL
		FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()
	items := make([]Category, 0)
	for rows.Next() {
		var item Category
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.CreatedAt, &item.Editable); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *Repository) Save(ctx context.Context, id, name string) (Category, error) {
	var item Category
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return item, err
	}
	defer tx.Rollback(ctx)
	if id == "" {
		err = tx.QueryRow(ctx, `INSERT INTO categories(name,kind,owner_id) VALUES($1,'expense',grace_owner_id()) RETURNING id::text,name,kind::text,created_at,true`, name).Scan(&item.ID, &item.Name, &item.Kind, &item.CreatedAt, &item.Editable)
	} else {
		err = tx.QueryRow(ctx, `UPDATE categories SET name=$2 WHERE id=$1 AND owner_id=grace_owner_id() RETURNING id::text,name,kind::text,created_at,true`, id, name).Scan(&item.ID, &item.Name, &item.Kind, &item.CreatedAt, &item.Editable)
	}
	if err != nil {
		return item, err
	}
	return item, tx.Commit(ctx)
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) { handler.save(w, r, "") }
func (handler *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	handler.save(w, r, chi.URLParam(r, "categoryId"))
}
func (handler *Handler) save(w http.ResponseWriter, r *http.Request, id string) {
	var request struct {
		Name string `json:"name"`
	}
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	name := strings.TrimSpace(request.Name)
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 80 || strings.ContainsAny(name, "\r\n\t") {
		web.WriteError(w, 422, "validation_error", "Название должно содержать от 1 до 80 символов в одной строке.")
		return
	}
	item, err := handler.repo.Save(r.Context(), id, name)
	var dbErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &dbErr) && dbErr.Code == "22P02") {
		web.WriteError(w, 404, "not_found", "Категория недоступна для изменения.")
		return
	}
	if errors.As(err, &dbErr) && dbErr.Code == "23505" {
		web.WriteError(w, 409, "conflict", "Такая категория уже существует.")
		return
	}
	if err != nil {
		web.WriteError(w, 500, "internal_error", "Не удалось сохранить категорию.")
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	web.WriteJSON(w, status, item)
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
