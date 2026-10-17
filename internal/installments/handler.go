package installments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct{ pool *database.TenantPool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{database.ForUsers(pool)} }
func (h *Handler) Routes(r chi.Router) {
	r.Get("/installment-plans", h.List)
	r.Post("/installment-plans", h.Create)
	r.Get("/installment-plans/{planId}", h.Get)
	r.Post("/installment-plans/{planId}/schedule-drafts", h.CreateDraft)
	r.Put("/installment-plans/{planId}/schedule-drafts/{draftId}", h.UpdateDraft)
	r.Post("/installment-plans/{planId}/schedule-drafts/{draftId}/apply", h.ApplyDraft)
	r.Get("/financial-events/{eventId}/installment-allocation", h.GetAllocation)
	r.Put("/financial-events/{eventId}/installment-allocation", h.SaveAllocation)
	r.Delete("/financial-events/{eventId}/installment-allocation", h.SaveAllocation)
}
func respondError(w http.ResponseWriter, err error) {
	status, code, msg := 500, "internal_error", "Не удалось обработать рассрочку."
	var pg *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, code, msg = 404, "not_found", "Связанный объект не найден."
	case errors.Is(err, ErrInvalid), errors.Is(err, financialevents.ErrInvalid):
		status, code, msg = 422, "validation_error", ErrInvalid.Error()
	case errors.Is(err, ErrConflict):
		status, code, msg = 409, "conflict", ErrConflict.Error()
	case errors.As(err, &pg):
		switch pg.Code {
		case "23503", "P0002":
			status, code, msg = 404, "not_found", "Связанный объект не найден."
		case "23505", "40001", "40P01":
			status, code, msg = 409, "conflict", ErrConflict.Error()
		case "22P02", "23514", "22001":
			status, code, msg = 422, "validation_error", ErrInvalid.Error()
		}
	}
	web.WriteError(w, status, code, msg)
}
func hashOf(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// write serializes owner ledger changes, persists the response with the write,
// and replays it before checking revisions (including after a lost response).
func (h *Handler) write(w http.ResponseWriter, r *http.Request, body any, fn func(context.Context, pgx.Tx) (any, error)) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 160 {
		web.WriteError(w, 400, "idempotency_key_required", "Передайте Idempotency-Key (до 160 символов).")
		return
	}
	ctx := r.Context()
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		respondError(w, err)
		return
	}
	defer tx.Rollback(ctx)
	err = financialevents.LockOwner(ctx, tx)
	if err != nil {
		respondError(w, err)
		return
	}
	hash := hashOf([]any{r.Method, r.URL.Path, body})
	var oldHash string
	var response json.RawMessage
	err = tx.QueryRow(ctx, `SELECT request_hash,response FROM installment_commands WHERE key=$1`, key).Scan(&oldHash, &response)
	if err == nil {
		if hash != oldHash {
			respondError(w, ErrConflict)
			return
		}
		web.WriteJSON(w, 200, response)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		respondError(w, err)
		return
	}
	result, err := fn(ctx, tx)
	if err != nil {
		respondError(w, err)
		return
	}
	response, err = json.Marshal(result)
	if err != nil {
		respondError(w, err)
		return
	}
	_, err = tx.Exec(ctx, `INSERT INTO installment_commands(key,request_hash,response) VALUES($1,$2,$3)`, key, hash, response)
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		respondError(w, err)
		return
	}
	web.WriteJSON(w, 200, response)
}
func (h *Handler) read(w http.ResponseWriter, r *http.Request, fn func(context.Context, pgx.Tx, string) (any, error)) {
	on := r.URL.Query().Get("on")
	if on == "" {
		on = Today()
	}
	if !validDay(on) {
		respondError(w, ErrInvalid)
		return
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		respondError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	out, err := fn(r.Context(), tx, on)
	if err != nil {
		respondError(w, err)
		return
	}
	web.WriteJSON(w, 200, out)
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	h.read(w, r, func(ctx context.Context, tx pgx.Tx, on string) (any, error) {
		rows, e := tx.Query(ctx, `SELECT id::text FROM installment_plans WHERE observed_on<=$1::date ORDER BY opened_on DESC,id`, on)
		if e != nil {
			return nil, e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return nil, e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		plans := []Plan{}
		for _, id := range ids {
			p, e := readPlan(ctx, tx, id, on)
			if e != nil {
				return nil, e
			}
			plans = append(plans, p)
		}
		return map[string]any{"items": plans, "on": on}, nil
	})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.read(w, r, func(ctx context.Context, tx pgx.Tx, on string) (any, error) {
		return readPlan(ctx, tx, chi.URLParam(r, "planId"), on)
	})
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var c CreateRequest
	if !web.DecodeJSON(w, r, &c) {
		return
	}
	if err := c.normalize(); err != nil {
		respondError(w, err)
		return
	}
	h.write(w, r, c, func(ctx context.Context, tx pgx.Tx) (any, error) {
		return createPlan(ctx, tx, c, hashOf(c), r.Header.Get("Idempotency-Key"))
	})
}
func (h *Handler) GetAllocation(w http.ResponseWriter, r *http.Request) {
	h.read(w, r, func(ctx context.Context, tx pgx.Tx, _ string) (any, error) {
		return readAllocation(ctx, tx, chi.URLParam(r, "eventId"))
	})
}
func (h *Handler) SaveAllocation(w http.ResponseWriter, r *http.Request) {
	var c AllocationRequest
	if !web.DecodeJSON(w, r, &c) {
		return
	}
	if r.Method == "DELETE" {
		c.Items = []Allocation{}
	}
	h.write(w, r, c, func(ctx context.Context, tx pgx.Tx) (any, error) {
		return saveAllocation(ctx, tx, chi.URLParam(r, "eventId"), c)
	})
}
