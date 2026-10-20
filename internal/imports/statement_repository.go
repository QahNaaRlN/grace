package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QahNaaRlN/grace/internal/importreview"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) SetPDFExtractor(e PDFExtractor) { r.extractor = e }
func loadStatement(ctx context.Context, tx pgx.Tx, id string, lock bool) (Statement, error) {
	var s Statement
	var layout, review, pages, mapping []byte
	query := `SELECT s.id::text,s.document_id::text,s.account_id::text,s.batch_id::text,s.revision,d.filename,d.sha256,s.status,s.retention,s.expires_at,s.deleted_at,s.created_at,s.roles_confirmed,s.reviewed_pages,s.review,s.layout,s.page_count,s.original_size_bytes,s.mapping_snapshot,(SELECT count(*) FROM import_records r WHERE r.batch_id=s.batch_id AND r.published),(SELECT count(*) FROM import_records r WHERE r.batch_id=s.batch_id AND r.published AND r.decision->>'action'='exclude') FROM statement_sources s JOIN documents d ON d.id=s.document_id AND d.owner_id=s.owner_id WHERE s.id=$1`
	if lock {
		query += " FOR UPDATE OF s"
	}
	e := tx.QueryRow(ctx, query, id).Scan(&s.ID, &s.DocumentID, &s.AccountID, &s.BatchID, &s.Revision, &s.FileName, &s.SHA256, &s.Status, &s.Retention, &s.ExpiresAt, &s.DeletedAt, &s.CreatedAt, &s.RolesConfirmed, &pages, &review, &layout, &s.PageCount, &s.SizeBytes, &mapping, &s.PublishedRows, &s.ExcludedRows)
	if errors.Is(e, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	if e != nil {
		return s, e
	}
	if e = json.Unmarshal(pages, &s.ReviewedPages); e != nil {
		return s, e
	}
	if len(layout) > 0 {
		if e = json.Unmarshal(layout, &s.Layout); e != nil {
			return s, e
		}
	}
	if len(review) > 0 {
		if e = json.Unmarshal(review, &s.Review); e != nil {
			return s, e
		}
	}
	s.DraftMapping = statementdata.MappingFor(s.Layout, s.Review, mapping, s.Revision)
	return s, nil
}
func statementAccount(ctx context.Context, tx pgx.Tx, id string) error {
	if !uuid.MatchString(id) {
		return fault("invalid_reference", 400)
	}
	var valid bool
	e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND type='debit' AND currency='RUB')", id).Scan(&valid)
	if e != nil {
		return e
	}
	if !valid {
		return fault("statement_debit_rub_account_required", 422)
	}
	return nil
}
func statementAudit(ctx context.Context, tx pgx.Tx, s Statement, action, reason string, metadata any) error {
	raw, _ := json.Marshal(metadata)
	_, e := tx.Exec(ctx, "INSERT INTO statement_audit(statement_id,revision,action,reason,metadata) VALUES($1,$2,$3,$4,$5)", s.ID, s.Revision, action, reason, raw)
	return e
}
func (r *Repository) Statement(ctx context.Context, id string) (Statement, error) {
	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, e
	}
	defer tx.Rollback(ctx)
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return s, e
	}
	if s.DeletedAt == nil && s.Retention != "keep" && !r.now().Before(s.ExpiresAt) {
		if e = r.deleteStatementOriginal(ctx, tx, &s, "expired"); e != nil {
			return s, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	applyStatementFlags(&s)
	return s, nil
}
func (r *Repository) UploadStatement(ctx context.Context, raw []byte, account, name, key string) (Statement, bool, error) {
	if len(raw) > statementdata.MaxPDFBytes {
		return Statement{}, false, fault("pdf_size_limit", 413)
	}
	if len(raw) < 5 || !bytes.HasPrefix(raw, []byte("%PDF-")) {
		return Statement{}, false, fault("invalid_pdf", 422)
	}
	payload := struct{ SHA256, AccountID string }{hashBytes(raw), account}
	// Check replay/dedup before expensive local extraction, then check again under
	// the common owner writer lock before inserting either source or document.
	check, e := r.begin(ctx, false)
	if e != nil {
		return Statement{}, false, e
	}
	var receipt struct{ ID string }
	replay, e := request(ctx, check, "statement-upload", key, payload, &receipt)
	if e == nil && replay {
		_ = check.Rollback(ctx)
		s, e := r.Statement(ctx, receipt.ID)
		return s, false, e
	}
	if e == nil {
		e = statementAccount(ctx, check, account)
	}
	var knownID string
	if e == nil {
		lookup := check.QueryRow(ctx, "SELECT s.id::text FROM statement_sources s JOIN documents d ON d.id=s.document_id AND d.owner_id=s.owner_id WHERE d.kind='bank_statement' AND d.sha256=$1", payload.SHA256).Scan(&knownID)
		if lookup != nil && !errors.Is(lookup, pgx.ErrNoRows) {
			e = lookup
		}
	}
	_ = check.Rollback(ctx)
	if e != nil {
		return Statement{}, false, e
	}
	var layout statementdata.Document
	if knownID == "" {
		if r.extractor == nil {
			return Statement{}, false, fault("extractor_unavailable", 503)
		}
		layout, e = r.extractor.Extract(ctx, raw)
		if e != nil {
			status := 422
			if e.Error() == "extraction_busy" {
				status = 429
			}
			if e.Error() == "extractor_unavailable" {
				status = 503
			}
			return Statement{}, false, fault(e.Error(), status)
		}
		if e = layout.Validate(); e != nil {
			return Statement{}, false, fault("invalid_pdf_layout", 422)
		}
	}

	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, false, e
	}
	defer tx.Rollback(ctx)
	replay, e = request(ctx, tx, "statement-upload", key, payload, &receipt)
	if e != nil {
		return Statement{}, false, e
	}
	if replay {
		s, e := loadStatement(ctx, tx, receipt.ID, false)
		return s, false, e
	}
	if e = statementAccount(ctx, tx, account); e != nil {
		return Statement{}, false, e
	}
	var existing string
	e = tx.QueryRow(ctx, `SELECT s.id::text FROM statement_sources s JOIN documents d ON d.id=s.document_id AND d.owner_id=s.owner_id WHERE d.kind='bank_statement' AND d.sha256=$1`, payload.SHA256).Scan(&existing)
	created := false
	if errors.Is(e, pgx.ErrNoRows) {
		var document string
		name = cleanFileName(name)
		if name == "" {
			name = "statement.pdf"
		}
		if len([]rune(name)) > 180 {
			name = string([]rune(name)[:180])
		}
		e = tx.QueryRow(ctx, "INSERT INTO documents(kind,filename,sha256,content) VALUES('bank_statement',$1,$2,$3) RETURNING id::text", name, payload.SHA256, raw).Scan(&document)
		if e != nil {
			return Statement{}, false, e
		}
		encoded, _ := json.Marshal(layout)
		e = tx.QueryRow(ctx, "INSERT INTO statement_sources(document_id,account_id,layout,page_count,original_size_bytes) VALUES($1,$2,$3,$4,$5) RETURNING id::text", document, account, encoded, len(layout.Pages), len(raw)).Scan(&existing)
		created = true
	}
	if e != nil {
		return Statement{}, false, e
	}
	s, e := loadStatement(ctx, tx, existing, false)
	if e != nil {
		return s, false, e
	}
	if s.AccountID != account {
		return s, false, fault("document_binding_conflict", 409)
	}
	receipt.ID = s.ID
	if e = remember(ctx, tx, "statement-upload", key, payload, receipt); e != nil {
		return s, false, e
	}
	if created {
		if e = statementAudit(ctx, tx, s, "upload", "Owner uploaded PDF", map[string]any{"extractor": layout.Extractor, "pages": len(layout.Pages)}); e != nil {
			return s, false, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return s, false, e
	}
	applyStatementFlags(&s)
	return s, created, nil
}
func sourceFingerprint(rec importreview.Record) string {
	if rec.PDF != nil {
		return digest(struct {
			DocumentID string
			Row        statementdata.Row
		}{rec.PDF.DocumentID, rec.PDF.Row})
	}
	return digest(rec.Raw)
}
func statementRecord(s Statement, row statementdata.Row) importreview.Record {
	n := importreview.Fields{Currency: "RUB", AccountID: s.AccountID, Description: row.Description}
	description := strings.ToLower(row.Description)
	for _, hint := range []string{"кубышк", "заем", "заём", "займ", "кредитные средства", "loan proceeds"} {
		if strings.Contains(description, hint) {
			n.Funding = &importreview.Funding{Kind: "borrowed"}
			break
		}
	}
	if row.AccountAmount != nil {
		amount := row.AccountAmount.AmountMinor
		n.AmountMinor = &amount
		n.Currency = row.AccountAmount.Currency
	}
	if row.OperationDate != nil {
		date := row.OperationDate
		n.Date = &importreview.Date{Local: date.Local[:10], Precision: "date", TimeZone: date.TimeZone}
		if date.UTC == "" {
			reason := "source_date_only"
			if date.Precision == "local_datetime" {
				reason = "timezone_unknown"
			}
			n.Labels = map[string]string{"dateProjectionReason": reason}
		}
		if date.UTC != "" {
			n.Date = &importreview.Date{Local: date.Local, Precision: "datetime", TimeZone: date.TimeZone, UTC: date.UTC}
		}
	}
	issues := []importreview.Issue{}
	for _, code := range row.Issues {
		issues = append(issues, importreview.Issue{Code: code, Severity: "error", Coordinate: fmt.Sprintf("pdf:%d", row.Ordinal), Message: code})
	}
	return importreview.Record{ID: fmt.Sprintf("pdf-%s-%d", s.ID, row.Ordinal), SourceKind: "bank_statement_pdf", PDF: &importreview.PDFSource{StatementID: s.ID, DocumentID: s.DocumentID, AccountID: s.AccountID, Ordinal: row.Ordinal, Row: row}, Domain: "transaction", Normalized: n, Issues: issues, Raw: map[string]importreview.Cell{}}
}
func (r *Repository) ManualStatementDraft(ctx context.Context, id, key string, in StatementManualDraft) (Statement, error) {
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 2000 {
		return Statement{}, fault("reason_required", 422)
	}
	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, e
	}
	defer tx.Rollback(ctx)
	var receipt struct{ ID string }
	replay, e := request(ctx, tx, "statement-manual:"+id, key, in, &receipt)
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
	s, e = r.MaterializeStatementInTx(ctx, tx, id, in.ExpectedRevision, in.Proposal, "owner", in.Reason, "")
	if e != nil {
		return s, e
	}
	receipt.ID = id
	if e = remember(ctx, tx, "statement-manual:"+id, key, in, receipt); e != nil {
		return s, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	applyStatementFlags(&s)
	return s, nil
}
func applyStatementFlags(s *Statement) {
	if s.Review == nil {
		return
	}
	statementdata.ApplyTurnoverRoleChecks(s.Layout, s.Review)
	statementdata.ApplyGroundingChecks(s.Layout, s.Review)
	coverage := statementdata.Check{Status: "incomplete", Reasons: []string{}}
	if s.PublishedRows == len(s.Review.Rows) && s.ExcludedRows == 0 {
		coverage.Status = "represented"
	} else if s.ExcludedRows > 0 {
		coverage.Reasons = append(coverage.Reasons, "excluded_rows_remain_visible")
	} else {
		coverage.Reasons = append(coverage.Reasons, "pending_financial_decisions")
	}
	s.Review.Reconciliation.FinancialCoverage = coverage
	if !s.RolesConfirmed {
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "owner_field_role_review_required")
	}
	pages := map[int]bool{}
	for _, p := range s.ReviewedPages {
		pages[p] = true
	}
	required := s.PageCount
	if len(pages) != required || required == 0 {
		s.Review.Reconciliation.RowCoverage.Status = "not_reviewed"
		s.Review.Reconciliation.Blockers = append(s.Review.Reconciliation.Blockers, "owner_page_review_required")
	}
}
func (r *Repository) deleteStatementOriginal(ctx context.Context, tx pgx.Tx, s *Statement, status string) error {
	if s.DeletedAt != nil {
		return nil
	}
	_, e := tx.Exec(ctx, "UPDATE documents SET content=NULL,original_deleted_at=$2 WHERE id=$1 AND kind='bank_statement' AND content IS NOT NULL", s.DocumentID, r.now())
	if e != nil {
		return e
	}
	fresh, e := loadStatement(ctx, tx, s.ID, false)
	if e != nil {
		return e
	}
	*s = fresh
	if status == "expired" {
		if _, e = tx.Exec(ctx, "UPDATE statement_sources SET status='expired' WHERE id=$1", s.ID); e != nil {
			return e
		}
		s.Status = "expired"
	}

	// Every record/report stores only the selected field fragments. Full page
	// state lives exclusively in layout and is removed with the original.
	return statementAudit(ctx, tx, *s, "delete_original", "Original removed: "+status, map[string]string{"backups": "retained_until_backup_expiry"})
}
func (r *Repository) DeleteStatementOriginal(ctx context.Context, id, key string, in StatementDelete) (Statement, error) {
	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, e
	}
	defer tx.Rollback(ctx)
	var receipt struct{ ID string }
	replay, e := request(ctx, tx, "statement-delete:"+id, key, in, &receipt)
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
	if s.DeletedAt == nil && s.Revision != in.ExpectedRevision {
		return s, fault("revision_conflict", 409)
	}
	if e = r.deleteStatementOriginal(ctx, tx, &s, "deleted"); e != nil {
		return s, e
	}
	receipt.ID = id
	if e = remember(ctx, tx, "statement-delete:"+id, key, in, receipt); e != nil {
		return s, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	applyStatementFlags(&s)
	return s, nil
}
func (r *Repository) StatementOriginal(ctx context.Context, id string) ([]byte, string, error) {
	s, e := r.Statement(ctx, id)
	if e != nil {
		return nil, "", e
	}
	if s.DeletedAt != nil {
		return nil, "", fault("statement_original_gone", 410)
	}
	tx, e := r.begin(ctx, false)
	if e != nil {
		return nil, "", e
	}
	defer tx.Rollback(ctx)
	var raw []byte
	e = tx.QueryRow(ctx, "SELECT content FROM documents WHERE id=$1 AND kind='bank_statement'", s.DocumentID).Scan(&raw)
	if e != nil {
		return nil, "", e
	}
	if raw == nil {
		return nil, "", fault("statement_original_gone", 410)
	}
	return raw, s.FileName, nil
}
func (r *Repository) EditStatement(ctx context.Context, id, key string, in StatementEdit) (Statement, error) {
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 2000 {
		return Statement{}, fault("reason_required", 422)
	}
	if in.Retention != "keep" && in.Retention != "delete_after_confirmation" {
		return Statement{}, fault("invalid_retention", 422)
	}
	tx, e := r.begin(ctx, true)
	if e != nil {
		return Statement{}, e
	}
	defer tx.Rollback(ctx)
	var receipt struct{ ID string }
	replay, e := request(ctx, tx, "statement-edit:"+id, key, in, &receipt)
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
	if s.Revision != in.ExpectedRevision {
		return s, fault("revision_conflict", 409)
	}
	if s.DeletedAt != nil {
		return s, fault("statement_original_gone", 410)
	}
	if e = statementAccount(ctx, tx, in.AccountID); e != nil {
		return s, e
	}
	pages := map[int]bool{}
	for _, p := range in.ReviewedPages {
		if s.Layout == nil || p < 1 || p > len(s.Layout.Pages) || pages[p] {
			return s, fault("invalid_page_review", 422)
		}
		pages[p] = true
	}
	if s.AccountID != in.AccountID && s.BatchID != nil {
		var published bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM import_records WHERE batch_id=$1 AND published)", *s.BatchID).Scan(&published); e != nil {
			return s, e
		}
		if published {
			return s, fault("statement_binding_published", 409)
		}
		// Function has a closed JSON patch and explicit owner checks; the ordinary
		// financial role never receives generic UPDATE access to source evidence.
		if _, e = tx.Exec(ctx, "SELECT grace_statement_rebind($1,$2)", s.ID, in.AccountID); e != nil {
			return s, e
		}
	}
	encoded, _ := json.Marshal(in.ReviewedPages)
	if in.ReviewedPages == nil {
		encoded = []byte("[]")
	}
	_, e = tx.Exec(ctx, "UPDATE statement_sources SET account_id=$2,retention=$3,roles_confirmed=$4,reviewed_pages=$5,revision=revision+1 WHERE id=$1", id, in.AccountID, in.Retention, in.RolesConfirmed, encoded)
	if e != nil {
		return s, e
	}
	if s.BatchID != nil {
		if _, e = tx.Exec(ctx, "UPDATE import_batches SET revision=revision+1 WHERE id=$1", *s.BatchID); e != nil {
			return s, e
		}
	}
	s, e = loadStatement(ctx, tx, id, false)
	if e != nil {
		return s, e
	}
	if e = statementAudit(ctx, tx, s, "owner_review", in.Reason, map[string]any{"accountId": in.AccountID, "retention": in.Retention, "rolesConfirmed": in.RolesConfirmed, "reviewedPages": in.ReviewedPages}); e != nil {
		return s, e
	}
	receipt.ID = id
	if e = remember(ctx, tx, "statement-edit:"+id, key, in, receipt); e != nil {
		return s, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	applyStatementFlags(&s)
	return s, nil
}

