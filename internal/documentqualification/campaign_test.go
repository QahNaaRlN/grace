package documentqualification

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func corpus(t *testing.T) Corpus {
	t.Helper()
	c, e := Load("../..")
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func candidate() *assistant.GroupMembership {
	return &assistant.GroupMembership{Policy: assistant.ModelPolicy{Model: "fixture/vision-a", Provider: "fixture/eu", ProviderName: "Fixture", InputMicrosPerMillion: 100000, OutputMicrosPerMillion: 100000, ImageTokenUpperBound: 1000, Vision: true}, Canonical: "fixture/vision-a-v1", MaxImages: 4, MaxImageSide: 2048, Region: "eu"}
}
func grant(m Manifest) assistant.ExperimentGrant {
	return assistant.ExperimentGrant{Source: m.Source, Approval: strings.Repeat("b", 40), ManifestSHA256: m.Hash(), ExpiresAt: time.Now().Add(35 * time.Minute)}
}
func fakeContent(c Case) json.RawMessage {
	if c.Golden != nil {
		raw, _ := json.Marshal(c.Golden)
		var v statementdata.VisualReading
		_ = json.Unmarshal(raw, &v)
		for _, f := range v.Header {
			if f != nil {
				for i := range f.Evidence {
					f.Evidence[i].Box = [4]float64{0, 0, 1, 1}
				}
			}
		}
		for _, r := range v.Rows {
			for i := range r.Anchor {
				r.Anchor[i].Box = [4]float64{0, 0, 1, 1}
			}
			for _, f := range r.Fields {
				if f != nil {
					for i := range f.Evidence {
						f.Evidence[i].Box = [4]float64{0, 0, 1, 1}
					}
				}
			}
		}
		out, _ := json.Marshal(v)
		return out
	}
	var receipt map[string]any
	_ = json.Unmarshal(c.ReceiptGolden, &receipt)
	receipt["fiscalDocumentNumber"] = nil
	receipt["sourceText"] = "Public synthetic receipt text"
	issues := []string{}
	if !c.Readable {
		issues = append(issues, "unreadable")
	}
	raw, _ := json.Marshal(map[string]any{"complete": c.Readable, "receiptCount": 1, "issues": issues, "receipt": receipt})
	return raw
}

type fakeHTTP struct {
	requestOmitted bool
	c              Corpus
	posts, gets    int
	faultAt        int
	fault          string
	mutate         func(int, json.RawMessage) json.RawMessage
	wire           []map[string]any
}

func response(v any) *http.Response {
	raw, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(raw)))}
}
func (f *fakeHTTP) Do(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "openrouter.ai" {
		return nil, errors.New("unexpected_host")
	}
	if r.Method == "POST" {
		f.posts++
		raw, _ := io.ReadAll(r.Body)
		var wire map[string]any
		_ = json.Unmarshal(raw, &wire)
		f.wire = append(f.wire, wire)
		if f.fault == "timeout" && (f.faultAt == 0 || f.posts == f.faultAt) {
			return nil, errors.New("timeout")
		}
		content := fakeContent(f.c.Cases[f.posts-1])
		if f.mutate != nil {
			content = f.mutate(f.posts, content)
		}
		model := candidate().Policy.Model
		canonical := candidate().Canonical
		usage := map[string]any{"prompt_tokens": 500, "completion_tokens": 10, "cost": 0.00001}
		if f.fault == "cost_overrun" {
			usage["cost"] = 1.0
		}
		if f.fault == "credential_echo" {
			content = json.RawMessage(`{"secret":"offline-live-credential"}`)
		}
		if f.fault == "unknown_cost" {
			delete(usage, "cost")
		}
		if f.fault == "wrong_model" {
			canonical = "fixture/unapproved"
		}
		return response(map[string]any{"id": "generation-" + string(rune('0'+f.posts)), "model": canonical, "provider": "Fixture", "usage": usage, "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": string(content)}}}, "openrouter_metadata": map[string]any{"requested": model, "strategy": "direct", "attempt": 1, "is_byok": false, "pipeline": []any{}, "endpoints": map[string]any{"available": []any{map[string]any{"provider": "Fixture", "model": canonical, "selected": true}}}}}), nil
	}
	f.gets++
	params := []string{"structured_outputs", "response_format", "max_tokens"}
	input := []string{"text", "image"}
	if f.fault == "text_only" {
		input = []string{"text"}
	}
	if f.fault == "schema" {
		params = []string{"max_tokens"}
	}
	ep := map[string]any{"model_id": candidate().Policy.Model, "tag": "fixture/eu", "provider_name": "Fixture", "name": "Fixture", "status": 0, "context_length": 64000, "max_completion_tokens": 20000, "supported_parameters": params, "pricing": map[string]string{"prompt": "0.0000001", "completion": "0.0000001", "request": "0", "image": "0"}}
	if f.requestOmitted || f.fault == "request_omitted" {
		ep["pricing"] = map[string]string{"prompt": "0.0000001", "completion": "0.0000001", "image": "0.000000275"}
	}
	if f.fault == "fees" {
		ep["pricing"] = map[string]string{"prompt": "0.0000001", "completion": "0.0000001"}
	}
	if f.fault == "price" {
		ep["pricing"] = map[string]string{"prompt": "0.01", "completion": "0.0000001", "request": "0", "image": "0"}
	}
	switch r.URL.Path {
	case "/api/v1/models":
		return response(map[string]any{"data": []any{map[string]any{"id": candidate().Policy.Model, "canonical_slug": candidate().Canonical, "architecture": map[string]any{"input_modalities": input, "output_modalities": []string{"text"}}, "supported_parameters": params}}}), nil
	case "/api/v1/endpoints/zdr":
		if f.fault == "zdr" {
			return response(map[string]any{"data": []any{}}), nil
		}
		return response(map[string]any{"data": []any{ep}}), nil
	default:
		return response(map[string]any{"data": map[string]any{"endpoints": []any{ep}}}), nil
	}
}

