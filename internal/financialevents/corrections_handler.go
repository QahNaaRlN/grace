package financialevents

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
)

type correctionRequest struct {
	Revision             int              `json:"revision"`
	Merchant             *string          `json:"merchant"`
	Description          *string          `json:"description"`
	CategoryID           *json.RawMessage `json:"categoryId"`
	OccurredAt           *string          `json:"occurredAt"`
	OccurredDate         *string          `json:"occurredDate"`
	Kind                 *string          `json:"kind"`
	AmountMinor          *int64           `json:"amountMinor"`
	AccountID            *string          `json:"accountId"`
	DestinationAccountID *string          `json:"destinationAccountId"`
}

// Correct applies a correction to one operation (SPEC-0041).
func (handler *Handler) Correct(w http.ResponseWriter, r *http.Request) {
	var request correctionRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	key, ok := requireKey(w, r)
	if !ok {
		return
	}
	correction := Correction{Revision: request.Revision, IdempotencyKey: key, Merchant: request.Merchant, Description: request.Description,
		OccurredDate: request.OccurredDate, Kind: request.Kind, AmountMinor: request.AmountMinor,
		AccountID: request.AccountID, DestinationAccountID: request.DestinationAccountID}
	if request.CategoryID != nil {
		// null clears the category; a string sets it.
		var id *string
		if err := json.Unmarshal(*request.CategoryID, &id); err != nil {
			web.WriteError(w, 422, "validation_error", "Категория должна быть идентификатором или null.")
			return
		}
		correction.CategoryID, correction.ClearCategory = id, id == nil
	}
	if request.OccurredAt != nil {
		at, err := time.Parse(time.RFC3339, *request.OccurredAt)
		if err != nil {
			web.WriteError(w, 422, "validation_error", "Дата операции должна быть в формате RFC 3339.")
			return
		}
		correction.OccurredAt = &at
	}
	event, err := handler.repo.Correct(r.Context(), chi.URLParam(r, "eventId"), correction)
	if writeCorrectionError(w, err) {
		return
	}
	web.WriteJSON(w, http.StatusOK, event)
}

func (handler *Handler) Void(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Revision int    `json:"revision"`
		Reason   string `json:"reason"`
	}
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	key, ok := requireKey(w, r)
	if !ok {
		return
	}
	event, err := handler.repo.Void(r.Context(), chi.URLParam(r, "eventId"), request.Revision, request.Reason, key)
	if writeCorrectionError(w, err) {
		return
	}
	web.WriteJSON(w, http.StatusOK, event)
}

func (handler *Handler) History(w http.ResponseWriter, r *http.Request) {
	items, err := handler.repo.History(r.Context(), chi.URLParam(r, "eventId"))
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить историю операции.")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func requireKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := idempotencyKey(r)
	if key == "" || len(key) > 160 {
		web.WriteError(w, http.StatusBadRequest, "idempotency_key_required", "Передайте заголовок Idempotency-Key до 160 символов.")
		return "", false
	}
	return key, true
}

func writeCorrectionError(w http.ResponseWriter, err error) bool {
	var linked *LinkError
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		web.WriteError(w, 404, "not_found", "Операция не найдена.")
	case errors.Is(err, ErrVoided):
		web.WriteError(w, 409, "voided", "Операция отменена и больше не меняется.")
	case errors.Is(err, ErrRevisionConflict):
		web.WriteError(w, 409, "revision_conflict", "Операция уже изменилась. Откройте её заново.")
	case errors.As(err, &linked):
		web.WriteJSON(w, 409, map[string]any{"code": "linked", "links": linked.Links,
			"message": "Сначала снимите связь: " + strings.Join(linked.Links, ", ") + "."})
	case errors.Is(err, ErrInvalid):
		web.WriteError(w, 422, "validation_error", "Проверьте сумму, счёт, вид, категорию и дату операции.")
	default:
		web.WriteError(w, 500, "internal_error", "Не удалось изменить операцию.")
	}
	return true
}
