package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type observedFake struct {
	comparisonFake
	behavior string
	returned statementdata.Proposal
}

func (f *observedFake) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	part, bill, err := f.comparisonFake.Complete(ctx, p, c)
	if err != nil || bill == nil {
		return part, bill, err
	}
	numeric, text := "", ""
	for _, s := range c.Spans {
		if s.LexicalClass == statementextract.ClassNumericFragment {
			numeric = s.ID
		} else {
			text = s.ID
		}
	}
	switch f.behavior {
	case "invalid_plan":
		c.Spans[0].LexicalClass = "PRIVATE"
	case "unknown_alias":
		part.Rows = []statementdata.RowMapping{{Description: statementdata.Selection{"PRIVATE-UNKNOWN"}}}
	case "duplicate_alias":
		part.Rows = []statementdata.RowMapping{{AccountAmount: statementdata.Selection{numeric, numeric}}}
	case "non_numeric_money_selection":
		part.Header.DebitTurnover = statementdata.Selection{text}
	case "limit":
		part.Rows = []statementdata.RowMapping{{Description: statementdata.Selection{strings.Repeat("x", 65)}}}
	case "private":
		part.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{c.Spans[0].ID}, Reason: "PRIVATE-SERVICE"}}
	case "trim", "base-overflow":
		n := 50
		if f.calls == 3 {
			n = 15
		}
		if f.behavior == "base-overflow" {
			n = 120
		}
		part.Service = make([]statementdata.ServiceMapping, n)
		for i := range part.Service {
			part.Service[i].Reason = strings.Repeat("x", 2000)
		}
	case "route":
		bill.ComparisonRoute.SelectedModelClass = "PRIVATE"
	}
	f.returned = part
	return part, bill, err
}
func observedSamples() []Sample {
	s := comparisonMinimal()
	for i := range s {
		s[i].Document.Spans = append(append([]statementdata.Span{}, s[i].Document.Spans...), statementdata.Span{ID: "number", Page: 1, Block: 1, Line: 1, Word: 2, Text: "123.45", Box: statementdata.Box{XMin: 25, YMin: 1, XMax: 55, YMax: 10}})
	}
	return s
}
func observedReasoners(behavior string) ([2]statementextract.Reasoner, *observedFake, *observedFake) {
	cs := statementextract.ComparisonCandidates()
	a := &observedFake{comparisonFake: comparisonFake{candidate: cs[0]}, behavior: behavior}
	b := &observedFake{comparisonFake: comparisonFake{candidate: cs[1]}, behavior: behavior}
	return [2]statementextract.Reasoner{a, b}, a, b
}
func TestObservedComparisonCaptureAndReplay(t *testing.T) {
	for _, code := range []string{"unknown_alias", "duplicate_alias", "non_numeric_money_selection"} {
		t.Run(code, func(t *testing.T) {
			rs, a, b := observedReasoners(code)
			m := &fakeMeter{}
			samples := observedSamples()
			e, err := RunObservedComparison(context.Background(), samples, rs, m)
			if err == nil || e.Report.Error != "qualification_protocol_failed" || a.calls != 1 || b.calls != 0 || m.reserves != 1 || m.settles != 1 || m.actual == nil || *m.actual != 10 {
				t.Fatal("capture changed accounting/stop", err)
			}
			o := e.Observations[0]
			if o.Stage != "canonicalization_failed" || o.CaptureStatus != "captured" || o.Snapshot == nil || o.Diagnostic.Code != code || e.Report.Tuples[0].Proposal != nil {
				t.Fatal("lost pre-canonical snapshot")
			}
			if ValidateObservedComparison(e, time.Now()) != nil {
				t.Fatal("invalid observed report")
			}
			replay := ReplayComparisonObservation(&e, 0, samples[0].Document, time.Now())
			if !replay.Verified || replay.Status != "matched" || !reflect.DeepEqual(replay.Diagnostic, o.Diagnostic) {
				t.Fatal("causal replay mismatch", replay)
			}
			before, _ := json.Marshal(o.Snapshot)
			if len(a.returned.Rows) > 0 && len(a.returned.Rows[0].Description) > 0 {
				a.returned.Rows[0].Description[0] = "PRIVATE-MUTATED"
			} else if len(a.returned.Rows) > 0 && len(a.returned.Rows[0].AccountAmount) > 0 {
				a.returned.Rows[0].AccountAmount[0] = "PRIVATE-MUTATED"
			} else {
				a.returned.Header.DebitTurnover = statementdata.Selection{"PRIVATE-MUTATED"}
			}
			after, _ := json.Marshal(o.Snapshot)
			if !bytes.Equal(before, after) {
				t.Fatal("snapshot was not a deep copy")
			}
			raw, _ := json.Marshal(e)
			r, decoded, err := DecodeComparisonArtifact(bytes.NewReader(raw))
			if err != nil || decoded == nil || !reflect.DeepEqual(r, e.Report) {
				t.Fatal("envelope roundtrip", err)
			}
			if _, err := DecodeReport(bytes.NewReader(raw)); err == nil {
				t.Fatal("observed artifact admitted as product qualification")
			}
			if _, err := DecodeComparisonReport(bytes.NewReader(raw)); err == nil {
				t.Fatal("legacy decoder accepts new fields")
			}
			oldRaw, _ := json.Marshal(e.Report)
			old, obs, err := DecodeComparisonArtifact(bytes.NewReader(oldRaw))
			again, _ := json.Marshal(old)
			if err != nil || obs != nil || !bytes.Equal(oldRaw, again) || ReplayComparisonObservation(obs, 0, samples[0].Document, time.Now()).Status != "unavailable_legacy" {
				t.Fatal("v1 migrated or guessed")
			}
		})
	}
}
func TestObservedBoundariesAndSnapshotLimits(t *testing.T) {
	for _, mode := range []string{"unknown", "protocol", "duplicate", "route", "limit", "budget", "settlement", "quality"} {
		t.Run(mode, func(t *testing.T) {
			rs, a, b := observedReasoners("")
			m := &fakeMeter{}
			var meter Meter = m
			switch mode {
			case "unknown", "protocol", "duplicate":
				a.mode = mode
				b.mode = mode
			case "route", "limit":
				a.behavior = mode
			case "budget":
				meter = &unavailableMeter{}
			case "settlement":
				a.behavior = "unknown_alias"
				meter = &comparisonSettleFail{}
			}
			e, err := RunObservedComparison(context.Background(), observedSamples(), rs, meter)
			if err == nil {
				t.Fatal("offline fake must not PASS quality")
			}
			if mode == "limit" {
				o := e.Observations[0]
				if o.CaptureStatus != "unavailable_limit" || o.Snapshot != nil || o.Diagnostic.Code != "unknown_alias" || m.actual == nil || *m.actual != 10 {
					t.Fatal("limit changed known accounting")
				}
				if ReplayComparisonObservation(&e, 0, observedSamples()[0].Document, time.Now()).Status != "unavailable_capture" {
					t.Fatal("limit replay guessed")
				}
				return
			}
			if mode == "settlement" {
				if e.Report.Error != "qualification_settlement_failed" || e.Observations[0].Snapshot == nil {
					t.Fatal("settlement discarded diagnostic")
				}
				return
			}
			if mode == "quality" {
				if a.calls != 3 || b.calls != 3 || e.Observations[5].Stage != "canonicalization_passed" || e.Report.ActualMicros == nil || *e.Report.ActualMicros != 60 {
					t.Fatal("quality continuation changed")
				}
				return
			}
			for i, o := range e.Observations {
				if mode == "duplicate" && i == 0 {
					if o.Stage != "canonicalization_passed" || o.Snapshot == nil {
						t.Fatal("first known call lost before duplicate generation stop")
					}
					continue
				}
				if o.Stage != "not_reached" || o.Snapshot != nil || o.Diagnostic != nil {
					t.Fatal("captured before provider/route/billing boundary")
				}
			}
		})
	}
	for _, mutate := range []func(*statementdata.Proposal){
		func(p *statementdata.Proposal) { p.Rows = make([]statementdata.RowMapping, 5001) },
		func(p *statementdata.Proposal) { p.Service = make([]statementdata.ServiceMapping, 2001) },
		func(p *statementdata.Proposal) { p.Header.Currency = make(statementdata.Selection, 2001) },
		func(p *statementdata.Proposal) {
			p.Rows = []statementdata.RowMapping{{Direction: strings.Repeat("x", 65)}}
		},
		func(p *statementdata.Proposal) {
			p.Header.Labels = map[string]statementdata.Selection{strings.Repeat("x", 129): nil}
		},
		func(p *statementdata.Proposal) {
			p.Service = []statementdata.ServiceMapping{{Reason: strings.Repeat("é", 1001)}}
		},
		func(p *statementdata.Proposal) {
			p.Service = []statementdata.ServiceMapping{{Reason: string([]byte{0xff})}}
		},
	} {
		p := statementdata.Proposal{}
		mutate(&p)
		if captureComparisonSnapshot(p) != nil {
			t.Fatal("snapshot limits silently truncated")
		}
	}
}
func TestObservedEnvelopeTamperingPrivacyAndSize(t *testing.T) {
	rs, _, _ := observedReasoners("unknown_alias")
	good, _ := RunObservedComparison(context.Background(), observedSamples(), rs, &fakeMeter{})
	for _, mode := range []string{"inner", "outer", "ordinal", "stage", "capture", "diagnostic-code", "diagnostic-field", "private-role", "snapshot", "not-sent"} {
		t.Run(mode, func(t *testing.T) {
			raw, _ := json.Marshal(good)
			var e ObservedComparisonReport
			json.Unmarshal(raw, &e)
			switch mode {
			case "inner":
				e.Report.Digest = strings.Repeat("a", 64)
			case "outer":
				e.Digest = "bad"
			case "ordinal":
				e.Observations[0].Ordinal = 2
			case "stage":
				e.Observations[0].Stage = "canonicalization_passed"
			case "capture":
				e.Observations[0].CaptureStatus = "not_reached"
			case "diagnostic-code":
				e.Observations[0].Diagnostic.Code = "PRIVATE"
			case "diagnostic-field":
				e.Observations[0].Diagnostic.Field = "PRIVATE"
			case "private-role":
				e.Observations[0].Diagnostic.Role = "PRIVATE"
			case "snapshot":
				e.Observations[0].Snapshot = nil
			case "not-sent":
				e.Observations[1].Stage = "canonicalization_passed"
			}
			if mode != "outer" {
				e.Digest = observedDigest(e)
			}
			if ValidateObservedComparison(e, time.Now()) == nil {
				t.Fatal("forged observation accepted")
			}
			var out bytes.Buffer
			if EncodeObservedComparison(&out, e) == nil || out.Len() != 0 {
				t.Fatal("partial invalid artifact emitted")
			}
			if _, ok := PublicComparisonObservations(&e, time.Now()); ok {
				t.Fatal("unverified public diagnostics")
			}
		})
	}
	rows, ok := PublicComparisonObservations(&good, time.Now())
	raw, _ := json.Marshal(rows)
	if !ok || bytes.Contains(raw, []byte("PRIVATE")) || bytes.Contains(raw, []byte("snapshot")) {
		t.Fatal("public snapshot/alias leaked")
	}
	base, _ := json.Marshal(good)
	for _, raw := range [][]byte{bytes.Replace(base, []byte(`"version":`), []byte(`"Version":"PRIVATE","version":`), 1), append(base, []byte(` {}`)...), bytes.Replace(base, []byte(`"observations":`), []byte(`"PRIVATE":true,"observations":`), 1), []byte(strings.Repeat("[", 22) + "0" + strings.Repeat("]", 22))} {
		if _, _, err := DecodeComparisonArtifact(bytes.NewReader(raw)); err == nil {
			t.Fatal("ambiguous or excessive JSON accepted")
		}
	}
	rs, _, _ = observedReasoners("trim")
	trimmed, _ := RunObservedComparison(context.Background(), observedSamples(), rs, &fakeMeter{})
	if ValidateObservedComparison(trimmed, time.Now()) != nil || trimmed.Observations[5].CaptureStatus != "unavailable_report_limit" || trimmed.Observations[4].CaptureStatus != "captured" {
		t.Fatal("reverse whole-snapshot trim failed")
	}
	rs, _, _ = observedReasoners("base-overflow")
	m := &fakeMeter{}
	huge, _ := RunObservedComparison(context.Background(), observedSamples(), rs, m)
	var out bytes.Buffer
	if EncodeObservedComparison(&out, huge) == nil || out.Len() != 0 || m.actual == nil || *m.actual != 60 || huge.Report.ActualMicros == nil || *huge.Report.ActualMicros != 60 {
		t.Fatal("artifact overflow erased accounting or emitted partial report")
	}
}

func TestObservedEligibilityFailsBeforeCapture(t *testing.T) {
	for _, mode := range []string{"fixture", "name", "policy"} {
		t.Run(mode, func(t *testing.T) {
			samples := observedSamples()
			rs, a, b := observedReasoners("private")
			meter := &fakeMeter{}
			switch mode {
			case "fixture":
				samples[0].SHA256 = "not-public"
			case "name":
				samples[0].Name = "personal"
			case "policy":
				a.candidate.Policy.Model = "PRIVATE"
			}
			report, err := RunObservedComparison(context.Background(), samples, rs, meter)
			if err == nil || a.calls != 0 || b.calls != 0 || meter.reserves != 0 {
				t.Fatal("ineligible capture reached call/meter")
			}
			for _, o := range report.Observations {
				if o.Stage != "not_reached" || o.CaptureStatus != "not_reached" || o.Snapshot != nil || o.Diagnostic != nil {
					t.Fatal("ineligible capture persisted proposal")
				}
			}
		})
	}
}
