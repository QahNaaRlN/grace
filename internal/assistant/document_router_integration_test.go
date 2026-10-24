//go:build integration

package assistant

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
	"time"
)

type s56Tool struct{ calls int }

func (t *s56Tool) Call(_ context.Context, name string, _ json.RawMessage) (ToolResult, error) {
	t.calls++
	return ToolResult{Name: name}, nil
}
func TestS56QuickRouterUsesConsentAndOneMeteredGateway(t *testing.T) {
	if os.Getenv("TEST_S56_DOCUMENTS") != "true" {
		t.Skip("disposable database required")
	}
	pool, e := pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	var owner string
	if e = pool.QueryRow(context.Background(), "INSERT INTO users(display_name,role) VALUES('S56 router owner','member') RETURNING id::text").Scan(&owner); e != nil {
		t.Fatal(e)
	}
	ctx := access.WithPrincipal(context.Background(), access.Principal{ID: owner, Role: "member"})
	var config []ModelGroup
	_ = json.Unmarshal([]byte(groupConfiguration(time.Now())), &config)
	config[0].Purpose = "analysis"
	config[0].Profile = TextReasoning
	f := &groupFixture{reply: func(v map[string]any) map[string]any {
		v["choices"] = []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": `{"route":"get_accounts","arguments":"none"}`}}}
		return v
	}}
	raw, _ := json.Marshal(config)
	groups, e := NewModelGroups(string(raw), "offline-group-key", f)
	if e != nil {
		t.Fatal(e)
	}
	provider, e := NewGroupedOpenRouterProvider("offline-group-key", groups, f)
	if e != nil {
		t.Fatal(e)
	}
	h := &AIHandler{enabled: true, verifier: &PolicyVerifier{Groups: groups}, policy: ModelPolicy{Selection: &ModelSelection{Purpose: "analysis"}}, provider: provider, usage: NewUsageRepository(pool), conversations: NewConversationRepository(pool)}
	tool := &s56Tool{}
	query := Query{Utterance: "Покажи мои счета", Language: "ru", ReferenceTime: time.Now()}
	h.QuickQuery(tool).Query(ctx, query)
	if f.posts != 0 || tool.calls != 0 {
		t.Fatal("missing consent sent query")
	}
	policy, e := groups.Resolve(ctx, "analysis", 0)
	if e != nil {
		t.Fatal(e)
	}
	if e = h.conversations.Consent(ctx, "analysis", ConsentHash("analysis", policy), true); e != nil {
		t.Fatal(e)
	}
	out := h.QuickQuery(tool).Query(ctx, query)
	if out.Status != StatusCompleted || f.posts != 1 || tool.calls != 1 {
		t.Fatal(out, f.posts, tool.calls)
	}
	encoded, _ := json.Marshal(out)
	if strings.Contains(string(encoded), "confidence") {
		t.Fatal("fabricated probability")
	}
	var settled int
	if e = pool.QueryRow(ctx, "SELECT count(*) FROM ai_usage WHERE owner_id=$1 AND status='settled' AND actual_micros=10", owner).Scan(&settled); e != nil || settled != 1 {
		t.Fatal(settled, e)
	}
}
