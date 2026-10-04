package analytics

import (
	"net/http"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (handler *Handler) PeriodSummary(w http.ResponseWriter, r *http.Request) {
	from, to, ok := requestPeriod(w, r)
	if !ok {
		return
	}
	summary, err := handler.repo.PeriodSummary(r.Context(), from, to)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось рассчитать сводку периода.")
		return
	}
	web.WriteJSON(w, http.StatusOK, summary)
}

func (handler *Handler) CashPosition(w http.ResponseWriter, r *http.Request) {
	asOf, through, ok := requestCashHorizon(w, r)
	if !ok {
		return
	}
	position, err := handler.repo.CashPosition(r.Context(), asOf, through)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось рассчитать доступность денег.")
		return
	}
	web.WriteJSON(w, http.StatusOK, position)
}

func requestPeriod(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	from, to, err := ParsePeriod(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "from и to должны задавать упорядоченный RFC 3339 интервал не длиннее 5 лет.")
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

func requestCashHorizon(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	asOf, through, err := ParseCashHorizon(r.URL.Query().Get("asOf"), r.URL.Query().Get("through"), time.Now())
	if err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "asOf и through должны задавать упорядоченные даты YYYY-MM-DD не дальше 5 лет.")
		return "", "", false
	}
	return asOf, through, true
}
