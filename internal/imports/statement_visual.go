package imports

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5"
	"reflect"
	"strconv"
	"strings"
)

func optionalRender(d documentdata.Document) any {
	if len(d.Pages) == 0 {
		return nil
	}
	d.Original = nil
	b, _ := json.Marshal(d)
	return b
}
func applyVisualFlags(s *Statement) {
	if s.Visual == nil {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "owner_visual_review_required")
		return
	}
	review, e := s.Visual.Candidate.Review(s.PageCount)
	if e != nil || s.Visual.SourceSHA256 != s.SHA256 {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "incomplete_reading")
		return
	}
	*s.Review = review
	if s.DeletedAt != nil || len(s.Rendered.Pages) != s.PageCount {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "statement_original_gone")
	}
	for i := range s.Visual.Candidate.Rows {
		if s.Visual.ReviewedRows[statementdata.VisualRowID(i)] != s.Visual.RowDigest(i) {
			s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "owner_visual_review_required")
			break
		}
	}
}
func visualSource(s Statement, expected int64) error {
	if s.Revision != expected {
		return fault("revision_conflict", 409)
	}
	if s.DeletedAt != nil || len(s.Rendered.Pages) != s.PageCount {
		return fault("statement_original_gone", 410)
	}
	if s.PublishedRows > 0 {
		return fault("statement_draft_published", 409)
	}
	if s.ReadingMode != "image_review" {
		return fault("native_mapping_required", 422)
	}
	return nil
}
func (r *Repository) MaterializeVisualInTx(ctx context.Context, tx pgx.Tx, id string, expected int64, reading statementdata.VisualReading, origin, reason, attempt string) (Statement, error) {
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return s, e
	}
	if e = visualSource(s, expected); e != nil {
		return s, e
	}
	review, e := reading.Review(s.PageCount)
	if e != nil {
		return s, fault(e.Error(), 422)
	}
	state := statementdata.VisualState{SourceSHA256: s.SHA256, Candidate: reading, Original: reading, ReviewedRows: map[string]string{}}
	if s.Visual != nil {
		state.Original = s.Visual.Original
		for i := range reading.Rows {
			key := statementdata.VisualRowID(i)
			if s.Visual.ReviewedRows[key] == state.RowDigest(i) {
				state.ReviewedRows[key] = state.RowDigest(i)
			}
		}
	}
	prior := s
	s, e = r.materializeStatementReview(ctx, tx, s, review, nil, origin, reason, attempt)
	if e != nil {
		return s, e
	}
	// A row edit revokes its digest; unchanged page coverage/header roles stay reviewed.
	if prior.Visual != nil && len(prior.Visual.Candidate.Rows) == len(reading.Rows) {
		pages, _ := json.Marshal(prior.ReviewedPages)
		roles := prior.RolesConfirmed && reflect.DeepEqual(prior.Visual.Candidate.Header, reading.Header)
		if _, e = tx.Exec(ctx, "UPDATE statement_sources SET roles_confirmed=$2,reviewed_pages=$3 WHERE id=$1", id, roles, pages); e != nil {
			return s, e
		}
	}
	encoded, _ := json.Marshal(state)
	_, e = tx.Exec(ctx, "UPDATE statement_sources SET visual_state=$2 WHERE id=$1", id, encoded)
	if e != nil {
		return s, e
	}
	return loadStatement(ctx, tx, id, false)
}
func (r *Repository) VisualDraft(ctx context.Context, id, key string, in StatementManualDraft) (Statement, error) {
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 2000 || in.Reading == nil {
		return Statement{}, fault("reason_required", 422)
	}
	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, e
	}
	defer tx.Rollback(ctx)
	var receipt struct{ ID string }
	replay, e := request(ctx, tx, "statement-visual:"+id, key, in, &receipt)
	if e != nil {
		return Statement{}, e
	}
	if replay {
		return loadStatement(ctx, tx, id, false)
	}
	source, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return source, e
	}
	if in.SourceSHA256 != source.SHA256 {
		return source, fault("revision_conflict", 409)
	}
	s, e := r.MaterializeVisualInTx(ctx, tx, id, in.ExpectedRevision, *in.Reading, "manual", in.Reason, "")
	if e != nil {
		return s, e
	}
	receipt.ID = id
	if e = remember(ctx, tx, "statement-visual:"+id, key, in, receipt); e != nil {
		return s, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	applyStatementFlags(&s)
	return s, nil
}

type VisualReviewInput struct {
	ExpectedRevision int64             `json:"expectedRevision"`
	SourceSHA256     string            `json:"sourceSHA256"`
	Rows             map[string]string `json:"rows"`
	ReviewedPages    []int             `json:"reviewedPages"`
	RolesConfirmed   bool              `json:"rolesConfirmed"`
}

func (r *Repository) ReviewVisual(ctx context.Context, id, key string, in VisualReviewInput) (Statement, error) {
	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, e
	}
	defer tx.Rollback(ctx)
	var receipt struct{ ID string }
	replay, e := request(ctx, tx, "statement-visual-review:"+id, key, in, &receipt)
	if e != nil {
		return Statement{}, e
	}
	if replay {
		return loadStatement(ctx, tx, id, false)
	}
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return s, e
	}
	if e = visualSource(s, in.ExpectedRevision); e != nil {
		return s, e
	}
	if s.Visual == nil || in.SourceSHA256 != s.SHA256 {
		return s, fault("revision_conflict", 409)
	}
	for k, digest := range in.Rows {
		n, e := strconv.Atoi(strings.TrimPrefix(k, "row-"))
		if e != nil || n < 1 || n > len(s.Visual.Candidate.Rows) || k != statementdata.VisualRowID(n-1) || digest != s.Visual.RowDigest(n-1) {
			return s, fault("revision_conflict", 409)
		}
	}
	seen := map[int]bool{}
	for _, p := range in.ReviewedPages {
		if p < 1 || p > s.PageCount || seen[p] {
			return s, fault("invalid_page_review", 422)
		}
		seen[p] = true
	}
	s.Visual.ReviewedRows = in.Rows
	v, _ := json.Marshal(s.Visual)
	pages, _ := json.Marshal(in.ReviewedPages)
	_, e = tx.Exec(ctx, "UPDATE statement_sources SET visual_state=$2,roles_confirmed=$3,reviewed_pages=$4,revision=revision+1 WHERE id=$1", id, v, in.RolesConfirmed, pages)
	if e != nil {
		return s, e
	}
	if e = statementAudit(ctx, tx, s, "visual_review", "Owner reviewed source images and selected row digests", map[string]any{"rows": in.Rows, "pages": in.ReviewedPages}); e != nil {
		return s, e
	}
	receipt.ID = id
	if e = remember(ctx, tx, "statement-visual-review:"+id, key, in, receipt); e != nil {
		return s, e
	}
	s, e = loadStatement(ctx, tx, id, false)
	if e != nil {
		return s, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	applyStatementFlags(&s)
	return s, nil
}
func (r *Repository) StatementPage(ctx context.Context, id string, page int) (documentdata.Page, error) {
	s, e := r.Statement(ctx, id)
	if e != nil {
		return documentdata.Page{}, e
	}
	if s.DeletedAt != nil || len(s.Rendered.Pages) == 0 {
		return documentdata.Page{}, fault("statement_original_gone", 410)
	}
	if page < 1 || page > len(s.Rendered.Pages) {
		return documentdata.Page{}, ErrNotFound
	}
	return s.Rendered.Pages[page-1], nil
}
