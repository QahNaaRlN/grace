package receipts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := handler.repo.List(r.Context(), 100)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить чеки.")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (handler *Handler) Get(w http.ResponseWriter, r *http.Request) {
	receipt, err := handler.repo.Get(r.Context(), chi.URLParam(r, "receiptId"))
	if errors.Is(err, ErrNotFound) {
		web.WriteError(w, http.StatusNotFound, "not_found", "Чек не найден.")
		return
	}
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить чек.")
		return
	}
	web.WriteJSON(w, http.StatusOK, receipt)
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request createRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	total, issuedAt, ok := normalizeAndValidate(&request)
	if !ok {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Проверьте покупку, продавца, дату и позиции чека.")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		web.WriteError(w, http.StatusBadRequest, "idempotency_key_required", "Передайте заголовок Idempotency-Key.")
		return
	}
	if len(idempotencyKey) > 160 {
		web.WriteError(w, http.StatusBadRequest, "invalid_idempotency_key", "Idempotency-Key слишком длинный.")
		return
	}

	receipt, created, err := handler.repo.Create(r.Context(), Command{
		PurchaseEventID: request.PurchaseEventID, Seller: request.Seller,
		IssuedAt: issuedAt, FiscalDocumentNumber: request.FiscalDocumentNumber,
		Items: request.Items, TotalMinor: total, IdempotencyKey: idempotencyKey,
		RequestHash: hashRequest(request),
	})
	if errors.Is(err, ErrIdempotencyConflict) {
		web.WriteError(w, http.StatusConflict, "idempotency_conflict", "Этот Idempotency-Key уже использован с другим запросом.")
		return
	}
	if errors.Is(err, ErrPurchaseHasReceipt) {
		web.WriteError(w, http.StatusConflict, "receipt_exists", "Для этой покупки уже сохранён чек.")
		return
	}
	if errors.Is(err, ErrInvalid) {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось сохранить чек.")
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	web.WriteJSON(w, status, receipt)
}

func normalizeAndValidate(request *createRequest) (int64, time.Time, bool) {
	request.PurchaseEventID = strings.TrimSpace(request.PurchaseEventID)
	request.Seller = strings.TrimSpace(request.Seller)
	request.FiscalDocumentNumber = strings.TrimSpace(request.FiscalDocumentNumber)
	issuedAt, err := time.Parse(time.RFC3339, request.IssuedAt)
	if err != nil || request.PurchaseEventID == "" || request.Seller == "" ||
		len(request.Seller) > 200 || len(request.FiscalDocumentNumber) > 160 ||
		len(request.Items) == 0 || len(request.Items) > 200 {
		return 0, time.Time{}, false
	}
	var total int64
	for index := range request.Items {
		item := &request.Items[index]
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || len(item.Name) > 300 || item.QuantityMilli <= 0 ||
			item.UnitPriceMinor <= 0 || item.LineTotalMinor <= 0 ||
			total > math.MaxInt64-item.LineTotalMinor {
			return 0, time.Time{}, false
		}
		total += item.LineTotalMinor
	}
	return total, issuedAt, true
}

func hashRequest(request createRequest) string {
	payload, _ := json.Marshal(request)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
