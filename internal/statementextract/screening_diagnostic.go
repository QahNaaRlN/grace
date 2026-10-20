package statementextract

import "regexp"

// ScreeningDiagnostic is operator-only evidence, never verified billing or consent.
// CorrelationID is private and unverified even when it has a valid gateway shape.
type ScreeningDiagnostic struct {
	Code          string `json:"code"`
	HTTPStatus    int    `json:"httpStatus,omitempty"`
	ContentType   string `json:"contentType,omitempty"`
	CorrelationID string `json:"unverifiedCorrelationId,omitempty"`
}

var correlationID = regexp.MustCompile(`^gen-[0-9A-Za-z-]+$`)

// Sanitize closes every field before a diagnostic is persisted or summarized.
func (d ScreeningDiagnostic) Sanitize() ScreeningDiagnostic {
	switch d.Code {
	case "request", "transport", "timeout", "http_4xx", "http_5xx", "http_other", "read", "oversize", "envelope", "metadata_missing", "requested", "strategy", "attempt", "byok", "pipeline", "selected_model", "provider", "count", "display_provider", "cost", "usage", "token_bound", "choices", "tools", "finish_length", "finish_other", "content":
	default:
		d.Code = "unknown"
	}
	if d.HTTPStatus < 100 || d.HTTPStatus > 599 {
		d.HTTPStatus = 0
	}
	switch d.ContentType {
	case "json", "event_stream", "other", "missing":
	default:
		d.ContentType = ""
	}
	if len(d.CorrelationID) > 128 || !correlationID.MatchString(d.CorrelationID) {
		d.CorrelationID = ""
	}
	return d
}
