package statementqualification

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func TestScreeningIdentityReportTampering(t *testing.T) {
	mutations := map[string]func(*Report){
		"missing_proof":      func(r *Report) { r.Calls[0].ScreeningRoute.Proof = nil },
		"wrong_pair":         func(r *Report) { r.Calls[0].ScreeningRoute.Proof.Canonical += "-NEW" },
		"wrong_provider":     func(r *Report) { r.Calls[0].ScreeningRoute.Proof.Provider = "google-vertex/global" },
		"wrong_digest":       func(r *Report) { r.Calls[0].ScreeningRoute.Proof.CatalogSHA256 = "PRIVATE" },
		"stale":              func(r *Report) { r.Calls[0].ScreeningRoute.Proof.VerifiedAt = r.Time.Add(-11 * time.Minute) },
		"future_proof":       func(r *Report) { r.Calls[0].ScreeningRoute.Proof.VerifiedAt = time.Now().Add(time.Hour) },
		"future_observation": func(r *Report) { r.Calls[0].ScreeningRoute.ObservedAt = time.Now().Add(time.Hour) },
		"before_run":         func(r *Report) { r.Calls[0].ScreeningRoute.ObservedAt = r.Time.Add(-time.Second) },
		"after_end":          func(r *Report) { r.Calls[0].ScreeningRoute.ObservedAt = r.ScreeningFinishedAt.Add(time.Second) },
		"missing_route":      func(r *Report) { r.Calls[0].ScreeningRoute = nil },
		"other":              func(r *Report) { r.Calls[0].ScreeningRoute.SelectedModelClass = "other" },
		"multiple":           func(r *Report) { r.Calls[0].ScreeningRoute.SelectedModelClass = "multiple" },
		"missing_end":        func(r *Report) { r.ScreeningFinishedAt = nil },
		"future_end":         func(r *Report) { end := time.Now().Add(time.Hour); r.ScreeningFinishedAt = &end },
		"legacy_version":     func(r *Report) { r.Version = "statement.screening.v2.gemini31-diagnostics" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			r := screeningReportFixture()
			mutate(&r)
			r.Digest = reportDigest(r)
			if ValidateScreeningReport(r, time.Now()) == nil {
				t.Fatal("tampered proof accepted")
			}
		})
	}
	// Возраст каталога проверяется на время вызова; чтение отчёта позже не меняет историю.
	good := screeningReportFixture()
	if ValidateScreeningReport(good, time.Now().Add(11*time.Minute)) != nil {
		t.Fatal("historical proof lost")
	}
}

func TestProductReportWireKeepsPreviousShape(t *testing.T) {
	good := validReportFixture()
	raw, err := json.Marshal(good)
	if err != nil || strings.Contains(string(raw), "screeningRoute") || strings.Contains(string(raw), "screeningFinishedAt") {
		t.Fatal("product wire changed")
	}
	// Зафиксированная форма прежнего Call, не включающая поля S1.7.
	type oldCall struct {
		GenerationID string `json:"generationId"`
		InputTokens  int    `json:"inputTokens"`
		OutputTokens int    `json:"outputTokens"`
		CostMicros   int64  `json:"costMicros"`
		LatencyMS    int64  `json:"latencyMs"`
	}
	type oldReport struct {
		Digest        string                                `json:"digest"`
		Profile       Profile                               `json:"profile"`
		Corpus        []CaseEvidence                        `json:"corpus"`
		Traces        []Trace                               `json:"traces,omitempty"`
		Version       string                                `json:"version"`
		Time          time.Time                             `json:"time"`
		Policy        statementextract.Policy               `json:"policy"`
		MaximumMicros int64                                 `json:"maximumMicros"`
		ActualMicros  *int64                                `json:"actualMicros"`
		Results       []Result                              `json:"results"`
		Calls         []oldCall                             `json:"calls"`
		Passed        bool                                  `json:"passed"`
		Error         string                                `json:"error,omitempty"`
		Failure       *statementextract.ScreeningDiagnostic `json:"failure,omitempty"`
	}
	old := oldReport{Profile: good.Profile, Corpus: good.Corpus, Traces: good.Traces, Version: good.Version, Time: good.Time, Policy: good.Policy, MaximumMicros: good.MaximumMicros, ActualMicros: good.ActualMicros, Results: good.Results, Passed: good.Passed}
	for _, c := range good.Calls {
		old.Calls = append(old.Calls, oldCall{c.GenerationID, c.InputTokens, c.OutputTokens, c.CostMicros, c.LatencyMS})
	}
	if statementextract.Hash(old) != good.Digest {
		t.Fatal("product report digest changed")
	}
	good.Calls[0].ScreeningRoute = screeningRouteFixture(time.Now())
	good.Digest = reportDigest(good)
	if ValidateReport(good, time.Now()) == nil {
		t.Fatal("screening evidence authorized product")
	}
}
