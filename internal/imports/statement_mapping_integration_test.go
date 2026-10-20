//go:build integration

package imports

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

type mappingPDFExtractor struct{}

func (mappingPDFExtractor) Extract(ctx context.Context, b []byte) (statementdata.Document, error) {
	d, e := (staticPDFExtractor{}).Extract(ctx, b)
	d.Spans = append(d.Spans, statementdata.Span{ID: "service-note", Page: 1, Block: 7, Line: 1, Word: 1, Box: statementdata.Box{XMin: 1, YMin: 140, XMax: 100, YMax: 150}, Text: "Synthetic privacy canary"})
	return d, e
}
func TestStatementMappingSnapshotLifecycle(t *testing.T) {
	ctx, p, r := statementSetup(t)
	r.SetPDFExtractor(mappingPDFExtractor{})
	var before int
	_ = p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&before)
	s, _, e := r.UploadStatement(ctx, []byte("%PDF-mapping-regression"), accountID, "mapping.pdf", "mapping-upload")
	if e != nil {
		t.Fatal(e)
	}
	proposal := manualStatementProposal()
	proposal.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{"service-note"}, Reason: "Synthetic service reason canary"}}
	input := StatementManualDraft{ExpectedRevision: s.Revision, Proposal: proposal, Reason: "Synthetic review save"}
	s, e = r.ManualStatementDraft(ctx, s.ID, "mapping-save", input)
	if e != nil || s.DraftMapping == nil || s.DraftMapping.Status != "complete" || len(s.DraftMapping.UnmappedSpanIDs) != 0 || s.DraftMapping.Proposal.Service[0].Reason != proposal.Service[0].Reason {
		t.Fatal(s.DraftMapping, e)
	}
	replay, e := r.ManualStatementDraft(ctx, s.ID, "mapping-save", input)
	if e != nil || replay.Revision != s.Revision {
		t.Fatal("save replay", e)
	}
	_, e = r.ManualStatementDraft(ctx, s.ID, "mapping-stale", input)
	requireCode(t, e, "revision_conflict")
	s, e = r.EditStatement(ctx, s.ID, "mapping-owner", StatementEdit{ExpectedRevision: s.Revision, AccountID: accountID, Retention: "keep", RolesConfirmed: true, ReviewedPages: []int{1}, Reason: "Checked source"})
	if e != nil || s.DraftMapping.Status != "complete" || s.DraftMapping.Revision != s.Revision {
		t.Fatal("owner patch lost mapping", e)
	}
	tx, e := r.begin(ctx, true)
	if e != nil {
		t.Fatal(e)
	}
	_, e = r.MaterializeStatementInTx(ctx, tx, s.ID, s.Revision, proposal, "model", "Synthetic model transaction rollback", "")
	if e != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(e)
	}
	_ = tx.Rollback(ctx)
	persisted, e := r.Statement(ctx, s.ID)
	if e != nil || persisted.Revision != s.Revision {
		t.Fatal("rollback leaked mapping", e)
	}
	var origin string
	if e = p.QueryRow(ctx, "SELECT mapping_snapshot->>'origin' FROM statement_sources WHERE id=$1", s.ID).Scan(&origin); e != nil || origin != "owner" {
		t.Fatal(origin, e)
	}
	var leaked bool
	if e = p.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM statement_audit WHERE statement_id=$1 AND metadata::text LIKE '%Synthetic service reason canary%')", s.ID).Scan(&leaked); e != nil || leaked {
		t.Fatal("service audit leak", e)
	}
	// Legacy projection is read-only and does not invent classification of service.
	_, e = p.Exec(ctx, "UPDATE statement_sources SET review=review WHERE id=$1", s.ID)
	if e != nil {
		t.Fatal(e)
	}
	legacy, e := r.Statement(ctx, s.ID)
	if e != nil || legacy.Revision != s.Revision || legacy.DraftMapping.Status != "reconstructed_partial" || len(legacy.DraftMapping.Proposal.Rows) != 1 || legacy.DraftMapping.UnmappedSpanIDs != nil || len(legacy.DraftMapping.UnclassifiedSpanIDs) != 1 {
		t.Fatal("legacy projection", legacy.DraftMapping, e)
	}
	// A blocked-but-reviewable selection still gets a complete snapshot.
	blocked := proposal
	blocked.Service = nil
	updated, e := r.ManualStatementDraft(ctx, s.ID, "mapping-blocked", StatementManualDraft{ExpectedRevision: s.Revision, Proposal: blocked, Reason: "Synthetic explicit partial edit"})
	if e != nil || updated.DraftMapping.Status != "complete" || len(updated.DraftMapping.UnmappedSpanIDs) != 1 || updated.RolesConfirmed || len(updated.ReviewedPages) != 0 {
		t.Fatal("blocked review lost snapshot", e)
	}
	// Direct document tombstone must clear snapshot through DB lifecycle, not just API.
	_, e = p.Exec(ctx, "UPDATE documents SET content=NULL,original_deleted_at=now() WHERE id=$1", s.DocumentID)
	if e != nil {
		t.Fatal(e)
	}
	var cleared bool
	if e = p.QueryRow(ctx, "SELECT mapping_snapshot IS NULL AND layout IS NULL FROM statement_sources WHERE id=$1", s.ID).Scan(&cleared); e != nil || !cleared {
		t.Fatal("direct deletion retained mapping", e)
	}
	gone, e := r.Statement(ctx, s.ID)
	if e != nil || gone.DraftMapping.Proposal != nil || gone.DraftMapping.Status != "unavailable" {
		t.Fatal("deleted mapping exposed", e)
	}
	// Expiry traverses the same cleanup path for a source with a saved mapping.
	exp, _, e := r.UploadStatement(ctx, []byte("%PDF-mapping-expiry"), accountID, "expiry.pdf", "mapping-expiry-upload")
	if e != nil {
		t.Fatal(e)
	}
	exp, e = r.ManualStatementDraft(ctx, exp.ID, "mapping-expiry-save", StatementManualDraft{ExpectedRevision: exp.Revision, Proposal: proposal, Reason: "Synthetic expiry"})
	if e != nil {
		t.Fatal(e)
	}
	r.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
	exp, e = r.Statement(ctx, exp.ID)
	r.now = time.Now
	if e != nil || exp.DraftMapping.Proposal != nil || exp.Status != "expired" {
		t.Fatal("expiry retained mapping", e)
	}
	var after int
	_ = p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&after)
	if after != before {
		t.Fatal("mapping wrote ledger")
	}
	// Public wire format must distinguish unknown coverage from a known empty list.
	b, _ := json.Marshal(legacy.DraftMapping)
	var wire map[string]any
	_ = json.Unmarshal(b, &wire)
	if wire["unmappedSpanIds"] != nil {
		t.Fatal("legacy coverage fabricated")
	}
}
