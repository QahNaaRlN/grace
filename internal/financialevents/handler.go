package financialevents

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	category, from, to := r.URL.Query().Get("categoryId"), r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if (from != "" && !ValidCalendarDate(from)) || (to != "" && !ValidCalendarDate(to)) || (from != "" && to != "" && from >= to) {
		web.WriteError(w, 422, "validation_error", "Проверьте даты: конец периода не включается и должен быть позже начала.")
		return
	}
	items, err := handler.repo.ListFiltered(r.Context(), 100, category, from, to)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить финансовые события.")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (handler *Handler) CreatePurchase(w http.ResponseWriter, r *http.Request) {
	var request singleAccountRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	handler.createSingle(w, r, "purchase", request)
}

func (handler *Handler) CreateIncome(w http.ResponseWriter, r *http.Request) {
	var request singleAccountRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	handler.createSingle(w, r, "income_received", request)
}

func (handler *Handler) CreateDebtPayment(w http.ResponseWriter, r *http.Request) {
	var request singleAccountRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	handler.createSingle(w, r, "debt_payment", request)
}

func (handler *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	var request transferRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	request.SourceAccountID = strings.TrimSpace(request.SourceAccountID)
	request.DestinationAccountID = strings.TrimSpace(request.DestinationAccountID)
	request.Description = strings.TrimSpace(request.Description)
	occurredAt, ok := validateCommon(w, request.AmountMinor, request.OccurredAt)
	if !ok {
		return
	}
	if request.SourceAccountID == "" || request.DestinationAccountID == "" {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Укажите оба счёта перевода.")
		return
	}
	handler.execute(w, r, Command{
		Kind: "internal_transfer", IdempotencyKey: idempotencyKey(r),
		RequestHash: hashRequest(request), AccountID: request.SourceAccountID,
		DestinationAccountID: request.DestinationAccountID, AmountMinor: request.AmountMinor,
		Description: request.Description, OccurredAt: occurredAt,
	})
}

func (handler *Handler) CreateRefund(w http.ResponseWriter, r *http.Request) {
	var request refundRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	request.OriginalEventID = strings.TrimSpace(request.OriginalEventID)
	request.Merchant = strings.TrimSpace(request.Merchant)
	request.Description = strings.TrimSpace(request.Description)
	occurredAt, ok := validateCommon(w, request.AmountMinor, request.OccurredAt)
	if !ok {
		return
	}
	if request.OriginalEventID == "" {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Укажите исходную покупку.")
		return
	}
	handler.execute(w, r, Command{
		Kind: "refund", IdempotencyKey: idempotencyKey(r), RequestHash: hashRequest(request),
		OriginalEventID: request.OriginalEventID, AmountMinor: request.AmountMinor,
		Merchant: request.Merchant, Description: request.Description, OccurredAt: occurredAt,
	})
}

func (handler *Handler) createSingle(w http.ResponseWriter, r *http.Request, kind string, request singleAccountRequest) {
	request.AccountID = strings.TrimSpace(request.AccountID)
	request.Merchant = strings.TrimSpace(request.Merchant)
	request.Description = strings.TrimSpace(request.Description)
	occurredAt, ok := validateCommon(w, request.AmountMinor, request.OccurredAt)
	if !ok {
		return
	}
	if request.AccountID == "" {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Укажите счёт.")
		return
	}
	if kind == "purchase" && request.Merchant == "" && request.Description == "" {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Для покупки укажите получателя или описание.")
		return
	}
	handler.execute(w, r, Command{
		Kind: kind, IdempotencyKey: idempotencyKey(r), RequestHash: hashRequest(request),
		AccountID: request.AccountID, CategoryID: request.CategoryID,
		AmountMinor: request.AmountMinor, Merchant: request.Merchant,
		Description: request.Description, OccurredAt: occurredAt,
	})
}

func (handler *Handler) execute(w http.ResponseWriter, r *http.Request, command Command) {
	if command.IdempotencyKey == "" {
		web.WriteError(w, http.StatusBadRequest, "idempotency_key_required", "Передайте заголовок Idempotency-Key.")
		return
	}
	if len(command.IdempotencyKey) > 160 {
		web.WriteError(w, http.StatusBadRequest, "invalid_idempotency_key", "Idempotency-Key слишком длинный.")
		return
	}
	event, created, err := handler.repo.Create(r.Context(), command)
	if errors.Is(err, ErrConflict) {
		web.WriteError(w, http.StatusConflict, "idempotency_conflict", "Этот Idempotency-Key уже использован с другим запросом.")
		return
	}
	if errors.Is(err, ErrInvalid) {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось сохранить финансовое событие.")
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	web.WriteJSON(w, status, event)
}

func validateCommon(w http.ResponseWriter, amount int64, occurredAtRaw string) (time.Time, bool) {
	if amount <= 0 {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Сумма должна быть больше нуля.")
		return time.Time{}, false
	}
	occurredAt, err := time.Parse(time.RFC3339, occurredAtRaw)
	if err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Дата операции должна быть в формате RFC 3339.")
		return time.Time{}, false
	}
	return occurredAt, true
}

func idempotencyKey(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("Idempotency-Key"))
}

func hashRequest(request any) string {
	payload, _ := json.Marshal(request)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
