//go:build integration

package imports

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

type turnoverPDFExtractor struct{}

func (turnoverPDFExtractor) Extract(ctx context.Context, raw []byte) (statementdata.Document, error) {
	d, e := (staticPDFExtractor{}).Extract(ctx, raw)
	d.Spans = append(d.Spans, statementdata.Span{ID: "bank-label", Page: 1, Block: 7, Line: 1, Word: 1, Box: statementdata.Box{XMin: 1, YMin: 140, XMax: 20, YMax: 150}, Text: "БИК"}, statementdata.Span{ID: "bank-value", Page: 1, Block: 7, Line: 1, Word: 2, Box: statementdata.Box{XMin: 25, YMin: 140, XMax: 60, YMax: 150}, Text: "12.34"})
	return d, e
}
func TestStatementTurnoverLegacyGuard(t *testing.T) {
	ctx, p, r := statementSetup(t)
	r.SetPDFExtractor(turnoverPDFExtractor{})
	var before int
	_ = p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&before)
	s, _, e := r.UploadStatement(ctx, []byte("%PDF-s13-legacy"), accountID, "s13.pdf", "s13-upload")
	if e != nil {
		t.Fatal(e)
	}
	proposal := manualStatementProposal()
	proposal.Header.DebitTurnover = statementdata.Selection{"bank-value"}
	proposal.Header.CreditTurnover = nil
	proposal.Header.Labels["debitTurnover"] = statementdata.Selection{"bank-label"}
	s, e = r.ManualStatementDraft(ctx, s.ID, "s13-save", StatementManualDraft{ExpectedRevision: s.Revision, Proposal: proposal, Reason: "Synthetic bank identifier negative"})
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(strings.Join(s.Review.Reconciliation.Blockers, "|"), "turnover_role_unverified") {
		t.Fatal("new draft role not guarded")
	}
	// Simulate an older binary's persisted review with no role blocker, while
	// preserving the exact rows and source evidence. Reading must not trust it.
	s.Review.Reconciliation.Blockers = []string{}
	s.Review.Reconciliation.Turnover = statementdata.Check{Status: "matched", Reasons: []string{}}
	raw, _ := json.Marshal(s.Review)
	_, e = p.Exec(ctx, "UPDATE statement_sources SET review=$2::jsonb,roles_confirmed=true,reviewed_pages='[1]'::jsonb WHERE id=$1", s.ID, raw)
	if e != nil {
		t.Fatal(e)
	}
	legacy, e := r.Statement(ctx, s.ID)
	if e != nil || !legacy.RolesConfirmed || !strings.Contains(strings.Join(legacy.Review.Reconciliation.Blockers, "|"), "turnover_role_unverified") {
		t.Fatal("legacy read accepted", legacy, e)
	}
	batch, e := r.Batch(ctx, *s.BatchID)
	if e != nil {
		t.Fatal(e)
	}
	records, e := r.Records(ctx, batch.ID, "", "", "", 100)
	if e != nil || len(records.Items) != 1 {
		t.Fatal(e)
	}
	decision := createDecision("purchase")
	decision.Fields.AmountMinor = 1234
	decision.Fields.OccurredAt = ""
	decision.Fields.OccurredDate = "2026-10-19"
	batch, e = r.Edit(ctx, batch.ID, records.Items[0].ID, "s13-decision", Edit{batch.Revision, decision})
	if e != nil {
		t.Fatal(e)
	}
	preview, e := r.Preview(ctx, batch.ID, "s13-preview", Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{records.Items[0].ID}})
	if e != nil || !strings.Contains(strings.Join(preview.Blockers, "|"), "turnover_role_unverified") {
		t.Fatal("preview trusted legacy numerical match", preview, e)
	}
	var after int
	_ = p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&after)
	if after != before {
		t.Fatal("financial write before confirmation")
	}
	var stored statementdata.Review
	var storedRaw []byte
	if e = p.QueryRow(ctx, "SELECT review FROM statement_sources WHERE id=$1", s.ID).Scan(&storedRaw); e != nil || json.Unmarshal(storedRaw, &stored) != nil || len(stored.Reconciliation.Blockers) != 0 {
		t.Fatal("read guard mutated stored review", e)
	}
}