type fakeBudget struct {
	n                int
	settled, unknown int
	guard            bool
	remaining        int64
	failSettle       bool
}

func (b *fakeBudget) Remaining(context.Context) (int64, error) {
	if b.remaining == 0 {
		return 10000000, nil
	}
	return b.remaining, nil
}
func (b *fakeBudget) Reserve(_ context.Context, _ assistant.ModelPolicy, _, _ int, _ string) (string, error) {
	b.n++
	return "reservation-" + string(rune('0'+b.n)), nil
}
func (b *fakeBudget) Allowed(context.Context, string) (bool, error) { return b.guard, nil }
func (b *fakeBudget) Settle(_ context.Context, _ string, out *assistant.LLMResponse) error {
	if b.failSettle {
		return errors.New("unavailable")
	}
	if out == nil {
		b.unknown++
	} else {
		b.settled++
	}
	return nil
}
func execute(t *testing.T, c Corpus, f *fakeHTTP, b *fakeBudget) (Manifest, Report) {
	t.Helper()
	source := strings.Repeat("a", 40)
	if v := os.Getenv("TEST_Q_SOURCE"); v != "" {
		source = v
	}
	m := NewManifest(c, source, candidate())
	r, e := Run(context.Background(), c, m, grant(m), "offline-live-credential", f, b)
	if e != nil {
		t.Fatal(e)
	}
	return m, r
}
func TestQActualGatewayEightCasesAndReplay(t *testing.T) {
	c := corpus(t)
	f := &fakeHTTP{c: c}
	b := &fakeBudget{guard: true}
	m, r := execute(t, c, f, b)
	if r.Core.Decision != "PASS_SYNTHETIC" || f.posts != 8 || f.gets != 6 || b.settled != 8 {
		for _, v := range r.Core.Results {
			t.Logf("%+v", v)
		}
		t.Fatal(r.Core.Decision, f.posts, f.gets, b.settled)
	}
	if len(r.Core.Results[3].Measurement.Checks) != 297 || len(r.Core.Results[4].Measurement.Checks) != 297 {
		t.Fatal("full32 checks changed")
	}
	for i, wire := range f.wire {
		if wire["model"] != candidate().Policy.Model || strings.Contains(string(mustJSON(wire["messages"])), "receiptGolden") {
			t.Fatal("wire changed or golden leaked")
		}
		if i == 2 || i == 4 {
			if strings.Contains(string(mustJSON(wire["messages"])), "Untrusted extracted document text:") {
				t.Fatal("invented native text")
			}
		}
	}
	raw, e := EncodeReport(r)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := Replay(context.Background(), c, m, raw)
	if e != nil {
		t.Fatal("replay failed", e)
	}
	s := Summarize(replay)
	if _, e := ValidateSummary(mustJSON(s), m); e != nil {
		t.Fatal("producer public summary invalid", e, len(mustJSON(s)))
	}
	if output := os.Getenv("TEST_Q_OUTPUT"); output != "" {
		if e := WriteBundle(filepath.Join(output, "bundle"), c, m); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(output, "good-summary.json"), mustJSON(s), 0600); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(output, "report.private.json"), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}

	if s.POST != 8 || s.MetadataGET != 6 || s.KnownCostMicros != 80 || s.ImportReady || s.ActivationAuthorized {
		t.Fatal(s)
	}
	for _, v := range r.Core.Results {
		if v.Measurement != nil && v.Measurement.ImportReady {
			t.Fatal("owner guard bypass")
		}
	}
}
func mustJSON(v any) []byte { raw, _ := json.Marshal(v); return raw }
func TestQQualityNegativesStopAndSettle(t *testing.T) {
	c := corpus(t)
	tests := []struct {
		name    string
		ordinal int
		mutate  func(map[string]any)
	}{
		{"missing_row32", 4, func(v map[string]any) { v["rows"] = v["rows"].([]any)[:31] }},
		{"extra_row", 1, func(v map[string]any) { v["rows"] = append(v["rows"].([]any), v["rows"].([]any)[0]) }},
		{"lost_time", 1, func(v map[string]any) {
			v["rows"].([]any)[0].(map[string]any)["fields"].(map[string]any)["operationDate"].(map[string]any)["value"] = "19.10.2026"
		}},
		{"wrong_currency", 1, func(v map[string]any) {
			v["rows"].([]any)[0].(map[string]any)["fields"].(map[string]any)["accountCurrency"].(map[string]any)["value"] = "USD"
		}},
		{"cross_page_quote_omission", 4, func(v map[string]any) {
			field := v["rows"].([]any)[10].(map[string]any)["fields"].(map[string]any)["description"].(map[string]any)
			field["evidence"] = field["evidence"].([]any)[:1]
		}},
		{"invent_seconds", 1, func(v map[string]any) {
			v["rows"].([]any)[0].(map[string]any)["fields"].(map[string]any)["operationDate"].(map[string]any)["value"] = "19.10.2026 10:30:01"
		}},
		{"missing_receipt_currency", 6, func(v map[string]any) { v["receipt"].(map[string]any)["currency"] = nil }},
		{"wrong_item_name", 6, func(v map[string]any) {
			v["receipt"].(map[string]any)["items"].([]any)[0].(map[string]any)["name"] = "WRONG ITEM"
		}},
		{"discount", 6, func(v map[string]any) {
			v["receipt"].(map[string]any)["items"].([]any)[0].(map[string]any)["discount"] = "0"
		}},
		{"missing_item", 6, func(v map[string]any) { v["receipt"].(map[string]any)["items"] = []any{} }},
		{"invented_unknown", 8, func(v map[string]any) { v["receipt"].(map[string]any)["total"] = "100.00" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeHTTP{c: c, mutate: func(n int, raw json.RawMessage) json.RawMessage {
				if n != tc.ordinal {
					return raw
				}
				var v map[string]any
				_ = json.Unmarshal(raw, &v)
				tc.mutate(v)
				return mustJSON(v)
			}}
			b := &fakeBudget{guard: true}
			m, r := execute(t, c, f, b)
			if r.Core.Decision != "REJECT" || f.posts != tc.ordinal || b.settled != tc.ordinal {
				t.Fatal(r.Core.Decision, f.posts, b.settled)
			}
			raw, _ := EncodeReport(r)
			if _, e := Replay(context.Background(), c, m, raw); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestQProtocolBudgetFailuresNeverBecomeQualityPass(t *testing.T) {
	c := corpus(t)
	for _, fault := range []string{"text_only", "schema", "zdr", "fees", "price", "timeout", "unknown_cost", "wrong_model", "cost_overrun", "credential_echo"} {
		t.Run(fault, func(t *testing.T) {
			f := &fakeHTTP{c: c, fault: fault}
			b := &fakeBudget{guard: true}
			m, r := execute(t, c, f, b)
			if r.Core.Decision != "INCONCLUSIVE" || f.posts > 1 || f.gets > 6 {
				t.Fatal(r.Core.Decision, f.posts, f.gets)
			}
			raw, _ := EncodeReport(r)
			if fault == "credential_echo" && strings.Contains(string(raw), "offline-live-credential") {
				t.Fatal("credential captured")
			}
			if fault == "cost_overrun" {
				summary := Summarize(r)
				if summary.KnownCostMicros != 1000000 || !summary.OverCap {
					t.Fatal("honest overrun lost", summary)
				}
				if _, e := ValidateSummary(mustJSON(summary), m); e != nil {
					t.Fatal("overrun projection invalid", e)
				}
			}
			if _, e := Replay(context.Background(), c, m, raw); e != nil {
				t.Fatal("failure replay", e)
			}
		})
	}
	for _, b := range []*fakeBudget{{guard: false}, {guard: true, remaining: 1}, {guard: true, failSettle: true}} {
		f := &fakeHTTP{c: c}
		_, r := execute(t, c, f, b)
		if r.Core.Decision != "INCONCLUSIVE" || f.posts > 1 {
			t.Fatal(r.Core.Decision, f.posts)
		}
	}
}
func TestQReplayRecomputesInsteadOfTrustingSavedResult(t *testing.T) {
	c := corpus(t)
	m, r := execute(t, c, &fakeHTTP{c: c}, &fakeBudget{guard: true})
	raw, _ := EncodeReport(r)
	var copy Report
	_ = json.Unmarshal(mustJSON(r), &copy)
	copy.Core.Results[0].Measurement.Checks[0].Passed = false
	altered, _ := EncodeReport(copy)
	if _, e := Replay(context.Background(), c, m, altered); e == nil {
		t.Fatal("trusted forged score with recomputed seal")
	}
	_ = json.Unmarshal(mustJSON(r), &copy)
	copy.Budget[len(copy.Budget)-1].Actual = nil
	altered, _ = EncodeReport(copy)
	if _, e := Replay(context.Background(), c, m, altered); e == nil {
		t.Fatal("missing settlement accepted")
	}
	var seal map[string]any
	_ = json.Unmarshal(raw, &seal)
	seal["sha256"] = strings.Repeat("0", 64)
	if _, e := Replay(context.Background(), c, m, mustJSON(seal)); e == nil {
		t.Fatal("seal ignored")
	}
	s := Summarize(r)
	encoded := string(mustJSON(s))
	for _, secret := range []string{"offline-live-credential", "generation-", "reservation-", "SYNTHETIC STORE", "MILK", "sourceText"} {
		if strings.Contains(encoded, secret) {
			t.Fatal("private data in summary", secret)
		}
	}
	if !reflect.DeepEqual(s.Cases[4].NativeBinding, "image_review_required") {
		t.Fatal("scan claims native binding")
	}
}

func TestQUnknownReceiptTransportFailureIsNotRecognition(t *testing.T) {
	c := corpus(t)
	f := &fakeHTTP{c: c, fault: "timeout", faultAt: 8}
	m, r := execute(t, c, f, &fakeBudget{guard: true})
	if r.Core.Decision != "INCONCLUSIVE" || f.posts != 8 || r.Core.Results[7].Measurement != nil {
		t.Fatal("transport failure counted as unknown recognition", r.Core)
	}
	raw, _ := EncodeReport(r)
	if _, e := Replay(context.Background(), c, m, raw); e != nil {
		t.Fatal(e)
	}
}
func TestQGoldenNeverEntersModelInput(t *testing.T) {
	c := corpus(t)
	var golden map[string]any
	_ = json.Unmarshal(c.Cases[5].ReceiptGolden, &golden)
	golden["seller"] = "GOLDEN_ONLY_MARKER_913"
	c.Cases[5].ReceiptGolden = mustJSON(golden)
	f := &fakeHTTP{c: c}
	_, r := execute(t, c, f, &fakeBudget{guard: true})
	if r.Core.Decision != "PASS_SYNTHETIC" {
		t.Fatal(r.Core.Decision)
	}
	for _, wire := range f.wire {
		if strings.Contains(string(mustJSON(wire["messages"])), "GOLDEN_ONLY_MARKER_913") {
			t.Fatal("golden leaked into model input")
		}
	}
}
func TestQSummaryRejectsPayloadAndSemanticForgeries(t *testing.T) {
	c := corpus(t)
	m, r := execute(t, c, &fakeHTTP{c: c}, &fakeBudget{guard: true})
	s := Summarize(r)
	for _, mutate := range []func(*Summary){func(s *Summary) { s.Cases[0].CheckResults[0].Code = "PRIVATE_PAYLOAD" }, func(s *Summary) { s.Cases[0].Case = "PRIVATE" }, func(s *Summary) { s.Cases[0].Reason = "PRIVATE" }, func(s *Summary) { s.Cases[0].Settlement = "PRIVATE" }, func(s *Summary) { s.POST = 9 }, func(s *Summary) { s.KnownCostMicros = 2000001 }, func(s *Summary) { s.ImportReady = true }, func(s *Summary) { s.Cases[0].FailedChecks = 1 }} {
		var copy Summary
		_ = json.Unmarshal(mustJSON(s), &copy)
		mutate(&copy)
		if _, e := ValidateSummary(mustJSON(copy), m); e == nil {
			t.Fatal("public forgery accepted")
		}
	}
	var boolForgery map[string]any
	_ = json.Unmarshal(mustJSON(s), &boolForgery)
	boolForgery["post"] = true
	if _, e := ValidateSummary(mustJSON(boolForgery), m); e == nil {
		t.Fatal("boolean accepted as count")
	}
	duplicate := strings.Replace(string(mustJSON(s)), `"post":8`, `"post":8,"Post":8`, 1)
	if _, e := ValidateSummary([]byte(duplicate), m); e == nil {
		t.Fatal("case alias duplicate accepted")
	}
}

func TestRResearchOmittedRequestCampaignAndStrictReplay(t *testing.T) {
	c := corpus(t)
	for _, mode := range []string{"", assistant.RequestFeeRouterCeilingOnly} {
		t.Run("mode="+mode, func(t *testing.T) {
			member := candidate()
			member.RequestFeeEvidence = mode
			member.Policy.RequestFeeMicros = 1000
			member.Policy.ImageFeeMicros = 1
			m := NewManifest(c, strings.Repeat("a", 40), member)
			f := &fakeHTTP{c: c, fault: "request_omitted"}
			b := &fakeBudget{guard: true}
			r, err := Run(context.Background(), c, m, grant(m), "offline-live-credential", f, b)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "" {
				if r.Core.Decision != "INCONCLUSIVE" || f.posts != 0 || f.gets != 3 || b.settled != 0 {
					t.Fatal("strict omission changed", r.Core.Decision, f.posts, f.gets)
				}
			} else {
				if r.Core.Decision != "PASS_SYNTHETIC" || f.posts != 8 || f.gets != 6 || b.settled != 8 {
					t.Fatal("cap campaign", r.Core.Decision, f.posts, f.gets)
				}
				if len(r.Core.Results[3].Measurement.Checks) != 297 || len(r.Core.Results[4].Measurement.Checks) != 297 {
					t.Fatal("full32 reduced")
				}
				for _, wire := range f.wire {
					caps := wire["provider"].(map[string]any)["max_price"].(map[string]any)
					if caps["request"] != 0.001 || caps["image"] != 0.000001 {
						t.Fatal("wire cap missing", caps)
					}
				}
			}
			encoded, err := EncodeReport(r)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := Replay(context.Background(), c, m, encoded)
			if err != nil {
				t.Fatal("replay", err)
			}
			if replay.Core.Decision != r.Core.Decision {
				t.Fatal("replay changed")
			}
			if _, err = ValidateSummary(mustJSON(Summarize(replay)), m); err != nil {
				t.Fatal("projection", err)
			}
		})
	}
}
func TestRManifestRejectsInvalidRequestEvidence(t *testing.T) {
	c := corpus(t)
	for _, tc := range []struct {
		mode string
		cap  int64
	}{{"unknown", 1000}, {assistant.RequestFeeRouterCeilingOnly, 0}, {assistant.RequestFeeRouterCeilingOnly, -1}} {
		member := candidate()
		member.RequestFeeEvidence = tc.mode
		member.Policy.RequestFeeMicros = tc.cap
		m := NewManifest(c, strings.Repeat("a", 40), member)
		if m.Validate(c, m.Source, true) == nil {
			t.Fatal("invalid manifest evidence", tc)
		}
	}
}

func TestRCapOnlyFailuresStopWithoutFollowingPOST(t *testing.T) {
	c := corpus(t)
	for _, fault := range []string{"quality", "timeout", "wrong_model", "unknown_cost", "cost_overrun", "zdr", "text_only"} {
		t.Run(fault, func(t *testing.T) {
			member := candidate()
			member.RequestFeeEvidence = assistant.RequestFeeRouterCeilingOnly
			member.Policy.RequestFeeMicros = 1000
			member.Policy.ImageFeeMicros = 1
			m := NewManifest(c, strings.Repeat("a", 40), member)
			f := &fakeHTTP{c: c, fault: fault, requestOmitted: true}
			b := &fakeBudget{guard: true}
			if fault == "quality" {
				f.mutate = func(_ int, raw json.RawMessage) json.RawMessage {
					var v map[string]any
					_ = json.Unmarshal(raw, &v)
					v["rows"] = v["rows"].([]any)[:1]
					return mustJSON(v)
				}
			}
			r, err := Run(context.Background(), c, m, grant(m), "offline-live-credential", f, b)
			if err != nil {
				t.Fatal(err)
			}
			if f.posts > 1 || f.gets > 6 || r.Core.Decision == "PASS_SYNTHETIC" {
				t.Fatal("unsafe continuation", r.Core.Decision, f.posts, f.gets)
			}
			for _, v := range r.Core.Results[1:] {
				if v.State != "not_sent" {
					t.Fatal("sent after stop", v)
				}
			}
			if fault == "quality" && (r.Core.Decision != "REJECT" || b.settled != 1) {
				t.Fatal("quality bill/decision", r.Core.Decision, b.settled)
			}
			if fault == "unknown_cost" && b.unknown != 1 {
				t.Fatal("unknown bill not frozen", b.unknown)
			}
			raw, err := EncodeReport(r)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Replay(context.Background(), c, m, raw); err != nil {
				t.Fatal("failure replay", err)
			}
		})
	}
}
