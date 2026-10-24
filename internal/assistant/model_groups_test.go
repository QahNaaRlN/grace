package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type groupDoer func(*http.Request) (*http.Response, error)

func (f groupDoer) Do(r *http.Request) (*http.Response, error) { return f(r) }

type groupFixture struct {
	fault       string
	posts, gets int
	wire        map[string]any
	reply       func(map[string]any) map[string]any
}

func groupConfiguration(now time.Time) string {
	members := []GroupMembership{}
	for i, name := range []string{"a", "b"} {
		members = append(members, GroupMembership{Policy: ModelPolicy{Model: "fixture/vision-" + name, Provider: "fixture/eu", ProviderName: "Fixture", InputMicrosPerMillion: 100000, OutputMicrosPerMillion: 100000, ImageTokenUpperBound: 1000, Vision: true}, Canonical: "fixture/vision-" + name + "-v1", Priority: i, MaxImages: 4, MaxImageSide: 2048, Region: "eu", Qualification: "public fixture quality evidence", QualifiedUntil: now.Add(time.Hour)})
	}
	b, _ := json.Marshal([]ModelGroup{{Revision: "fixture-group-v1", Purpose: "statement", Profile: DocumentReading, Members: members}})
	return string(b)
}
func (f *groupFixture) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "openrouter.ai" {
		return nil, errors.New("unexpected_host")
	}
	if req.Method == "POST" {
		f.posts++
		raw, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(raw, &f.wire)
		model := f.wire["model"].(string)
		body := map[string]any{"id": "fixture-generation", "model": model + "-v1", "provider": "Fixture", "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": "{\"ok\":true}"}}}, "usage": map[string]any{"prompt_tokens": 500, "completion_tokens": 10, "cost": 0.00001}, "openrouter_metadata": map[string]any{"requested": model, "strategy": "direct", "attempt": 1, "is_byok": false, "pipeline": []any{}, "endpoints": map[string]any{"available": []any{map[string]any{"model": model + "-v1", "provider": "Fixture", "selected": true}}}}}
		if f.reply != nil {
			body = f.reply(body)
		}
		return groupJSON(body), nil
	}
	f.gets++
	if req.Body != nil {
		return nil, errors.New("metadata_contains_body")
	}
	models := []any{}
	endpoints := []any{}
	for _, name := range []string{"a", "b"} {
		input := []string{"text", "image"}
		params := []string{"structured_outputs", "response_format", "max_tokens"}
		if f.fault == "image" {
			input = []string{"text"}
		}
		if f.fault == "parameters" {
			params = []string{"max_tokens"}
		}
		models = append(models, map[string]any{"id": "fixture/vision-" + name, "canonical_slug": "fixture/vision-" + name + "-v1", "architecture": map[string]any{"input_modalities": input, "output_modalities": []string{"text"}}, "supported_parameters": params})
		tag := "fixture/eu"
		if f.fault == "provider" {
			tag = "other/eu"
		}
		prompt := "0.0000001"
		if f.fault == "price" {
			prompt = "0.001"
		}
		output := 16000
		if f.fault == "context" {
			output = 100
		}
		endpoints = append(endpoints, map[string]any{"model_id": "fixture/vision-" + name, "tag": tag, "provider_name": "Fixture", "status": 0, "context_length": 50000, "max_completion_tokens": output, "supported_parameters": params, "pricing": map[string]any{"prompt": prompt, "completion": "0.0000001", "image": "0", "request": "0"}})
	}
	switch req.URL.Path {
	case "/api/v1/models":
		return groupJSON(map[string]any{"data": models}), nil
	case "/api/v1/endpoints/zdr":
		if f.fault == "zdr" {
			endpoints = nil
		}
		return groupJSON(map[string]any{"data": endpoints}), nil
	default:
		return groupJSON(map[string]any{"data": map[string]any{"endpoints": endpoints}}), nil
	}
}
func groupJSON(v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(b))}
}
func groupSource(t *testing.T) documentdata.Document {
	t.Helper()
	var b bytes.Buffer
	if e := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 32, 32))); e != nil {
		t.Fatal(e)
	}
	d, e := documentdata.Photo(b.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestDocumentGroupsFilterResolveAndFreezeSelection(t *testing.T) {
	now := time.Now()
	fixture := &groupFixture{}
	groups, e := NewModelGroups(groupConfiguration(now), "offline-group-key", fixture)
	if e != nil {
		t.Fatal(e)
	}
	p, e := groups.Resolve(context.Background(), "statement", 1)
	if e != nil || p.Model != "fixture/vision-a" {
		t.Fatal(p, e)
	}
	before := fixture.gets
	again, e := groups.Resolve(context.Background(), "statement", 4)
	if e != nil || aiHash(p) != aiHash(again) || fixture.gets != before {
		t.Fatal("lost cached exact selection", e)
	}
	// A caller cannot alter the trusted cache through the returned pointer.
	p.Selection.Canonical = "fixture/forged"
	if groups.check(p) {
		t.Fatal("forged selection accepted")
	}
	if !groups.check(again) {
		t.Fatal("caller mutated cache")
	}
	var config []ModelGroup
	_ = json.Unmarshal([]byte(groupConfiguration(now)), &config)
	config[0].Members[0].QualifiedUntil = now.Add(-time.Minute)
	raw, _ := json.Marshal(config)
	fallback, e := NewModelGroups(string(raw), "offline-group-key", fixture)
	if e != nil {
		t.Fatal(e)
	}
	other, e := fallback.Resolve(context.Background(), "statement", 1)
	if e != nil || other.Model != "fixture/vision-b" {
		t.Fatal("group is hard-coded to first model", other, e)
	}
	if fixture.posts != 0 {
		t.Fatal("selection sent document")
	}
}
func TestDocumentGroupsCapabilityFailuresNeverPost(t *testing.T) {
	for _, fault := range []string{"image", "parameters", "provider", "price", "context", "zdr"} {
		t.Run(fault, func(t *testing.T) {
			f := &groupFixture{fault: fault}
			g, e := NewModelGroups(groupConfiguration(time.Now()), "offline-group-key", f)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = g.Resolve(context.Background(), "statement", 1); e == nil {
				t.Fatal("unqualified capability accepted")
			}
			if f.posts != 0 || f.gets > 10 {
				t.Fatal(f.posts, f.gets)
			}
		})
	}
}
func TestGroupedDocumentWireAndIdentity(t *testing.T) {
	for _, fault := range []string{"none", "model", "provider", "pipeline", "attempt", "cost", "unknown_bill"} {
		t.Run(fault, func(t *testing.T) {
			f := &groupFixture{reply: func(v map[string]any) map[string]any {
				m := v["openrouter_metadata"].(map[string]any)
				switch fault {
				case "model":
					v["model"] = "fixture/other"
				case "provider":
					v["provider"] = "Other"
				case "pipeline":
					m["pipeline"] = []any{map[string]any{"id": "hidden"}}
				case "attempt":
					m["attempt"] = 2
				case "cost":
					v["usage"].(map[string]any)["cost"] = 1
				case "unknown_bill":
					delete(v["usage"].(map[string]any), "cost")
				}
				return v
			}}
			g, e := NewModelGroups(groupConfiguration(time.Now()), "offline-group-key", f)
			if e != nil {
				t.Fatal(e)
			}
			policy, e := g.Resolve(context.Background(), "statement", 1)
			if e != nil {
				t.Fatal(e)
			}
			provider, e := NewGroupedOpenRouterProvider("offline-group-key", g, f)
			if e != nil {
				t.Fatal(e)
			}
			request, e := DocumentRequest(policy, groupSource(t), "Read untrusted document", json.RawMessage(`{"type":"object"}`), 16000)
			if e != nil {
				t.Fatal(e)
			}
			out, e := provider.Complete(context.Background(), request)
			if fault == "none" && e != nil {
				t.Fatal(e)
			}
			if fault != "none" && e == nil {
				t.Fatal("unsafe response accepted")
			}
			if f.posts != 1 {
				t.Fatal(f.posts)
			}
			if fault == "none" && (!out.BillingKnown || out.CostMicros != 10) {
				t.Fatal(out)
			}
			wire, _ := json.Marshal(f.wire)
			if !bytes.Contains(wire, []byte("data:image/jpeg;base64,")) || bytes.Contains(wire, []byte("golden")) {
				t.Fatal("wrong document wire")
			}
			prefs := f.wire["provider"].(map[string]any)
			if prefs["allow_fallbacks"] != false || prefs["zdr"] != true || prefs["require_parameters"] != true || prefs["data_collection"] != "deny" {
				t.Fatal(prefs)
			}
		})
	}
}
func TestGroupedTransportRejectsStaleProofAndRemoteFiles(t *testing.T) {
	now := time.Now()
	f := &groupFixture{}
	g, e := NewModelGroups(groupConfiguration(now), "offline-group-key", f)
	if e != nil {
		t.Fatal(e)
	}
	p, e := g.Resolve(context.Background(), "statement", 1)
	if e != nil {
		t.Fatal(e)
	}
	provider, _ := NewGroupedOpenRouterProvider("offline-group-key", g, f)
	request, e := DocumentRequest(p, groupSource(t), "Read", json.RawMessage(`{"type":"object"}`), 16000)
	if e != nil {
		t.Fatal(e)
	}
	request.Messages = []LLMMessage{{Role: "user", Content: []any{map[string]any{"type": "file", "file": map[string]any{"file_data": "https://example.invalid/private.pdf"}}}}}
	if _, e = provider.Complete(context.Background(), request); e == nil || f.posts != 0 {
		t.Fatal("hidden file parser allowed")
	}
	g.now = func() time.Time { return now.Add(25 * time.Hour) }
	if _, e = provider.Complete(context.Background(), request); e == nil || f.posts != 0 {
		t.Fatal("stale proof sent")
	}
	if strings.Contains(p.Selection.ProofSHA256, "offline") {
		t.Fatal("credential leaked")
	}
}

func TestModelGroupsRefuseRoutingAliases(t *testing.T) {
	for _, id := range []string{"openrouter/auto", "fixture/model:latest", "fixture/*", "fixture/model-latest", "fixture/auto"} {
		var cfg []ModelGroup
		_ = json.Unmarshal([]byte(groupConfiguration(time.Now())), &cfg)
		cfg[0].Members[0].Policy.Model = id
		raw, _ := json.Marshal(cfg)
		if _, e := NewModelGroups(string(raw), "offline-group-key", &groupFixture{}); e == nil {
			t.Fatal("routing alias accepted", id)
		}
	}
}
func TestGroupedPrePOSTRefusalSettlesZero(t *testing.T) {
	f := &groupFixture{}
	g, e := NewModelGroups(groupConfiguration(time.Now()), "offline-group-key", f)
	if e != nil {
		t.Fatal(e)
	}
	policy, e := g.Resolve(context.Background(), "statement", 1)
	if e != nil {
		t.Fatal(e)
	}
	provider, _ := NewGroupedOpenRouterProvider("offline-group-key", g, f)
	request, e := DocumentRequest(policy, groupSource(t), "Read", json.RawMessage(`{"type":"object"}`), 16000)
	if e != nil {
		t.Fatal(e)
	}
	request.Policy.Selection.Canonical = "fixture/forged"
	meter := &failedBillingMeter{}
	_, e = MeteredCompletion(context.Background(), provider, meter, request, "statement")
	if e == nil || f.posts != 0 || meter.calls != 1 || meter.settled == nil || !meter.settled.NotSent || meter.settled.CostMicros != 0 {
		t.Fatal("unsent request froze or posted", e, f.posts, meter.settled)
	}
}

func TestGroupedReaderFull32ThroughActualAdapter(t *testing.T) {
	corpus, e := statementqualification.LoadDirectCorpus("../..")
	if e != nil {
		t.Fatal(e)
	}
	c := corpus.Cases[3]
	raw, _ := json.Marshal(c.Golden)
	var visual statementdata.VisualReading
	if e = json.Unmarshal(raw, &visual); e != nil {
		t.Fatal(e)
	}
	for _, f := range visual.Header {
		if f != nil {
			for i := range f.Evidence {
				f.Evidence[i].Box = [4]float64{0, 0, 1, 1}
			}
		}
	}
	for _, r := range visual.Rows {
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
	content, _ := json.Marshal(visual)
	f := &groupFixture{reply: func(v map[string]any) map[string]any {
		v["choices"] = []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": string(content)}}}
		return v
	}}
	groups, e := NewModelGroups(groupConfiguration(time.Now()), "offline-group-key", f)
	if e != nil {
		t.Fatal(e)
	}
	provider, e := NewGroupedOpenRouterProvider("offline-group-key", groups, f)
	if e != nil {
		t.Fatal(e)
	}
	reasoner := &DocumentReasoner{Groups: groups, Provider: provider}
	doc := documentdata.Document{Kind: "pdf", SourceSHA256: strings.Repeat("a", 64), Text: c.Input.Text}
	for _, p := range c.Input.Pages {
		page, e := documentdata.NewPage(p.Number, p.PNG)
		if e != nil {
			t.Fatal(e)
		}
		doc.Pages = append(doc.Pages, page)
	}
	policy, e := reasoner.CurrentDocumentPolicy(context.Background(), 3)
	if e != nil {
		t.Fatal(e)
	}
	if e = reasoner.PrepareDocument(context.Background(), policy, doc); e != nil {
		t.Fatal(e)
	}
	observed, bill, e := reasoner.CompleteDocument(context.Background(), policy, doc)
	if e != nil || bill == nil || f.posts != 1 || len(observed.Rows) != 32 {
		t.Fatal(e, bill, f.posts)
	}
	encoded, _ := json.Marshal(observed)
	var direct statementextract.DirectReading
	if json.Unmarshal(encoded, &direct) != nil {
		t.Fatal("decode")
	}
	measured, e := statementqualification.MeasureDirect(c, direct)
	if e != nil || !measured.RecognitionPassed || !measured.SourceBindingPassed || measured.ImportReady || len(measured.Checks) != 297 {
		t.Fatal("full scorer mismatch", e, measured.RecognitionPassed, measured.SourceBindingPassed, len(measured.Checks))
	}
	binding, e := statementextract.BindVisualReading(c.Document, observed)
	if e != nil || !binding.Passed || len(binding.Review.Rows) != 32 {
		t.Fatal("new native binder differs", e, binding.Passed)
	}
}

func TestDocumentReservationIncludesFlatFees(t *testing.T) {
	p := ModelPolicy{Model: "fixture/model", Provider: "fixture/eu", ProviderName: "Fixture", InputMicrosPerMillion: 100000, OutputMicrosPerMillion: 100000, Vision: true, ImageTokenUpperBound: 1000, RequestFeeMicros: 17, ImageFeeMicros: 23, VerifiedAt: time.Now(), Selection: &ModelSelection{Profile: DocumentReading, MaxImages: 4}}
	amount, e := p.ReservationMicros(30000, 16000)
	if e != nil || amount != 4709 {
		t.Fatal(amount, e)
	}
}

func TestImageFormatsReachGroupedImageWire(t *testing.T) {
	var pngBytes, jpg bytes.Buffer
	bitmap := image.NewRGBA(image.Rect(0, 0, 32, 32))
	if e := png.Encode(&pngBytes, bitmap); e != nil {
		t.Fatal(e)
	}
	if e := jpeg.Encode(&jpg, bitmap, nil); e != nil {
		t.Fatal(e)
	}
	webp, e := os.ReadFile("../documentdata/testdata/colors.webp")
	if e != nil {
		t.Fatal(e)
	}
	for _, raw := range [][]byte{pngBytes.Bytes(), jpg.Bytes(), webp} {
		f := &groupFixture{}
		groups, e := NewModelGroups(groupConfiguration(time.Now()), "offline-group-key", f)
		if e != nil {
			t.Fatal(e)
		}
		policy, e := groups.Resolve(context.Background(), "statement", 1)
		if e != nil {
			t.Fatal(e)
		}
		doc, e := documentdata.Photo(raw)
		if e != nil {
			t.Fatal(e)
		}
		request, e := DocumentRequest(policy, doc, "Read", json.RawMessage(`{"type":"object"}`), 16000)
		if e != nil {
			t.Fatal(e)
		}
		provider, _ := NewGroupedOpenRouterProvider("offline-group-key", groups, f)
		if _, e = provider.Complete(context.Background(), request); e != nil || f.posts != 1 {
			t.Fatal("image format did not reach gateway", e)
		}
	}
}
