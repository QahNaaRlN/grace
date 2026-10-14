package categories

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
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

// Icons is the fixed set allowed for categories (SPEC-0042); it mirrors the
// CHECK constraint of migration 000019.
var Icons = []string{
	"shopping-cart", "utensils", "coffee", "car", "bus", "fuel", "home", "zap", "wifi", "smartphone",
	"heart-pulse", "pill", "shirt", "gift", "gamepad-2", "film", "music", "book-open", "graduation-cap",
	"baby", "paw-print", "plane", "dumbbell", "scissors", "wrench", "credit-card", "landmark", "receipt",
	"repeat", "briefcase", "banknote", "piggy-bank", "sparkles", "tag", "sofa", "cigarette",
}

var (
	ErrNotFound      = errors.New("category not found")
	ErrNeedsTarget   = errors.New("category has operations")
	ErrInvalidTarget = errors.New("invalid target category")
	ErrCountChanged  = errors.New("operation count changed")
	ErrDuplicateName = errors.New("duplicate category name")
)

type Category struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"`
	CreatedAt      time.Time `json:"createdAt"`
	Editable       bool      `json:"editable"`
	Icon           *string   `json:"icon"`
	Hidden         bool      `json:"hidden"`
	OperationCount int       `json:"operationCount"`
}

type Repository struct{ pool *database.TenantPool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: database.ForUsers(pool)} }

// Operation counts and hidden flags are owner-scoped by RLS on both tables.
const selectCategory = `
	SELECT c.id::text, c.name, c.kind::text, c.created_at, c.owner_id IS NOT NULL, c.icon,
		EXISTS (SELECT 1 FROM category_hidden h WHERE h.category_id = c.id),
		(SELECT count(*) FROM financial_events e WHERE e.category_id = c.id)::int
	FROM categories c`

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func scanCategory(row pgx.Row) (Category, error) {
	var item Category
	err := row.Scan(&item.ID, &item.Name, &item.Kind, &item.CreatedAt, &item.Editable, &item.Icon, &item.Hidden, &item.OperationCount)
	return item, err
}

