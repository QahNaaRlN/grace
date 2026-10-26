package assistant

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPlainProductWireFivePagesWithoutEtalonOrSchema(t *testing.T) {
	f := &groupFixture{fault: "parameters", reply: func(body map[string]any) map[string]any {
		body["choices"] = []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": "Ответ обычным текстом"}}}
		return body
	}}
	var config []ModelGroup
	_ = json.Unmarshal([]byte(groupConfiguration(time.Now())), &config)
	config[0].Profile = PlainDocumentReading
	config[0].Members = config[0].Members[:1]
	config[0].Members[0].MaxImages = 5
	raw, _ := json.Marshal(config)
	groups, e := NewModelGroups(string(raw), "offline", f)
	if e != nil {
		t.Fatal(e)
	}
	provider, e := NewGroupedOpenRouterProvider("offline", groups, f)
	if e != nil {
		t.Fatal(e)
	}
	r := &DocumentReasoner{Groups: groups, Provider: provider}
	doc := groupSource(t)
	doc.Text = "ETALON_MUST_NOT_BE_SENT"
	one := doc.Pages[0]
	doc.Pages = nil
	for i := 1; i <= 5; i++ {
		p := one
		p.Number = i
		doc.Pages = append(doc.Pages, p)
	}
	p, e := r.PlainPolicy(context.Background(), 5)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.PreparePlain(context.Background(), p, doc); e != nil {
		t.Fatal(e)
	}
	out, bill, e := r.CompletePlain(context.Background(), p, doc)
	if e != nil || bill == nil || bill.CostMicros != 10 || out.RawAnswer != "Ответ обычным текстом" || out.ParseStatus != "parse_error" {
		t.Fatal(out, bill, e)
	}
	wire, _ := json.Marshal(f.wire)
	reasoning, ok := f.wire["reasoning"].(map[string]any)
	if !ok || reasoning["enabled"] != false {
		t.Fatal("plain request did not preserve disabled reasoning from the successful prototype")
	}
	if strings.Contains(string(wire), "ETALON") || f.wire["response_format"] != nil || f.wire["tools"] != nil || strings.Count(string(wire), "data:image/") != 5 || f.posts != 1 || f.gets != 3 {
		t.Fatal("wrong product wire", f.posts, f.gets)
	}
	if p.Profile != "plain_document_reading" || p.MaximumMicros <= 0 {
		t.Fatal("wrong policy")
	}
	doc.Pages = append(doc.Pages, one)
	if e = r.PreparePlain(context.Background(), p, doc); e == nil || f.posts != 1 {
		t.Fatal("sixth page accepted")
	}
	if _, e = groups.Resolve(context.Background(), "receipt", 1); e == nil {
		t.Fatal("plain profile leaked into receipts")
	}
	// A strict legacy membership still requires structured-output capabilities and four images.
	strict, e := NewModelGroups(groupConfiguration(time.Now()), "offline", f)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = strict.Resolve(context.Background(), "statement", 5); e == nil {
		t.Fatal("legacy accepts fifth page")
	}
	if _, e = strict.Resolve(context.Background(), "statement", 1); e == nil {
		t.Fatal("legacy accepts absent structured_outputs")
	}
}
