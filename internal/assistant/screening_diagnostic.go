package assistant

import (
	"mime"
	"strings"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type screeningDiagnosticError struct {
	cause  error
	detail statementextract.ScreeningDiagnostic
}

func (e *screeningDiagnosticError) Error() string { return e.cause.Error() }
func (e *screeningDiagnosticError) Unwrap() error { return e.cause }
func (e *screeningDiagnosticError) ScreeningDiagnostic() statementextract.ScreeningDiagnostic {
	return e.detail.Sanitize()
}

func screeningContentType(value string) string {
	if value == "" {
		return "missing"
	}
	media, _, err := mime.ParseMediaType(value)
	if err != nil {
		return "other"
	}
	media = strings.ToLower(media)
	if media == "application/json" || strings.HasSuffix(media, "+json") {
		return "json"
	}
	if media == "text/event-stream" {
		return "event_stream"
	}
	return "other"
}
