package main

import (
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"strings"
	"testing"
)

func TestScreeningSummaryOnlyExportsClosedCodes(t *testing.T) {
	for _, reason := range []string{"metadata_missing", "PRIVATE-key"} {
		privateID := "gen-SYNTHETIC-private-correlation"
		report := statementqualification.Report{Version: statementqualification.ScreeningReportVersion, Error: "qualification_provider_failed_" + reason, Failure: &statementextract.ScreeningDiagnostic{Route: &statementextract.ScreeningRouteEvidence{SelectedModelClass: "PRIVATE-model"}, Code: reason, HTTPStatus: 200, ContentType: "PRIVATE-header", CorrelationID: privateID}, Calls: []statementqualification.Call{{GenerationID: privateID, CostMicros: 123456}}, Results: []statementqualification.Result{{Name: "columns", Codes: []string{"row_count", "PRIVATE-body"}, Validation: []string{"PRIVATE-error"}, Coverage: []string{"PRIVATE-request"}}}}
		raw, err := json.Marshal(screeningSummary(report))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, secret := range []string{"PRIVATE", privateID, "123456", "httpStatus", "Content-Type"} {
			if strings.Contains(text, secret) {
				t.Fatal("public summary leak", text)
			}
		}
		want := reason
		if reason != "metadata_missing" {
			want = "unknown"
		}
		if !strings.Contains(text, `"diagnosticCode":"`+want+`"`) || !strings.Contains(text, `"activationAuthorized":false`) {
			t.Fatal(text)
		}
	}
}

func TestScreeningSummaryModelClassesStayClosed(t *testing.T) {
	for _, class := range []string{"alias", "pinned_canonical", "other", "empty", "multiple", "not_observed", "PRIVATE-model"} {
		route := &statementextract.ScreeningRouteEvidence{SelectedModelClass: class}
		report := statementqualification.Report{Calls: []statementqualification.Call{{ScreeningRoute: route}}, Failure: &statementextract.ScreeningDiagnostic{Code: "selected_model", Route: route}}
		raw, _ := json.Marshal(screeningSummary(report))
		want := class
		if class == "PRIVATE-model" {
			want = "not_observed"
		}
		text := string(raw)
		if strings.Contains(text, "PRIVATE") || !strings.Contains(text, `"selectedModelClass":"`+want+`"`) || !strings.Contains(text, `"selectedModelClasses":["`+want+`"]`) || !strings.Contains(text, `"activationAuthorized":false`) {
			t.Fatal("класс или приватность сводки", text)
		}
	}
}

func TestGroundingSummaryClosedCodes(t *testing.T) {
	allowed := []string{"description_evidence_gap", "description_evidence_unverified", "book_balance_role_unverified", "openingBookBalance:book_balance_role_unverified", "closingBookBalance:book_balance_role_unverified", "row_1:description_evidence_gap", "row_5000:description_evidence_unverified"}
	for _, code := range allowed {
		got := screeningPublicCodes([]string{code})
		if len(got) != 1 || got[0] != code {
			t.Fatal("closed code lost", code, got)
		}
	}
	for _, code := range []string{"row_0:description_evidence_gap", "row_5001:description_evidence_gap", "row_01:description_evidence_gap", "row_99999999999999999999:description_evidence_gap", "row_1:PRIVATE-error", "PRIVATE:book_balance_role_unverified", "openingBookBalance:book_balance_role_unverified:gen-PRIVATE", "row_1:description_evidence_gap\nPRIVATE"} {
		raw, _ := json.Marshal(screeningPublicCodes([]string{code}))
		if string(raw) != `["unknown"]` {
			t.Fatal("unclosed code leaked", string(raw))
		}
	}
}
