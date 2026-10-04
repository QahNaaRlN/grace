package analytics

import (
	"net/http"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
)

const dateLayout = "2006-01-02"

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
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	from, fromErr := time.Parse(time.RFC3339, fromRaw)
	to, toErr := time.Parse(time.RFC3339, toRaw)
	if fromRaw == "" || toRaw == "" || fromErr != nil || toErr != nil || !from.Before(to) || to.After(from.AddDate(5, 0, 0)) {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "from и to должны задавать упорядоченный RFC 3339 интервал не длиннее 5 лет.")
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

func requestCashHorizon(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	asOfRaw := strings.TrimSpace(r.URL.Query().Get("asOf"))
	if asOfRaw == "" {
		asOfRaw = time.Now().UTC().Format(dateLayout)
	}
	throughRaw := strings.TrimSpace(r.URL.Query().Get("through"))
	asOf, asOfErr := time.Parse(dateLayout, asOfRaw)
	through, throughErr := time.Parse(dateLayout, throughRaw)
	if throughRaw == "" || asOfErr != nil || throughErr != nil || asOf.Format(dateLayout) != asOfRaw ||
		through.Format(dateLayout) != throughRaw || through.Before(asOf) || through.After(asOf.AddDate(5, 0, 0)) {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "asOf и through должны задавать упорядоченные даты YYYY-MM-DD не дальше 5 лет.")
		return "", "", false
	}
	return asOfRaw, throughRaw, true
}
