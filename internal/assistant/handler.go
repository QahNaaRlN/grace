package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/QahNaaRlN/grace/internal/platform/web"
)

type QueryRunner interface {
	Query(context.Context, Query) QueryResult
}

type Handler struct{ router QueryRunner }

func NewHandler(router QueryRunner) *Handler { return &Handler{router: router} }

func (handler *Handler) Query(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Utterance     string `json:"utterance"`
		Language      string `json:"language"`
		ReferenceTime string `json:"referenceTime"`
	}
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		web.WriteError(writer, http.StatusUnprocessableEntity, "validation_error", "Запрос ассистента некорректен.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		web.WriteError(writer, http.StatusUnprocessableEntity, "validation_error", "Запрос ассистента некорректен.")
		return
	}
	query, err := ValidateQuery(input.Utterance, input.Language, input.ReferenceTime)
	if err != nil {
		web.WriteError(writer, http.StatusUnprocessableEntity, "validation_error", "Запрос ассистента некорректен.")
		return
	}
	result := handler.router.Query(request.Context(), query)
	status := http.StatusOK
	if result.Status == StatusUnavailable {
		status = http.StatusServiceUnavailable
	}
	web.WriteJSON(writer, status, result)
}
