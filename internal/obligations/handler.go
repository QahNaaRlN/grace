package obligations

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
	asOf, ok := requestAsOf(w, r)
	if !ok {
		return
	}
	items, err := handler.repo.List(r.Context(), asOf, 100)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить обязательства.")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"asOf": asOf, "items": items})
}

func (handler *Handler) Get(w http.ResponseWriter, r *http.Request) {
	asOf, ok := requestAsOf(w, r)
	if !ok {
		return
	}
	obligation, err := handler.repo.Get(r.Context(), chi.URLParam(r, "obligationId"), asOf)
	if errors.Is(err, ErrNotFound) {
		web.WriteError(w, http.StatusNotFound, "not_found", "Обязательство не найдено.")
		return
	}
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить обязательство.")
		return
	}
	web.WriteJSON(w, http.StatusOK, obligation)
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request createRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	if !normalizeAndValidateCreate(&request) {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Проверьте название, тип, валюту, общую сумму и график.")
		return
	}
	key, ok := requestIdempotencyKey(w, r)
	if !ok {
		return
	}
	asOf := time.Now().UTC().Format(dateLayout)
	obligation, created, err := handler.repo.Create(r.Context(), CreateCommand{
		Name: request.Name, Kind: request.Kind, Currency: request.Currency,
		TotalMinor: request.TotalMinor, Payments: request.Payments,
		IdempotencyKey: key, RequestHash: hashRequest(request), AsOf: asOf,
	})
	if errors.Is(err, ErrIdempotencyConflict) {
		web.WriteError(w, http.StatusConflict, "idempotency_conflict", "Этот Idempotency-Key уже использован с другим запросом.")
		return
	}
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось сохранить обязательство.")
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	web.WriteJSON(w, status, obligation)
}

func (handler *Handler) CreateSettlement(w http.ResponseWriter, r *http.Request) {
	var request settlementRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	request.FinancialEventID = strings.TrimSpace(request.FinancialEventID)
	obligationID := strings.TrimSpace(chi.URLParam(r, "obligationId"))
	paymentID := strings.TrimSpace(chi.URLParam(r, "paymentId"))
	if request.FinancialEventID == "" || obligationID == "" || paymentID == "" {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Укажите обязательство, платёж графика и финансовое событие.")
		return
	}
	key, ok := requestIdempotencyKey(w, r)
	if !ok {
		return
	}
	hashPayload := struct {
		ObligationID     string `json:"obligationId"`
		PaymentID        string `json:"paymentId"`
		FinancialEventID string `json:"financialEventId"`
	}{obligationID, paymentID, request.FinancialEventID}
	settlement, created, err := handler.repo.CreateSettlement(r.Context(), SettlementCommand{
		ObligationID: obligationID, PaymentID: paymentID,
		FinancialEventID: request.FinancialEventID, IdempotencyKey: key,
		RequestHash: hashRequest(hashPayload),
	})
	if errors.Is(err, ErrNotFound) {
		web.WriteError(w, http.StatusNotFound, "not_found", "Обязательство или платёж графика не найден.")
		return
	}
	if errors.Is(err, ErrIdempotencyConflict) {
		web.WriteError(w, http.StatusConflict, "idempotency_conflict", "Этот Idempotency-Key уже использован с другим запросом.")
		return
	}
	if errors.Is(err, ErrEventAlreadySettled) {
		web.WriteError(w, http.StatusConflict, "financial_event_settled", "Этот платёж уже связан с графиком.")
		return
	}
	if errors.Is(err, ErrInvalid) {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось связать платёж с графиком.")
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	web.WriteJSON(w, status, settlement)
}

func requestAsOf(w http.ResponseWriter, r *http.Request) (string, bool) {
	asOf, err := ParseAsOf(r.URL.Query().Get("asOf"), time.Now())
	if err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "asOf должен быть календарной датой YYYY-MM-DD.")
		return "", false
	}
	return asOf, true
}

func requestIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		web.WriteError(w, http.StatusBadRequest, "idempotency_key_required", "Передайте заголовок Idempotency-Key.")
		return "", false
	}
	if len(key) > 160 {
		web.WriteError(w, http.StatusBadRequest, "invalid_idempotency_key", "Idempotency-Key слишком длинный.")
		return "", false
	}
	return key, true
}

func normalizeAndValidateCreate(request *createRequest) bool {
	request.Name = strings.TrimSpace(request.Name)
	request.Kind = strings.TrimSpace(request.Kind)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	if request.Name == "" || len(request.Name) > 200 || request.TotalMinor <= 0 ||
		!validKind(request.Kind) || !validCurrency(request.Currency) ||
		len(request.Payments) == 0 || len(request.Payments) > 600 {
		return false
	}
	var sum int64
	var previous time.Time
	for index := range request.Payments {
		payment := &request.Payments[index]
		payment.DueOn = strings.TrimSpace(payment.DueOn)
		parsed, err := time.Parse(dateLayout, payment.DueOn)
		if err != nil || parsed.Format(dateLayout) != payment.DueOn || payment.AmountMinor <= 0 ||
			(index > 0 && !parsed.After(previous)) || sum > math.MaxInt64-payment.AmountMinor {
			return false
		}
		sum += payment.AmountMinor
		previous = parsed
	}
	return sum == request.TotalMinor
}

func validKind(value string) bool {
	return value == "loan" || value == "installment" || value == "other_fixed"
}

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

func hashRequest(request any) string {
	payload, _ := json.Marshal(request)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