// MaterializeStatementInTx is shared by local/manual and AI orchestration. The
// caller holds the common owner writer lock and commits source/batch/attempt
// together. It never calls a provider or writes a financial fact.
func (r *Repository) MaterializeStatementInTx(ctx context.Context, tx pgx.Tx, id string, expectedRevision int64, proposal statementdata.Proposal, origin, reason, attemptID string) (Statement, error) {
	s, e := loadStatement(ctx, tx, id, true)
	if e != nil {
		return s, e
	}
	if s.Revision != expectedRevision {
		return s, fault("revision_conflict", 409)
	}
	if s.DeletedAt != nil || s.Layout == nil || s.Retention != "keep" && !r.now().Before(s.ExpiresAt) {
		return s, fault("statement_original_gone", 410)
	}
	if s.BatchID != nil {
		var published bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM import_records WHERE batch_id=$1 AND published)", *s.BatchID).Scan(&published); e != nil {
			return s, e
		}
		if published {
			return s, fault("statement_draft_published", 409)
		}
	}

	pages := []int{}
	for _, p := range s.Layout.Pages {
		pages = append(pages, p.Number)
	}
	review, e := statementdata.ValidateProposal(*s.Layout, proposal, pages, true, "")
	if e != nil {
		return s, fault(e.Error(), 422)
	}
	report := importreview.Report{SchemaVersion: importreview.ReportVersion, Source: importreview.Metadata{SourceID: "bank_statement:" + s.ID}, PayloadSHA256: s.SHA256, Records: []importreview.Record{}, Candidates: []importreview.Candidate{}, Issues: []importreview.Issue{}, DomainCounts: map[string]int{"transaction": len(review.Rows)}}
	for _, row := range review.Rows {
		rec := statementRecord(s, row)
		rec.PDF.ProposalOrigin = origin
		rec.PDF.AttemptID = attemptID
		report.Records = append(report.Records, rec)
	}
	encoded, _ := json.Marshal(report)
	// Snapshot is a small source manifest, not a hidden copy of full PDF text.
	manifest, _ := json.Marshal(map[string]string{"schema": statementdata.SchemaVersion, "statementId": s.ID, "documentId": s.DocumentID, "sha256": s.SHA256})
	var batch string
	if s.BatchID != nil {
		e = tx.QueryRow(ctx, "SELECT grace_statement_prepare_regroup($1,$2,$3)::text", s.ID, encoded, reason).Scan(&batch)
	} else {
		e = tx.QueryRow(ctx, `INSERT INTO import_batches(source_id,payload_hash,snapshot,report,format,file_name) VALUES($1,$2,$3,$4,'bank_statement_pdf',$5) RETURNING id::text`, report.Source.SourceID, digest(report.Source), manifest, encoded, s.FileName).Scan(&batch)
		if e != nil {
			return s, e
		}
	}
	if e != nil {
		return s, e
	}

	for _, rec := range report.Records {
		source, _ := json.Marshal(rec)
		_, e = tx.Exec(ctx, "INSERT INTO import_records(batch_id,snapshot_record_id,domain,source,raw_hash,questions) VALUES($1,$2,'transaction',$3,$4,$5)", batch, rec.ID, source, sourceFingerprint(rec), len(rec.Issues)+1)
		if e != nil {
			return s, e
		}
	}
	over, e := statementCandidateLimit(ctx, tx, batch)
	if e != nil {
		return s, e
	}
	if over {
		return s, fault("statement_candidate_limit", 422)
	}

	encoded, _ = json.Marshal(review)
	snapshot, e := json.Marshal(statementdata.NewMappingSnapshot(*s.Layout, proposal, review, origin, s.Revision+1))
	if e != nil {
		return s, e
	}
	_, e = tx.Exec(ctx, "UPDATE statement_sources SET batch_id=$2,review=$3,revision=revision+1,status='reviewing',roles_confirmed=false,reviewed_pages='[]',mapping_snapshot=$4 WHERE id=$1", id, batch, encoded, snapshot)
	if e != nil {
		return s, e
	}
	s, e = loadStatement(ctx, tx, id, false)
	if e != nil {
		return s, e
	}
	if e = statementAudit(ctx, tx, s, "draft_"+origin, reason, map[string]int{"rows": len(review.Rows)}); e != nil {
		return s, e
	}
	return s, nil
}
