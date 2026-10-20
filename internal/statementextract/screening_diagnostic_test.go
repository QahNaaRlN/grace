package statementextract

import (
	"strings"
	"testing"
)

func TestScreeningDiagnosticSanitize(t *testing.T) {
	for _, id := range []string{"gen-aA0-Z9", "gen-1792514513-SYNTHETIC"} {
		d := (ScreeningDiagnostic{Code: "envelope", HTTPStatus: 200, ContentType: "json", CorrelationID: id}).Sanitize()
		if d.CorrelationID != id || d.Code != "envelope" || d.HTTPStatus != 200 {
			t.Fatal(d)
		}
	}
	for _, id := range []string{"", "PRIVATE-key", "gen-a/b", "gen-PRIVATE\nkey", "gen-" + strings.Repeat("a", 125)} {
		d := (ScreeningDiagnostic{Code: "PRIVATE-key", HTTPStatus: 999, ContentType: "PRIVATE-key", CorrelationID: id}).Sanitize()
		if d.CorrelationID != "" || d.Code != "unknown" || d.HTTPStatus != 0 || d.ContentType != "" {
			t.Fatal(d)
		}
	}
}
