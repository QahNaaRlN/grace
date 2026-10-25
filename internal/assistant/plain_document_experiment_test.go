package assistant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestRealDocumentPlainWireAndLegacyBoundaries(t *testing.T) {
	f := &requestPricingFixture{omitRequest: true, image: "0"}
	f.inner.reply = func(v map[string]any) map[string]any {
		v["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"] = "| date | amount |\n| today | -1.00 |"
		return v
	}
	member := pricingMember()
	member.MaxImages = 5
	provider, groups, e := NewPlainDocumentExperiment(member, pricingGrant(), "offline-group-key", f)
	if e != nil {
		t.Fatal(e)
	}
	policy, e := groups.Resolve(context.Background(), "statement", 5)
	if e != nil {
		t.Fatal(e)
	}
	source := groupSource(t)
	page := source.Pages[0]
	content := []any{map[string]any{"type": "text", "text": "Read all operations"}}
	for i := 0; i < 5; i++ {
		content = append(content, map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:" + page.ContentType + ";base64," + base64.StdEncoding.EncodeToString(page.Data), "detail": "high"}})
	}
	request := LLMRequest{Policy: policy, Messages: []LLMMessage{{Role: "user", Content: content}}, InputTokenUpperBound: 30000, MaxOutputTokens: 16000}
	out, e := provider.Complete(context.Background(), request)
	if e != nil || !out.BillingKnown || string(out.Content) != "| date | amount |\n| today | -1.00 |" || f.inner.posts != 1 || f.inner.gets != 3 {
		t.Fatal(e, out, f.inner.posts, f.inner.gets)
	}
	if f.inner.wire["response_format"] != nil || f.inner.wire["tools"] != nil || f.inner.wire["plugins"] != nil {
		t.Fatal("plain request contains complex/auxiliary fields")
	}
	route := f.inner.wire["provider"].(map[string]any)
	if route["allow_fallbacks"] != false || route["zdr"] != true || route["data_collection"] != "deny" {
		t.Fatal("privacy route weakened")
	}
	actual := f.inner.wire["messages"].([]any)[0].(map[string]any)["content"].([]any)
	if len(actual) != 6 {
		t.Fatal("five pages not sent")
	}
	content = append(content, content[1])
	request.Messages[0].Content = content
	if _, e = provider.Complete(context.Background(), request); e == nil || f.inner.posts != 1 {
		t.Fatal("sixth image sent")
	}
	if _, e = NewExperimentModelGroups(member, pricingGrant(), "offline-group-key", f); e == nil {
		t.Fatal("old experiment permits five images")
	}
	raw, _ := json.Marshal(groups.groups)
	if _, e = NewModelGroups(string(raw), "offline-group-key", f); e == nil {
		t.Fatal("research entered production")
	}
	member.MaxImages = 4
	old, e := NewExperimentModelGroups(member, pricingGrant(), "offline-group-key", f)
	if e != nil {
		t.Fatal(e)
	}
	oldPolicy, e := old.Resolve(context.Background(), "statement", 1)
	if e != nil {
		t.Fatal(e)
	}
	strict, _ := NewGroupedOpenRouterProvider("offline-group-key", old, f)
	request.Policy = oldPolicy
	request.Messages[0].Content = content[:2]
	request.Schema = json.RawMessage(`{}`)
	if reply, e := strict.Complete(context.Background(), request); e == nil || !reply.BillingKnown {
		t.Fatal("old strict provider accepted plaintext or lost bill")
	}
}
