package assistant

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestQExperimentNeverBecomesProductionQualification(t *testing.T) {
	var configured []ModelGroup
	_ = json.Unmarshal([]byte(groupConfiguration(time.Now())), &configured)
	member := configured[0].Members[0]
	g := ExperimentGrant{Source: strings.Repeat("a", 40), Approval: strings.Repeat("b", 40), ManifestSHA256: strings.Repeat("c", 64), ExpiresAt: time.Now().Add(35 * time.Minute)}
	f := &groupFixture{}
	groups, e := NewExperimentModelGroups(member, g, "offline-group-key", f)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(groups.groups)
	if _, e = NewModelGroups(string(raw), "offline-group-key", f); e == nil {
		t.Fatal("research scope accepted in production config")
	}
	policy, e := groups.Resolve(context.Background(), "statement", 3)
	if e != nil || f.gets != 3 {
		t.Fatal(e, f.gets)
	}
	groups.now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	if groups.check(policy) {
		t.Fatal("expired research policy accepted")
	}
	if _, e = groups.Resolve(context.Background(), "statement", 3); e == nil || f.gets != 3 {
		t.Fatal("expired proof silently refreshed", e, f.gets)
	}
	g.ExpiresAt = time.Now().Add(-time.Second)
	if _, e = NewExperimentModelGroups(member, g, "offline-group-key", f); e == nil {
		t.Fatal("expired grant accepted")
	}
	g.ExpiresAt = time.Now().Add(time.Hour)
	if _, e = NewExperimentModelGroups(member, g, "offline-group-key", f); e == nil {
		t.Fatal("unbounded grant accepted")
	}
}