func (repo *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := repo.pool.Query(ctx, selectCategory+` ORDER BY c.name`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()
	items := make([]Category, 0)
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func get(ctx context.Context, q queryer, id string) (Category, error) {
	item, err := scanCategory(q.QueryRow(ctx, selectCategory+` WHERE c.id = $1`, id))
	if isMissing(err) {
		return item, ErrNotFound
	}
	return item, err
}

// Save creates a personal category or updates the name and/or icon of one.
// An empty icon clears it.
func (repo *Repository) Save(ctx context.Context, id string, name, icon *string) (Category, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Category{}, err
	}
	defer tx.Rollback(ctx)
	if id == "" {
		err = tx.QueryRow(ctx, `INSERT INTO categories(name,kind,owner_id,icon) VALUES($1,'expense',grace_owner_id(),$2) RETURNING id::text`, *name, nilIfEmpty(icon)).Scan(&id)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE categories SET name=COALESCE($2,name), icon=CASE WHEN $3 THEN $4 ELSE icon END
			WHERE id=$1 AND owner_id=grace_owner_id()`, id, name, icon != nil, nilIfEmpty(icon))
		if err == nil && tag.RowsAffected() == 0 {
			err = ErrNotFound
		}
	}
	if err != nil {
		return Category{}, mapError(err)
	}
	item, err := get(ctx, tx, id)
	if err != nil {
		return Category{}, err
	}
	return item, tx.Commit(ctx)
}

// Delete removes a personal category. Operations that use it move to target in
// the same transaction; expected guards against operations added meanwhile.
func (repo *Repository) Delete(ctx context.Context, id, target string, expected int) (int, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var kind string
	err = tx.QueryRow(ctx, `SELECT kind::text FROM categories WHERE id=$1 AND owner_id=grace_owner_id() FOR UPDATE`, id).Scan(&kind)
	if isMissing(err) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*)::int FROM financial_events WHERE category_id=$1`, id).Scan(&count); err != nil {
		return 0, err
	}
	if count != expected {
		return 0, ErrCountChanged
	}
	if count > 0 && target == "" {
		return 0, ErrNeedsTarget
	}
	if target != "" {
		var targetKind string
		err = tx.QueryRow(ctx, `SELECT kind::text FROM categories WHERE id=$1`, target).Scan(&targetKind)
		if isMissing(err) || (err == nil && (targetKind != kind || target == id)) {
			return 0, ErrInvalidTarget
		}
		if err != nil {
			return 0, err
		}
	}
	if count > 0 {
		tag, err := tx.Exec(ctx, `UPDATE financial_events SET category_id=$2 WHERE category_id=$1`, id, target)
		if err != nil {
			return 0, mapError(err)
		}
		if int(tag.RowsAffected()) != count {
			return 0, ErrCountChanged
		}
	}
	// SPEC-0047: subscriptions follow their operations to the target; without
	// a target the foreign key clears their category.
	if target != "" {
		if _, err := tx.Exec(ctx, `UPDATE subscriptions SET category_id=$2, revision=revision+1 WHERE category_id=$1`, id, target); err != nil {
			return 0, mapError(err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM categories WHERE id=$1 AND owner_id=grace_owner_id()`, id); err != nil {
		return 0, mapError(err)
	}
	return count, tx.Commit(ctx)
}

// SetHidden hides or shows a category in the owner's choices for new operations.
func (repo *Repository) SetHidden(ctx context.Context, id string, hidden bool) (Category, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Category{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := get(ctx, tx, id); err != nil {
		return Category{}, err
	}
	if hidden {
		_, err = tx.Exec(ctx, `INSERT INTO category_hidden(owner_id,category_id) VALUES(grace_owner_id(),$1) ON CONFLICT DO NOTHING`, id)
	} else {
		_, err = tx.Exec(ctx, `DELETE FROM category_hidden WHERE category_id=$1`, id)
	}
	if err != nil {
		return Category{}, err
	}
	item, err := get(ctx, tx, id)
	if err != nil {
		return Category{}, err
	}
	return item, tx.Commit(ctx)
}

func nilIfEmpty(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

func isMissing(err error) bool {
	var dbErr *pgconn.PgError
	return errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &dbErr) && dbErr.Code == "22P02")
}

func mapError(err error) error {
	var dbErr *pgconn.PgError
	switch {
	case isMissing(err):
		return ErrNotFound
	case errors.As(err, &dbErr) && dbErr.Code == "23505":
		return ErrDuplicateName
	case errors.As(err, &dbErr) && (dbErr.Code == "23514" || dbErr.Code == "23503"):
		return ErrInvalidTarget
	}
	return err
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := handler.repo.List(r.Context())
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить категории.")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "icons": Icons})
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) { handler.save(w, r, "") }
func (handler *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	handler.save(w, r, chi.URLParam(r, "categoryId"))
}

func (handler *Handler) save(w http.ResponseWriter, r *http.Request, id string) {
	var request struct {
		Name *string `json:"name"`
		Icon *string `json:"icon"`
	}
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 80 || strings.ContainsAny(name, "\r\n\t") {
			web.WriteError(w, 422, "validation_error", "Название должно содержать от 1 до 80 символов в одной строке.")
			return
		}
		request.Name = &name
	}
	if (id == "" && request.Name == nil) || (request.Name == nil && request.Icon == nil) {
		web.WriteError(w, 422, "validation_error", "Укажите название или иконку категории.")
		return
	}
	if request.Icon != nil && *request.Icon != "" && !slices.Contains(Icons, *request.Icon) {
		web.WriteError(w, 422, "validation_error", "Такой иконки нет в наборе.")
		return
	}
	item, err := handler.repo.Save(r.Context(), id, request.Name, request.Icon)
	if handler.writeError(w, err, "Не удалось сохранить категорию.") {
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	web.WriteJSON(w, status, item)
}

// DeletePreview tells how many operations a deletion would move.
func (handler *Handler) DeletePreview(w http.ResponseWriter, r *http.Request) {
	item, err := get(r.Context(), handler.repo.pool, chi.URLParam(r, "categoryId"))
	if err == nil && !item.Editable {
		err = ErrNotFound
	}
	if handler.writeError(w, err, "Не удалось проверить категорию.") {
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"operations": item.OperationCount})
}

func (handler *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	expected, err := strconv.Atoi(query.Get("expectedOperations"))
	if err != nil || expected < 0 {
		web.WriteError(w, 422, "validation_error", "Передайте expectedOperations из предпросмотра удаления.")
		return
	}
	moved, err := handler.repo.Delete(r.Context(), chi.URLParam(r, "categoryId"), strings.TrimSpace(query.Get("moveTo")), expected)
	if handler.writeError(w, err, "Не удалось удалить категорию.") {
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"movedOperations": moved})
}

func (handler *Handler) Hide(w http.ResponseWriter, r *http.Request)   { handler.setHidden(w, r, true) }
func (handler *Handler) Unhide(w http.ResponseWriter, r *http.Request) { handler.setHidden(w, r, false) }

func (handler *Handler) setHidden(w http.ResponseWriter, r *http.Request, hidden bool) {
	item, err := handler.repo.SetHidden(r.Context(), chi.URLParam(r, "categoryId"), hidden)
	if handler.writeError(w, err, "Не удалось изменить видимость категории.") {
		return
	}
	web.WriteJSON(w, http.StatusOK, item)
}

func (handler *Handler) writeError(w http.ResponseWriter, err error, fallback string) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		web.WriteError(w, 404, "not_found", "Категория недоступна для изменения.")
	case errors.Is(err, ErrDuplicateName):
		web.WriteError(w, 409, "conflict", "Такая категория уже существует.")
	case errors.Is(err, ErrCountChanged):
		web.WriteError(w, 409, "operations_changed", "Количество операций изменилось. Откройте удаление заново.")
	case errors.Is(err, ErrNeedsTarget):
		web.WriteError(w, 422, "move_target_required", "Выберите категорию, в которую перенести операции.")
	case errors.Is(err, ErrInvalidTarget):
		web.WriteError(w, 422, "invalid_move_target", "Перенести операции можно только в другую доступную категорию расходов.")
	default:
		web.WriteError(w, 500, "internal_error", fallback)
	}
	return true
}
