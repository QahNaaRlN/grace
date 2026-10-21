//go:build integration

package imports

import (
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"testing"
)

func TestStatementComparisonSharedLedger(t *testing.T) {
	ctx, p, _ := statementSetup(t)
	tx, e := p.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	// This isolated test changes its temporary budget only inside a rollback.
	if _, e = tx.Exec(ctx, "UPDATE ai_experiment_budget SET allowance_micros=10000000,fees_micros=0"); e != nil {
		t.Fatal(e)
	}
	var beforePosts, beforeEvents int
	if e = tx.QueryRow(ctx, "SELECT (SELECT count(*) FROM postings),(SELECT count(*) FROM financial_events)").Scan(&beforePosts, &beforeEvents); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, "SET LOCAL ROLE grace_financial"); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, "SELECT set_config('grace.owner_id',$1,true)", access.BootstrapOwnerID); e != nil {
		t.Fatal(e)
	}
	var id, other string
	if e = tx.QueryRow(ctx, "SELECT grace_ai_reserve(208500,'statement-comparison.s111','fixed-A-B','statement')::text").Scan(&id); e != nil {
		t.Fatal(e)
	}
	var allowed bool
	if e = tx.QueryRow(ctx, "SELECT grace_statement_call_allowed($1::uuid)", id).Scan(&allowed); e != nil || !allowed {
		t.Fatal("reserved campaign not allowed", e)
	}
	if e = tx.QueryRow(ctx, "SELECT grace_ai_reserve(100,'synthetic-other','synthetic','analysis')::text").Scan(&other); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, "SELECT grace_ai_settle($1::uuid,NULL,$2)", other, "statement-qualification:"+other); e != nil {
		t.Fatal(e)
	}
	if e = tx.QueryRow(ctx, "SELECT grace_statement_call_allowed($1::uuid)", id).Scan(&allowed); e != nil || allowed {
		t.Fatal("shared uncertain bypass", e)
	}
	if _, e = tx.Exec(ctx, "SELECT grace_ai_settle($1::uuid,600,$2)", id, "statement-qualification:"+id); e != nil {
		t.Fatal(e)
	}
	var status, model, provider, purpose string
	var actual, reserved int64
	if e = tx.QueryRow(ctx, "SELECT status,model,provider,purpose,reserved_micros,actual_micros FROM ai_usage WHERE id=$1::uuid", id).Scan(&status, &model, &provider, &purpose, &reserved, &actual); e != nil || status != "settled" || model != "statement-comparison.s111" || provider != "fixed-A-B" || purpose != "statement" || reserved != 208500 || actual != 600 {
		t.Fatal("campaign accounting", e)
	}
	var remaining int64
	if e = tx.QueryRow(ctx, "SELECT grace_ai_remaining()").Scan(&remaining); e != nil || remaining != 0 {
		t.Fatal("other uncertain cleared", e)
	}
	var afterPosts, afterEvents int
	if e = tx.QueryRow(ctx, "SELECT (SELECT count(*) FROM postings),(SELECT count(*) FROM financial_events)").Scan(&afterPosts, &afterEvents); e != nil || afterPosts != beforePosts || afterEvents != beforeEvents {
		t.Fatal("financial write", e)
	}
}
