//go:build integration

package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/go-chi/chi/v5"
)

type groundingPDFExtractor struct{}

func (groundingPDFExtractor) Extract(ctx context.Context, raw []byte) (statementdata.Document, error) {
	d, e := (staticPDFExtractor{}).Extract(ctx, raw)
	d.Spans[3].Text = "OWN"
	d.Spans[3].Box.XMax = 40
	extra := []statementdata.Span{
		{ID: "desc-mid", Page: 1, Block: 4, Line: 1, Word: 2, Box: statementdata.Box{XMin: 41, YMin: 60, XMax: 45, YMax: 70}, Text: "-"},
		{ID: "desc-end", Page: 1, Block: 4, Line: 1, Word: 3, Box: statementdata.Box{XMin: 50, YMin: 60, XMax: 100, YMax: 70}, Text: "TRANSFER"},
	}
	original := append([]statementdata.Span{}, d.Spans[4:]...)
	d.Spans = append(d.Spans[:4], extra...)
	d.Spans = append(d.Spans, original...)
	d.Spans = append(d.Spans,
		statementdata.Span{ID: "book-label", Page: 1, Block: 7, Line: 1, Word: 1, Box: statementdata.Box{XMin: 1, YMin: 140, XMax: 110, YMax: 150}, Text: "Opening balance"},
		statementdata.Span{ID: "book-value", Page: 1, Block: 7, Line: 1, Word: 2, Box: statementdata.Box{XMin: 140, YMin: 140, XMax: 190, YMax: 150}, Text: "100.00"})
	return d, e
}
func groundingProposal() statementdata.Proposal {
	p := manualStatementProposal()
	p.Rows[0].Description = statementdata.Selection{"merchant", "desc-mid", "desc-end"}
	p.Header.OpeningBookBalance = statementdata.Selection{"book-value"}
	p.Header.Labels["openingBookBalance"] = statementdata.Selection{"book-label"}
	return p
}
func TestStatementGroundingLegacyPreviewAndConfirmation(t *testing.T) {
	for _, mode := range []string{"gap", "book_role", "missing_evidence"} {
		for _, retention := range []string{"full", "deleted", "expired"} {
			t.Run(mode+"/"+retention, func(t *testing.T) {
				ctx, pool, r := statementSetup(t)
				r.SetPDFExtractor(groundingPDFExtractor{})
				prefix := "s110-" + mode + "-" + retention
				count := func() (int, int) {
					t.Helper()
					var posts, events int
					if e := pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM postings),(SELECT count(*) FROM financial_events)").Scan(&posts, &events); e != nil {
						t.Fatal(e)
					}
					return posts, events
				}
				beforePosts, beforeEvents := count()
				s, _, e := r.UploadStatement(ctx, []byte("%PDF-"+prefix), accountID, prefix+".pdf", prefix+"-upload")
				if e != nil {
					t.Fatal(e)
				}
				good := groundingProposal()
				s, e = r.ManualStatementDraft(ctx, s.ID, prefix+"-save", StatementManualDraft{ExpectedRevision: s.Revision, Proposal: good, Reason: "Verified complete synthetic source"})
				if e != nil || s.BatchID == nil {
					t.Fatal(e)
				}
				if len(s.Review.Rows[0].Issues) > 0 || strings.Contains(strings.Join(s.Review.Reconciliation.Blockers, "|"), "book_balance_role_unverified") {
					t.Fatal("positive mapping grounding failed", s.Review)
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
				decision.Fields.Description = "Owner renamed merchant text"
				batch, e = r.Edit(ctx, batch.ID, records.Items[0].ID, prefix+"-decision", Edit{batch.Revision, decision})
				if e != nil {
					t.Fatal(e)
				}
				s, e = r.EditStatement(ctx, s.ID, prefix+"-roles", StatementEdit{ExpectedRevision: s.Revision, AccountID: accountID, Retention: "delete_after_confirmation", RolesConfirmed: true, ReviewedPages: []int{1}, Reason: "Owner checked source roles/pages"})
				if e != nil {
					t.Fatal(e)
				}
				batch, e = r.Batch(ctx, batch.ID)
				if e != nil {
					t.Fatal(e)
				}
				selection := Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{records.Items[0].ID}}
				goodPreview, e := r.Preview(ctx, batch.ID, prefix+"-good-preview", selection)
				if e != nil || len(goodPreview.Blockers) > 0 {
					t.Fatal("manual row text/source evidence mixed", e, goodPreview.Blockers)
				}
				bad := groundingProposal()
				want := "row_1:description_evidence_gap"
				if mode == "gap" {
					bad.Rows[0].Description = statementdata.Selection{"merchant", "desc-end"}
					bad.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{"desc-mid"}, Reason: "Synthetic separator mistakenly treated as service"}}
				}
				if mode == "book_role" {
					bad.Header.Labels["openingBookBalance"] = statementdata.Selection{"available-label"}
					bad.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{"book-label"}, Reason: "Synthetic unused heading"}}
					want = "openingBookBalance:book_balance_role_unverified"
				}
				doc, e := (groundingPDFExtractor{}).Extract(ctx, nil)
				if e != nil {
					t.Fatal(e)
				}
				review, e := statementdata.ValidateProposal(doc, bad, []int{1}, true, "")
				if e != nil {
					t.Fatal(e)
				}
				if mode != "missing_evidence" && !strings.Contains(strings.Join(review.Reconciliation.Blockers, "|"), want) {
					t.Fatal("fresh guard missed", review)
				}
				if mode == "missing_evidence" {
					review.Rows[0].Evidence = nil
					want = "row_1:description_evidence_unverified"
				}
				// An older binary's cached matched/cleared review must not grant permission.
				review.Reconciliation.Blockers = []string{}
				review.Reconciliation.BookBalance = statementdata.Check{Status: "matched", Reasons: []string{}}
				for i := range review.Rows {
					review.Rows[i].Issues = []string{}
				}
				raw, _ := json.Marshal(review)
				if _, e = pool.Exec(ctx, "UPDATE statement_sources SET review=$2::jsonb,roles_confirmed=true,reviewed_pages='[1]'::jsonb WHERE id=$1", s.ID, raw); e != nil {
					t.Fatal(e)
				}
				if retention == "deleted" {
					s, e = r.DeleteStatementOriginal(ctx, s.ID, prefix+"-delete", StatementDelete{ExpectedRevision: s.Revision})
					if e != nil {
						t.Fatal(e)
					}
				}
				if retention == "expired" {
					r.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
					s, e = r.Statement(ctx, s.ID)
					r.now = time.Now
					if e != nil || s.Layout != nil || s.Status != "expired" {
						t.Fatal("expiry did not retain review", e, s)
					}
				}
				var storedBefore []byte
				var revisionBefore int64
				if e = pool.QueryRow(ctx, "SELECT review,revision FROM statement_sources WHERE id=$1", s.ID).Scan(&storedBefore, &revisionBefore); e != nil {
					t.Fatal(e)
				}
				legacy, e := r.Statement(ctx, s.ID)
				if e != nil || !legacy.RolesConfirmed || !strings.Contains(strings.Join(legacy.Review.Reconciliation.Blockers, "|"), want) {
					t.Fatal("legacy guard bypass", e, legacy.Review)
				}
				if (legacy.Layout == nil) != (retention != "full") {
					t.Fatal("unexpected retained layout")
				}
				router := chi.NewRouter()
				NewHandler(r).Routes(router)
				request := func(method, path, key string, payload any) *httptest.ResponseRecorder {
					t.Helper()
					raw, _ := json.Marshal(payload)
					req := httptest.NewRequest(method, path, bytes.NewReader(raw)).WithContext(ctx)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Idempotency-Key", prefix+key)
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)
					return w
				}
				view := request(http.MethodGet, "/statements/"+s.ID, "-get", nil)
				if view.Code != 200 || !strings.Contains(view.Body.String(), want) {
					t.Fatal("HTTP read lost diagnostic", view.Code, view.Body.String())
				}
				// Previously valid preview is rejected after re-reading current source state.
				stale := request(http.MethodPost, "/batches/"+batch.ID+"/confirmations", "-stale-confirm", Confirmation{goodPreview.ID, goodPreview.Hash})
				if stale.Code != 409 || !strings.Contains(stale.Body.String(), "preview_changed") {
					t.Fatal("stale preview accepted", stale.Code, stale.Body.String())
				}
				blocked := request(http.MethodPost, "/batches/"+batch.ID+"/previews", "-bad-preview", selection)
				var preview Preview
				if blocked.Code != 200 || json.Unmarshal(blocked.Body.Bytes(), &preview) != nil || !strings.Contains(strings.Join(preview.Blockers, "|"), want) {
					t.Fatal("HTTP preview trusted cache", blocked.Code, blocked.Body.String())
				}
				confirmed := request(http.MethodPost, "/batches/"+batch.ID+"/confirmations", "-blocked-confirm", Confirmation{preview.ID, preview.Hash})
				if confirmed.Code != 422 || !strings.Contains(confirmed.Body.String(), "preview_blocked") {
					t.Fatal("blocked preview accepted", confirmed.Code, confirmed.Body.String())
				}
				_, e = r.EditStatement(ctx, s.ID, prefix+"-revision-race", StatementEdit{ExpectedRevision: revisionBefore + 1, AccountID: accountID, Retention: "delete_after_confirmation", RolesConfirmed: true, ReviewedPages: []int{1}, Reason: "Stale or future revision"})
				// Deleted/expired originals retain the pre-existing read-only edit policy.
				if retention == "full" {
					requireCode(t, e, "revision_conflict")
				}
				var otherID string
				if e = pool.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES($1,'admin') RETURNING id::text", prefix+"-other").Scan(&otherID); e != nil {
					t.Fatal(e)
				}
				other := access.WithPrincipal(context.Background(), access.Principal{ID: otherID, Role: "admin"})
				_, e = r.Statement(other, s.ID)
				requireCode(t, e, "not_found")
				var storedAfter []byte
				var revisionAfter int64
				if e = pool.QueryRow(ctx, "SELECT review,revision FROM statement_sources WHERE id=$1", s.ID).Scan(&storedAfter, &revisionAfter); e != nil {
					t.Fatal(e)
				}
				if !bytes.Equal(storedBefore, storedAfter) || revisionBefore != revisionAfter {
					t.Fatal("read/preview/failed confirm rewrote source")
				}
				afterPosts, afterEvents := count()
				if beforePosts != afterPosts || beforeEvents != afterEvents {
					t.Fatal(fmt.Sprint("financial write on failure ", beforePosts, "/", afterPosts, " ", beforeEvents, "/", afterEvents))
				}
			})
		}
	}
}
