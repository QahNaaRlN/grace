package imports

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5"
	"strings"
)

func plainSource(s Statement, expected int64) error {
	if s.Revision != expected {
		return fault("revision_conflict", 409)
	}
	if s.DeletedAt != nil || s.Rendered.Validate() != nil {
		return fault("statement_original_gone", 410)
	}
	if s.PublishedRows > 0 {
		return fault("statement_draft_published", 409)
	}
	if s.ReadingMode != "plain_review" {
		return fault("plain_review_required", 422)
	}
	return nil
}
func (r *Repository) MaterializePlainInTx(ctx context.Context, tx pgx.Tx, id string, expected int64, p statementdata.PlainReading, origin, reason, attempt string) (Statement, error) {
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return s, e
	}
	if e = plainSource(s, expected); e != nil {
		return s, e
	}
	if e = p.Normalize(); e != nil {
		return s, fault("plain_draft_invalid", 422)
	}
	state := statementdata.PlainState{SourceSHA256: s.SHA256, Candidate: p, Original: p}
	if s.Plain != nil {
		state.Original = s.Plain.Original
		if origin != "model" {
			state.Candidate.RawAnswer = s.Plain.Candidate.RawAnswer
		}
	} else if origin != "model" {
		state.Candidate.RawAnswer = ""
		state.Original.RawAnswer = ""
	}
	review, e := state.Candidate.Review()
	if e != nil {
		return s, fault("plain_draft_invalid", 422)
	}
	s, e = r.materializeStatementReview(ctx, tx, s, review, nil, origin, reason, attempt)
	if e != nil {
		return s, e
	}
	raw, e := json.Marshal(state)
	if e != nil {
		return s, e
	}
	if _, e = tx.Exec(ctx, "UPDATE statement_sources SET plain_state=$2 WHERE id=$1", id, raw); e != nil {
		return s, e
	}
	return loadStatement(ctx, tx, id, false)
}
func applyPlainFlags(s *Statement) {
	if s.Plain == nil || s.Plain.SourceSHA256 != s.SHA256 {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "plain_review_required")
		return
	}
	review, e := s.Plain.Candidate.Review()
	if e != nil {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "plain_draft_invalid")
		return
	}
	*s.Review = review
	if s.Plain.ReviewedDigest != s.Plain.CurrentDigest() {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "owner_document_review_required")
	}
	if s.DeletedAt != nil || len(s.Rendered.Pages) != s.PageCount {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "statement_original_gone")
	}
}
func (r *Repository) PlainDraft(ctx context.Context, id, key string, in StatementManualDraft) (Statement, error) {
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 2000 || in.Plain == nil {
		return Statement{}, fault("reason_required", 422)
	}
	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, e
	}
	defer tx.Rollback(ctx)
	var receipt struct{ ID string }
	replay, e := request(ctx, tx, "statement-plain:"+id, key, in, &receipt)
	if e != nil {
		return Statement{}, e
	}
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return s, e
	}
	if replay {
		applyStatementFlags(&s)
		return s, nil
	}
	if in.SourceSHA256 != s.SHA256 {
		return s, fault("revision_conflict", 409)
	}
	p := *in.Plain
	p.ParseStatus = "manual"
	s, e = r.MaterializePlainInTx(ctx, tx, id, in.ExpectedRevision, p, "manual", in.Reason, "")
	if e != nil {
		return s, e
	}
	receipt.ID = id
	if e = remember(ctx, tx, "statement-plain:"+id, key, in, receipt); e != nil {
		return s, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	applyStatementFlags(&s)
	return s, nil
}

type PlainReviewInput struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	SourceSHA256     string `json:"sourceSHA256"`
	DraftDigest      string `json:"draftDigest"`
	ReviewedPages    []int  `json:"reviewedPages"`
	RolesConfirmed   bool   `json:"rolesConfirmed"`
}

func (r *Repository) ReviewPlain(ctx context.Context, id, key string, in PlainReviewInput) (Statement, error) {
	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, e
	}
	defer tx.Rollback(ctx)
	var receipt struct{ ID string }
	replay, e := request(ctx, tx, "statement-plain-review:"+id, key, in, &receipt)
	if e != nil {
		return Statement{}, e
	}
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return s, e
	}
	if replay {
		applyStatementFlags(&s)
		return s, nil
	}
	if e = plainSource(s, in.ExpectedRevision); e != nil {
		return s, e
	}
	if s.Plain == nil || s.SHA256 != in.SourceSHA256 || in.DraftDigest != s.Plain.CurrentDigest() || !in.RolesConfirmed {
		return s, fault("plain_review_changed", 409)
	}
	pages := map[int]bool{}
	for _, n := range in.ReviewedPages {
		if n < 1 || n > s.PageCount || pages[n] {
			return s, fault("invalid_page_review", 422)
		}
		pages[n] = true
	}
	if len(pages) != s.PageCount {
		return s, fault("owner_page_review_required", 422)
	}
	review, e := s.Plain.Candidate.Review()
	if e != nil || len(review.Reconciliation.Blockers) > 0 {
		return s, fault("plain_draft_invalid", 422)
	}
	s.Plain.ReviewedDigest = in.DraftDigest
	raw, _ := json.Marshal(s.Plain)
	pageJSON, _ := json.Marshal(in.ReviewedPages)
	if _, e = tx.Exec(ctx, "UPDATE statement_sources SET plain_state=$2,roles_confirmed=true,reviewed_pages=$3,revision=revision+1 WHERE id=$1", id, raw, pageJSON); e != nil {
		return s, e
	}
	if e = statementAudit(ctx, tx, s, "plain_review", "Owner reviewed the document and header roles", map[string]any{"digest": in.DraftDigest, "pages": in.ReviewedPages}); e != nil {
		return s, e
	}
	receipt.ID = id
	if e = remember(ctx, tx, "statement-plain-review:"+id, key, in, receipt); e != nil {
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
