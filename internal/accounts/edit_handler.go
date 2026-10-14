package accounts

import (
	"errors"
	"net/http"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
)

// Update changes account details (SPEC-0040).
func (handler *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var request Update
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	item, err := handler.repo.Update(r.Context(), chi.URLParam(r, "accountId"), request)
	if writeAccountError(w, err) {
		return
	}
	web.WriteJSON(w, http.StatusOK, item)
}

func (handler *Handler) AddCard(w http.ResponseWriter, r *http.Request) {
	var request CardChange
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	item, err := handler.repo.AddCard(r.Context(), chi.URLParam(r, "accountId"), request)
	if writeAccountError(w, err) {
		return
	}
	web.WriteJSON(w, http.StatusCreated, item)
}

func (handler *Handler) UpdateCard(w http.ResponseWriter, r *http.Request) {
	var request CardChange
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	item, err := handler.repo.UpdateCard(r.Context(), chi.URLParam(r, "accountId"), chi.URLParam(r, "cardId"), request)
	if writeAccountError(w, err) {
		return
	}
	web.WriteJSON(w, http.StatusOK, item)
}

type reconciliationInput struct {
	AsOf        string `json:"asOf"`
	StatedMinor *int64 `json:"statedMinor"`
	PreviewHash string `json:"previewHash"`
}

func (input reconciliationInput) parse(w http.ResponseWriter) (ReconcileRequest, bool) {
	at, err := time.Parse(time.RFC3339, input.AsOf)
	if err != nil || input.StatedMinor == nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Укажите остаток из банка и момент в формате RFC 3339.")
		return ReconcileRequest{}, false
	}
	return ReconcileRequest{AsOf: at, StatedMinor: *input.StatedMinor, PreviewHash: input.PreviewHash}, true
}

func (handler *Handler) PreviewReconciliation(w http.ResponseWriter, r *http.Request) {
	var input reconciliationInput
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	request, ok := input.parse(w)
	if !ok {
		return
	}
	result, err := handler.repo.PreviewReconciliation(r.Context(), chi.URLParam(r, "accountId"), request.AsOf, request.StatedMinor)
	if writeAccountError(w, err) {
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

func (handler *Handler) Reconcile(w http.ResponseWriter, r *http.Request) {
	var input reconciliationInput
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	request, ok := input.parse(w)
	if !ok {
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" || len(key) > 160 {
		web.WriteError(w, http.StatusBadRequest, "idempotency_key_required", "Передайте заголовок Idempotency-Key до 160 символов.")
		return
	}
	reconciliation, account, err := handler.repo.Reconcile(r.Context(), chi.URLParam(r, "accountId"), key, request)
	if writeAccountError(w, err) {
		return
	}
	web.WriteJSON(w, http.StatusCreated, map[string]any{"reconciliation": reconciliation, "account": account})
}

func (handler *Handler) Reconciliations(w http.ResponseWriter, r *http.Request) {
	items, err := handler.repo.Reconciliations(r.Context(), chi.URLParam(r, "accountId"))
	if writeAccountError(w, err) {
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func writeAccountError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		web.WriteError(w, http.StatusNotFound, "not_found", "Счёт или карта не найдены.")
	case errors.Is(err, ErrRevisionConflict):
		web.WriteError(w, http.StatusConflict, "revision_conflict", "Счёт уже изменился. Откройте его заново.")
	case errors.Is(err, ErrHasOperations):
		web.WriteError(w, http.StatusConflict, "has_operations", "Тип счёта нельзя менять, когда по нему есть операции.")
	case errors.Is(err, ErrStalePreview):
		web.WriteError(w, http.StatusConflict, "stale_preview", "Операции по счёту изменились. Проверьте сверку ещё раз.")
	case errors.Is(err, ErrInvalid):
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Проверьте название, тип, карту, остаток и момент сверки (не в будущем).")
	default:
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось изменить счёт.")
	}
	return true
}
