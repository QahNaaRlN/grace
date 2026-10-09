package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AISettings struct {
	Enabled        bool
	Key            string
	AnalysisPolicy string
	ReceiptPolicy  string
	PhotoDir       string
}
type AIHandler struct {
	enabled         bool
	policy          ModelPolicy
	receiptPolicy   ModelPolicy
	receiptsEnabled bool
	provider        ReasoningProvider
	usage           *UsageRepository
	conversations   *ConversationRepository
	reader          MonthlyReader
}

func NewAIHandler(settings AISettings, pool *pgxpool.Pool, reader MonthlyReader) *AIHandler {
	h := &AIHandler{usage: NewUsageRepository(pool), conversations: NewConversationRepository(pool), reader: reader}
	if !settings.Enabled {
		return h
	}
	analysisValid := json.Unmarshal([]byte(settings.AnalysisPolicy), &h.policy) == nil && h.policy.Valid(time.Now())
	provider, err := NewOpenRouterProvider(settings.Key, nil)
	if err != nil {
		return h
	}
	h.provider = provider
	h.enabled = analysisValid
	if json.Unmarshal([]byte(settings.ReceiptPolicy), &h.receiptPolicy) == nil && h.receiptPolicy.Vision && h.receiptPolicy.Valid(time.Now()) && settings.PhotoDir != "" {
		h.receiptsEnabled = true
	}
	return h
}
func (h *AIHandler) SetReceiptStorageAvailable(available bool) {
	if !available {
		h.receiptsEnabled = false
	}
}
func (h *AIHandler) available() bool {
	return h.enabled && h.provider != nil && h.policy.Valid(time.Now())
}
func (h *AIHandler) ReceiptExtractor() *ReceiptExtractor {
	if !h.receiptsEnabled {
		return nil
	}
	return &ReceiptExtractor{Policy: h.receiptPolicy, Provider: h.provider, Meter: h.usage, Consents: h.conversations}
}
func (h *AIHandler) Routes(r chi.Router) {
	r.Get("/capabilities", h.Capabilities)
	r.Post("/consents", h.Consent)
	r.Delete("/consents/{scope}", h.RevokeConsent)
	r.Post("/conversations", h.CreateConversation)
	r.Get("/conversations/{conversationId}", h.GetConversation)
	r.Delete("/conversations/{conversationId}", h.DeleteConversation)
	r.Post("/conversations/{conversationId}/messages", h.Message)
}
func (h *AIHandler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	if err := h.conversations.Delete(r.Context(), chi.URLParam(r, "conversationId")); err != nil {
		aiError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *AIHandler) Capabilities(w http.ResponseWriter, r *http.Request) {
	remaining := int64(0)
	if h.available() || h.receiptsEnabled {
		n, err := h.usage.Remaining(r.Context())
		if err != nil {
			aiError(w, errors.New("ai_usage_requires_review"))
			return
		}
		remaining = n
	}
	consent := false
	if h.available() {
		var err error
		consent, err = h.conversations.HasConsent(r.Context(), "analysis", ConsentHash("analysis", h.policy))
		if err != nil {
			aiError(w, err)
			return
		}
	}
	receiptConsent := false
	if h.receiptsEnabled {
		var err error
		receiptConsent, err = h.conversations.HasConsent(r.Context(), "receipt", ConsentHash("receipt", h.receiptPolicy))
		if err != nil {
			aiError(w, err)
			return
		}
	}
	web.WriteJSON(w, 200, map[string]any{"analysis": map[string]any{"enabled": h.available() && remaining > 0, "model": h.policy.Model, "provider": h.policy.Provider, "consentGranted": consent, "policyHash": ConsentHash("analysis", h.policy), "dataDescription": "Твой вопрос, выбранный календарный месяц и итоги подтверждённых операций по категориям. Без номеров карт, снимка таблицы и фото.", "privacy": "OpenRouter и выбранный провайдер; ZDR, без обучения, внешних инструментов и скрытой смены маршрута."}, "receipt": map[string]any{"enabled": h.receiptsEnabled && h.receiptPolicy.Valid(time.Now()) && remaining > 0, "model": h.receiptPolicy.Model, "provider": h.receiptPolicy.Provider, "consentGranted": receiptConsent, "policyHash": ConsentHash("receipt", h.receiptPolicy), "dataDescription": "Нормализованное фото чека без метаданных. Передача отдельно от подтверждения расхода.", "privacy": "OpenRouter и выбранный провайдер; маршрут с ZDR и запретом сбора данных."}, "remainingMicros": strconv.FormatInt(remaining, 10), "policyVersion": "grace.ai.20261008.zdr.confirmed"})
}
func (h *AIHandler) Consent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Scope      string `json:"scope"`
		PolicyHash string `json:"policyHash"`
	}
	if !web.DecodeJSON(w, r, &in) {
		return
	}
	if !h.available() && !(in.Scope == "receipt" && h.receiptsEnabled && h.receiptPolicy.Valid(time.Now())) {
		aiError(w, errors.New("ai_disabled"))
		return
	}
	policy := h.policy
	if in.Scope == "receipt" && h.receiptsEnabled {
		policy = h.receiptPolicy
	} else if in.Scope != "analysis" {
		aiError(w, errors.New("policy_changed"))
		return
	}
	if in.PolicyHash != ConsentHash(in.Scope, policy) {
		aiError(w, errors.New("policy_changed"))
		return
	}
	if err := h.conversations.Consent(r.Context(), in.Scope, in.PolicyHash, true); err != nil {
		aiError(w, err)
		return
	}
	web.WriteJSON(w, 200, map[string]bool{"granted": true})
}
func (h *AIHandler) RevokeConsent(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	if scope != "analysis" && scope != "receipt" {
		aiError(w, errors.New("invalid_request"))
		return
	}
	if err := h.conversations.Consent(r.Context(), scope, "", false); err != nil {
		aiError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *AIHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	if !h.available() {
		aiError(w, errors.New("ai_disabled"))
		return
	}
	id, err := h.conversations.Create(r.Context())
	if err != nil {
		aiError(w, err)
		return
	}
	web.WriteJSON(w, 201, map[string]string{"id": id})
}
func (h *AIHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	items, err := h.conversations.Get(r.Context(), chi.URLParam(r, "conversationId"))
	if err != nil {
		aiError(w, err)
		return
	}
	web.WriteJSON(w, 200, map[string]any{"id": chi.URLParam(r, "conversationId"), "messages": items})
}
func (h *AIHandler) Message(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Question string `json:"question"`
		Month    string `json:"month"`
	}
	if !web.DecodeJSON(w, r, &in) {
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 160 || len(in.Question) > 2000 || strings.TrimSpace(in.Question) == "" {
		aiError(w, errors.New("invalid_request"))
		return
	}
	if !h.available() {
		aiError(w, errors.New("ai_disabled"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(95 * time.Second))
	consent, err := h.conversations.HasConsent(ctx, "analysis", ConsentHash("analysis", h.policy))
	if err != nil {
		aiError(w, err)
		return
	}
	if !consent {
		aiError(w, errors.New("consent_required"))
		return
	}
	from, to, err := CalendarMonth(in.Month)
	if err != nil {
		aiError(w, err)
		return
	}
	summary, err := h.reader.PeriodSummary(ctx, from, to)
	if err != nil {
		aiError(w, err)
		return
	}
	facts := BuildAnalysisFacts(in.Month, summary, time.Now())
	conversation := chi.URLParam(r, "conversationId")
	id, old, err := h.conversations.BeginMessage(ctx, conversation, key, aiHash(in), in.Question, in.Month, facts)
	if err != nil {
		aiError(w, err)
		return
	}
	if old != nil {
		web.WriteJSON(w, 200, old)
		return
	}
	answer := AnalysisAnswer{Facts: facts, Interpretations: []AnalysisInterpretation{}, Notice: "Числа рассчитаны Grace. Текст модели — предположения и предложения, а не подтверждённые финансовые факты. Полнота исходной истории неизвестна."}
	request, err := AnalysisRequest(h.policy, in.Question, facts)
	var out LLMResponse
	if err == nil {
		// Recheck consent immediately before transmission, including route changes.
		var yes bool
		yes, err = h.conversations.HasConsent(ctx, "analysis", ConsentHash("analysis", h.policy))
		if err == nil && !yes {
			err = errors.New("consent_required")
		}
	}
	if err == nil {
		out, err = MeteredCompletion(ctx, h.provider, h.usage, request, "analysis")
	}
	if err == nil {
		answer.Interpretations, err = ValidateInterpretations(out.Content, facts)
		answer.Model = out.Model
		answer.Provider = out.Provider
		answer.CostMicros = strconv.FormatInt(out.CostMicros, 10)
	}
	persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer persistCancel()
	code := ""
	if err != nil {
		code = aiCode(err)
	}
	if finishErr := h.conversations.Finish(persistCtx, id, answer, code); finishErr != nil {
		aiError(w, finishErr)
		return
	}
	if err != nil {
		web.WriteJSON(w, 200, map[string]any{"id": id, "question": in.Question, "month": in.Month, "status": "failed", "facts": facts, "response": nil, "errorCode": code})
		return
	}
	web.WriteJSON(w, 200, ConversationMessage{ID: id, Question: in.Question, Month: in.Month, Status: "completed", Facts: mustJSON(facts), Response: mustJSON(answer)})
}
func aiCode(err error) string {
	allowed := []string{"ai_disabled", "invalid_request", "invalid_period", "consent_required", "policy_changed", "budget_exhausted", "price_unknown", "ai_usage_requires_review", "ai_concurrency_limit", "provider_timeout", "provider_unavailable", "provider_protocol", "provider_request", "unsupported_claim", "context_limit", "conversation_limit", "not_found", "ai_request_conflict", "ai_request_in_progress"}
	for _, code := range allowed {
		if err.Error() == code || strings.Contains(err.Error(), ": "+code) {
			return code
		}
	}
	return "ai_unavailable"
}
func aiError(w http.ResponseWriter, err error) {
	code := aiCode(err)
	status := http.StatusUnprocessableEntity
	if code == "not_found" {
		status = 404
	}
	if code == "ai_unavailable" || code == "provider_unavailable" || code == "ai_disabled" {
		status = 503
	}
	if code == "ai_request_conflict" || code == "ai_request_in_progress" {
		status = 409
	}
	web.WriteError(w, status, code, "ИИ-запрос не выполнен. Обычные функции Grace доступны; проверь согласие, выбранный период и лимит эксперимента.")
}
