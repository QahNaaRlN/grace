package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestS114ContextCaptureAndLegacyIsolation(t *testing.T) {
	for _, mode := range []string{"unknown_alias", "duplicate_alias", "non_numeric_money_selection", "invalid_plan", "limit", "route", "quality", "unknown", "protocol", "duplicate", "settlement", "trim", "base-overflow"} {
		t.Run(mode, func(t *testing.T) {
			behavior := mode
			if mode == "quality" || mode == "unknown" || mode == "protocol" || mode == "duplicate" {
				behavior = ""
			}
			oldRS, oldA, oldB := observedReasoners(behavior)
			rs, a, b := observedReasoners(behavior)
			if mode == "unknown" || mode == "protocol" || mode == "duplicate" {
				oldA.mode = mode
				oldB.mode = mode
				a.mode = mode
				b.mode = mode
			}
			oldMeter, m := &fakeMeter{}, &fakeMeter{}
			var om, nm Meter = oldMeter, m
			if mode == "settlement" {
				oldA.behavior = "unknown_alias"
				a.behavior = "unknown_alias"
				om = &comparisonSettleFail{}
				nm = &comparisonSettleFail{}
			}
			samples := observedSamples()
			old, oe := RunObservedComparison(context.Background(), samples, oldRS, om)
			e, err := RunS114Comparison(context.Background(), samples, rs, nm)
			if (oe == nil) != (err == nil) || old.Report.Error != e.Report.Error || a.calls != oldA.calls || b.calls != oldB.calls || m.reserves != oldMeter.reserves || m.settles != oldMeter.settles || !reflect.DeepEqual(m.actual, oldMeter.actual) {
				t.Fatal("context changed stop/accounting", mode)
			}
			if e.Version != S114ObservedComparisonVersion || e.Report.Authorization != S114ComparisonAuthorization || old.Report.Authorization != ComparisonAuthorization {
				t.Fatal("campaign identity")
			}
			for i, tuple := range e.Report.Tuples {
				if tuple.RequestHash != old.Report.Tuples[i].RequestHash || tuple.Evidence != old.Report.Tuples[i].Evidence {
					t.Fatal("input contract changed")
				}
			}
			if mode == "base-overflow" {
				var out bytes.Buffer
				if EncodeS114Comparison(&out, e) == nil || out.Len() != 0 || m.actual == nil || *m.actual != 60 {
					t.Fatal("overflow accounting")
				}
				return
			}
			if ValidateS114Comparison(e, time.Now()) != nil || ValidateObservedComparison(e, time.Now()) == nil || ValidateComparisonReport(e.Report, time.Now()) == nil || ValidateS114Comparison(old, time.Now()) == nil {
				t.Fatal("cross-context validation")
			}
			var raw bytes.Buffer
			if EncodeS114Comparison(&raw, e) != nil {
				t.Fatal("writer")
			}
			_, decoded, err := DecodeS114ComparisonArtifact(bytes.NewReader(raw.Bytes()))
			if err != nil || decoded == nil {
				t.Fatal("S114 decode", err)
			}
			if _, _, err := DecodeComparisonArtifact(bytes.NewReader(raw.Bytes())); err == nil {
				t.Fatal("legacy envelope decoder widened")
			}
			bare, _ := json.Marshal(e.Report)
			if _, _, err := DecodeS114ComparisonArtifact(bytes.NewReader(bare)); err == nil {
				t.Fatal("new bare v1 admitted")
			}
			if decodedBare, err := DecodeComparisonReport(bytes.NewReader(bare)); err == nil && ValidateComparisonReport(decodedBare, time.Now()) == nil {
				t.Fatal("legacy v1 authorization widened")
			}
			projection, ok := PublicComparisonObservations(&e, time.Now())
			public, _ := json.Marshal(projection)
			if !ok || bytes.Contains(public, []byte("PRIVATE")) || bytes.Contains(public, []byte("snapshot")) {
				t.Fatal("privacy")
			}
			if mode == "unknown_alias" || mode == "duplicate_alias" || mode == "non_numeric_money_selection" {
				replay := ReplayComparisonObservation(&e, 0, samples[0].Document, time.Now())
				if !replay.Verified || replay.Diagnostic.Code != mode {
					t.Fatal("replay", replay)
				}
			}
		})
	}
}
func TestS114ForgeryRejectsActualIdentityWithoutRepair(t *testing.T) {
	rs, _, _ := observedReasoners("unknown_alias")
	good, _ := RunS114Comparison(context.Background(), observedSamples(), rs, &fakeMeter{})
	for _, mode := range []string{"old-auth", "arbitrary-auth", "old-outer", "wrong-inner", "inner-digest", "outer-digest", "diagnostic", "snapshot"} {
		t.Run(mode, func(t *testing.T) {
			raw, _ := json.Marshal(good)
			var e ObservedComparisonReport
			json.Unmarshal(raw, &e)
			switch mode {
			case "old-auth":
				e.Report.Authorization = ComparisonAuthorization
			case "arbitrary-auth":
				e.Report.Authorization = "PRIVATE"
			case "old-outer":
				e.Version = ObservedComparisonVersion
			case "wrong-inner":
				e.Report.Version = "PRIVATE"
			case "inner-digest":
				e.Report.Digest = strings.Repeat("0", 64)
			case "diagnostic":
				e.Observations[0].Diagnostic.Code = "PRIVATE"
			case "snapshot":
				e.Observations[0].Snapshot = nil
			}
			if mode != "inner-digest" {
				e.Report.Digest = comparisonDigest(e.Report)
			}
			e.Digest = observedDigest(e)
			if mode == "outer-digest" {
				e.Digest = strings.Repeat("0", 64)
			}
			raw, _ = json.Marshal(e)
			if _, _, err := DecodeS114ComparisonArtifact(bytes.NewReader(raw)); err == nil {
				t.Fatal("forged context accepted")
			}
		})
	}
	for _, mode := range []string{"fixture", "name", "policy", "budget"} {
		samples := observedSamples()
		rs, a, b := observedReasoners("unknown_alias")
		m := &fakeMeter{}
		var meter Meter = m
		switch mode {
		case "fixture":
			samples[0].SHA256 = "PRIVATE"
		case "name":
			samples[0].Name = "PRIVATE"
		case "policy":
			a.candidate.Policy.Model = "PRIVATE"
		case "budget":
			meter = &unavailableMeter{}
		}
		e, err := RunS114Comparison(context.Background(), samples, rs, meter)
		if err == nil || a.calls+b.calls != 0 || m.reserves != 0 {
			t.Fatal("eligibility reached provider", mode)
		}
		for _, o := range e.Observations {
			if o.Snapshot != nil || o.Stage != "not_reached" {
				t.Fatal("ineligible capture")
			}
		}
	}
}
