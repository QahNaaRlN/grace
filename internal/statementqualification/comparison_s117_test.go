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

func TestS117ContextCaptureAndLegacyIsolation(t *testing.T) {
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
			e, err := RunS117Comparison(context.Background(), samples, rs, nm)
			if (oe == nil) != (err == nil) || old.Report.Error != e.Report.Error || a.calls != oldA.calls || b.calls != oldB.calls || m.reserves != oldMeter.reserves || m.settles != oldMeter.settles || !reflect.DeepEqual(m.actual, oldMeter.actual) {
				t.Fatal("context changed stop/accounting", mode)
			}
			if e.Version != S117ObservedComparisonVersion || e.Report.Authorization != S117ComparisonAuthorization || old.Report.Authorization != ComparisonAuthorization {
				t.Fatal("campaign identity")
			}
			for i, tuple := range e.Report.Tuples {
				if tuple.RequestHash != old.Report.Tuples[i].RequestHash || tuple.Evidence != old.Report.Tuples[i].Evidence {
					t.Fatal("input contract changed")
				}
			}
			if mode == "base-overflow" {
				var out bytes.Buffer
				if EncodeS117Comparison(&out, e) == nil || out.Len() != 0 || m.actual == nil || *m.actual != 60 {
					t.Fatal("overflow accounting")
				}
				return
			}
			if ValidateS117Comparison(e, time.Now()) != nil || ValidateObservedComparison(e, time.Now()) == nil || ValidateComparisonReport(e.Report, time.Now()) == nil || ValidateS117Comparison(old, time.Now()) == nil {
				t.Fatal("cross-context validation")
			}
			var raw bytes.Buffer
			if EncodeS117Comparison(&raw, e) != nil {
				t.Fatal("writer")
			}
			_, decoded, err := DecodeS117ComparisonArtifact(bytes.NewReader(raw.Bytes()))
			if err != nil || decoded == nil {
				t.Fatal("S117 decode", err)
			}
			if _, _, err := DecodeComparisonArtifact(bytes.NewReader(raw.Bytes())); err == nil {
				t.Fatal("legacy envelope decoder widened")
			}
			bare, _ := json.Marshal(e.Report)
			if _, _, err := DecodeS117ComparisonArtifact(bytes.NewReader(bare)); err == nil {
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
func TestS117ForgeryRejectsActualIdentityWithoutRepair(t *testing.T) {
	rs, _, _ := observedReasoners("unknown_alias")
	good, _ := RunS117Comparison(context.Background(), observedSamples(), rs, &fakeMeter{})
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
			if _, _, err := DecodeS117ComparisonArtifact(bytes.NewReader(raw)); err == nil {
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
		e, err := RunS117Comparison(context.Background(), samples, rs, meter)
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

// Reports originate in their own runner. No fixture is repaired or translated
// into another context; each decoder must reject both neighbouring envelopes.
func TestS117ActualCampaignDecoderMatrix(t *testing.T) {
	runners := []struct {
		name     string
		campaign comparisonCampaign
	}{{"legacy", legacyComparison}, {"s114", s114Comparison}, {"s117", s117Comparison}}
	for _, origin := range runners {
		rs, _, _ := observedReasoners("unknown_alias")
		report, _ := runObservedComparisonFor(context.Background(), observedSamples(), rs, &fakeMeter{}, origin.campaign)
		raw, _ := json.Marshal(report)
		for _, reader := range runners {
			_, _, err := decodeComparisonArtifactFor(bytes.NewReader(raw), reader.campaign, reader.campaign == legacyComparison)
			if (err == nil) != (reader.campaign == origin.campaign) {
				t.Fatalf("%s -> %s", origin.name, reader.name)
			}
		}
	}
	rs, _, _ := observedReasoners("unknown_alias")
	good, _ := RunS117Comparison(context.Background(), observedSamples(), rs, &fakeMeter{})
	for _, kind := range []string{"s114-auth", "s114-outer", "old-profile", "old-schema", "new-schema-old-planner", "duplicate-field", "trailing", "unknown-field"} {
		t.Run(kind, func(t *testing.T) {
			raw, _ := json.Marshal(good)
			var bad ObservedComparisonReport
			if json.Unmarshal(raw, &bad) != nil {
				t.Fatal("test fixture")
			}
			switch kind {
			case "s114-auth":
				bad.Report.Authorization = S114ComparisonAuthorization
			case "s114-outer":
				bad.Version = S114ObservedComparisonVersion
			case "old-profile":
				bad.Report.Profile.Planner = "statement-document-atom.v2.lexical-grounding"
				bad.Report.Profile.Schema = "52a36f2b6159b7e9a3e2fc0509ba8c688142eebd12b60be1556d7165da56f0bf"
			case "old-schema":
				bad.Report.Profile.Schema = "52a36f2b6159b7e9a3e2fc0509ba8c688142eebd12b60be1556d7165da56f0bf"
			case "new-schema-old-planner":
				bad.Report.Profile.Planner = "statement-document-atom.v2.lexical-grounding"
			}
			// A correct digest does not authorize an incompatible profile/identity.
			bad.Report.Digest = comparisonDigest(bad.Report)
			bad.Digest = observedDigest(bad)
			raw, _ = json.Marshal(bad)
			switch kind {
			case "duplicate-field":
				raw = bytes.Replace(raw, []byte(`"version":`), []byte(`"Version":"PRIVATE","version":`), 1)
			case "trailing":
				raw = append(raw, []byte(` {}`)...)
			case "unknown-field":
				raw = bytes.Replace(raw, []byte(`"observations":`), []byte(`"PRIVATE":true,"observations":`), 1)
			}
			if _, _, err := DecodeS117ComparisonArtifact(bytes.NewReader(raw)); err == nil {
				t.Fatal("incompatible report accepted")
			}
		})
	}
}
