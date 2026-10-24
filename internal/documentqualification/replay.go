package documentqualification

import (
	"context"
	"errors"
	"reflect"
	"time"
)

func Replay(ctx context.Context, c Corpus, m Manifest, raw []byte) (Report, error) {
	var sealed struct {
		SHA256 string `json:"sha256"`
		Report Report `json:"report"`
	}
	fail := errors.New("report_invalid")
	if len(raw) > ReportLimit || artifactJSON(raw, &sealed) != nil {
		return Report{}, fail
	}
	r := sealed.Report
	if r.Version != Version || sealed.SHA256 != r.Hash() || !reflect.DeepEqual(r.Manifest, m) || m.Validate(c, m.Source, true) != nil || r.Grant.Source != m.Source || r.Grant.ManifestSHA256 != m.Hash() || !r.Grant.Valid(r.StartedAt) || r.FinishedAt.Before(r.StartedAt) || r.FinishedAt.After(r.Grant.ExpiresAt) || len(r.Wire) > 14 || len(r.Budget) > 40 {
		return r, fail
	}
	get, post, bytes := 0, 0, 0
	for _, v := range r.Wire {
		if v.DurationMS < 0 || v.DurationMS > 185000 || len(v.Body) > 8<<20 || !hash64(v.RequestSHA256) {
			return r, fail
		}
		bytes += len(v.Body)
		if v.Method == "GET" {
			get++
		} else if v.Method == "POST" {
			post++
		} else {
			return r, fail
		}
	}
	if get > 6 || post > 8 || bytes > 16<<20 {
		return r, fail
	}
	for _, v := range r.Budget {
		if len(v.ID) > 200 || len(v.Generation) > 200 || v.Amount < 0 || v.Amount > 1000000000 || v.Actual != nil && (*v.Actual < 0 || *v.Actual > MaximumObservedMicros) {
			return r, fail
		}
	}
	http := &replayHTTP{events: r.Wire}
	budget := &replayBudget{events: r.Budget}
	// Extend only the in-memory permission for this no-network replay client.
	// The original execution interval above remains bound to the original grant.
	grant := r.Grant
	grant.ExpiresAt = time.Now().Add(35 * time.Minute)
	core := runCore(ctx, c, m, grant, "offline-replay-credential", http, budget)
	if http.invalid || budget.invalid || http.index != len(r.Wire) || budget.index != len(r.Budget) || !reflect.DeepEqual(core, r.Core) {
		return r, fail
	}
	return r, nil
}

type PublicCase struct {
	InputTokens        *int    `json:"inputTokens"`
	OutputTokens       *int    `json:"outputTokens"`
	DurationMS         *int64  `json:"durationMs"`
	CheckResults       []Check `json:"checkResults"`
	LegacyCheckResults []Check `json:"legacyCheckResults"`
	Ordinal            int     `json:"ordinal"`
	Case               string  `json:"case"`
	State              string  `json:"state"`
	Reason             string  `json:"reason"`
	Checks             int     `json:"checks"`
	FailedChecks       int     `json:"failedChecks"`
	Recognition        bool    `json:"recognition"`
	NativeBinding      string  `json:"nativeBinding"`
	Settlement         string  `json:"settlement"`
	ActualMicros       *int64  `json:"actualMicros"`
}
type Summary struct {
	ProofSHA256          map[string]string `json:"proofSHA256"`
	OverCap              bool              `json:"overCap"`
	Version              string            `json:"version"`
	Source               string            `json:"source"`
	ManifestSHA256       string            `json:"manifestSHA256"`
	ReportSHA256         string            `json:"reportSHA256"`
	Decision             string            `json:"decision"`
	POST                 int               `json:"post"`
	MetadataGET          int               `json:"metadataGET"`
	KnownCostMicros      int64             `json:"knownCostMicros"`
	UnknownBills         int               `json:"unknownBills"`
	DurationMS           int64             `json:"durationMs"`
	ImportReady          bool              `json:"importReady"`
	ActivationAuthorized bool              `json:"activationAuthorized"`
	Cases                []PublicCase      `json:"cases"`
}

func Summarize(r Report) Summary {
	s := Summary{ProofSHA256: r.Core.Proofs, Version: Version, Source: r.Manifest.Source, ManifestSHA256: r.Manifest.Hash(), ReportSHA256: r.Hash(), Decision: r.Core.Decision, Cases: []PublicCase{}}
	for _, v := range r.Wire {
		if v.Method == "GET" {
			s.MetadataGET++
		} else {
			s.POST++
		}
		s.DurationMS += v.DurationMS
	}
	durations := []int64{}
	for _, v := range r.Wire {
		if v.Method == "POST" {
			durations = append(durations, v.DurationMS)
		}
	}
	postIndex := 0
	for _, v := range r.Core.Results {
		c := PublicCase{CheckResults: []Check{}, LegacyCheckResults: []Check{}, Ordinal: v.Ordinal, Case: v.Case, InputTokens: v.InputTokens, OutputTokens: v.OutputTokens, State: v.State, Reason: v.Reason, Settlement: v.Settlement, ActualMicros: v.ActualMicros}
		if v.State != "not_sent" && postIndex < len(durations) {
			n := durations[postIndex]
			c.DurationMS = &n
			postIndex++
		}
		if v.Measurement != nil {
			c.Checks = len(v.Measurement.Checks)
			c.CheckResults = append([]Check{}, v.Measurement.Checks...)
			c.LegacyCheckResults = []Check{}
			for _, v := range v.Measurement.LegacyChecks {
				c.LegacyCheckResults = append(c.LegacyCheckResults, Check{v.Code, v.RowOrdinal, v.Passed})
			}
			for _, check := range v.Measurement.Checks {
				if !check.Passed {
					c.FailedChecks++
				}
			}
			c.Recognition = v.Measurement.Recognition
			c.NativeBinding = v.Measurement.NativeBinding
		}
		if v.ActualMicros != nil {
			s.KnownCostMicros += *v.ActualMicros
			if *v.ActualMicros > MaximumCallMicros {
				s.OverCap = true
			}
		} else if v.State != "not_sent" {
			s.UnknownBills++
		}
		s.Cases = append(s.Cases, c)
	}
	if s.KnownCostMicros > MaximumMicros {
		s.OverCap = true
	}
	return s
}
