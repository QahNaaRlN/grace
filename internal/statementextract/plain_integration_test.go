//go:build integration

package statementextract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"os"
	"strings"
	"testing"
)

type s58Reader struct{ s56Reader }

func (r *s58Reader) PlainPolicy(ctx context.Context, n int) (statementextract.DocumentPolicy, error) {
	p, e := r.CurrentDocumentPolicy(ctx, n)
	p.Profile = "plain_document_reading"
	// Match a full operator snapshot, including nested object ordering. A one-key
	// fixture cannot expose JSONB reordering that broke the real product consent.
	p.Snapshot = json.RawMessage(`{"model":"fixture/plain","provider":"fixture/eu","providerName":"Fixture","inputMicrosPerMillion":100000,"outputMicrosPerMillion":100000,"imageTokenUpperBound":1000,"vision":true,"selection":{"profile":"plain_document_reading","purpose":"statement","groupRevision":"s58-v1","maxImages":5}}`)
	return p, e
}
func (r *s58Reader) PreparePlain(_ context.Context, _ statementextract.DocumentPolicy, d documentdata.Document) error {
	return d.CheckReadable()
}
func (r *s58Reader) CompletePlain(_ context.Context, _ statementextract.DocumentPolicy, d documentdata.Document) (statementdata.PlainReading, *statementextract.Billing, error) {
	r.calls++
	raw := "| Дата операции | Дата списания | Сумма операции | Сумма карты | Описание | Карта |\n| --- | --- | --- | --- | --- | --- |\n"
	for i := range d.Pages {
		raw += fmt.Sprintf("| 19.10.2026 12:%02d | 19.10.2026 13:%02d | -12,34 RUB | -12,34 RUB | Public fixture page %d attempt %d | 1234 |\n", i, i, i+1, r.calls)
	}
	raw += "Итог пополнений: 0,00 RUB\nИтог списаний: 61,70 RUB"
	return statementdata.ParsePlain(raw), &statementextract.Billing{GenerationID: "plain-fixture", CostMicros: 100, InputTokens: 500, OutputTokens: 500}, nil
}
func fivePagePDF() []byte {
	objects := []string{`<< /Type /Catalog /Pages 2 0 R >>`, `<< /Type /Pages /Count 5 /Kids [3 0 R 4 0 R 5 0 R 6 0 R 7 0 R] >>`}
	for i := 0; i < 5; i++ {
		objects = append(objects, fmt.Sprintf(`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 600 800] /Resources << /Font << /F1 13 0 R >> >> /Contents %d 0 R >>`, 8+i))
	}
	for i := 0; i < 5; i++ {
		text := fmt.Sprintf("BT /F1 24 Tf 40 700 Td (Public document page %d -12.34 RUB) Tj ET", i+1)
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(text), text))
	}
	objects = append(objects, `<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>`)
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, n := range offsets[1:] {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", n)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return pdf.Bytes()
}
func TestS58FivePageProductAttempt(t *testing.T) {
	_, pool := s56Pool(t)
	ctx, account, _ := s56Owner(t, pool)
	if _, e := pool.Exec(ctx, "UPDATE ai_experiment_budget SET allowance_micros=1000000"); e != nil {
		t.Fatal(e)
	}
	helper, e := pdfextract.NewClient(os.Getenv("TEST_PDF_EXTRACTOR_SOCKET"))
	if e != nil {
		t.Fatal(e)
	}
	repo := imports.NewRepository(pool)
	repo.SetPDFExtractor(helper)
	source, _, e := repo.UploadStatement(ctx, fivePagePDF(), account, "five-public-pages.pdf", "s58-upload", "plain_review")
	if e != nil {
		t.Fatal(e)
	}
	if source.ReadingMode != "plain_review" || source.PageCount != 5 || len(source.Pages) != 5 || source.Rendered.CheckReadable() != nil {
		t.Fatal("five page input", source.ReadingMode, source.PageCount, len(source.Pages), source.Rendered.CheckReadable())
	}
	reader := &s58Reader{}
	service := statementextract.NewService(pool, repo, nil)
	service.SetDocumentReasoner(reader)
	preview, e := service.Preview(ctx, source.ID, statementextract.PreviewInput{ExpectedRevision: source.Revision})
	if e != nil {
		t.Fatal(e)
	}
	if preview.Plan.PromptVersion != statementextract.PlainPromptVersion || len(preview.Plan.Document.Pages) != 5 || preview.Plan.Document.TextSHA256 != documentdata.Hash(nil) {
		t.Fatal("wrong plain plan")
	}
	// Key-order normalization must never authorize changed policy contents.
	if _, e = pool.Exec(ctx, `UPDATE statement_extraction_previews SET plan=jsonb_set(plan,'{policy,document,snapshot,inputMicrosPerMillion}','100001'::jsonb) WHERE id=$1`, preview.ID); e != nil {
		t.Fatal(e)
	}
	_, e = service.Consent(ctx, source.ID, "s58-tampered-consent", statementextract.ConsentInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash})
	var changed *statementextract.Fault
	if !errors.As(e, &changed) || changed.Code != "preview_changed" || reader.calls != 0 {
		t.Fatal("changed price authorized", e)
	}
	if _, e = pool.Exec(ctx, `UPDATE statement_extraction_previews SET plan=jsonb_set(plan,'{policy,document,snapshot,inputMicrosPerMillion}','100000'::jsonb) WHERE id=$1`, preview.ID); e != nil {
		t.Fatal(e)
	}
	consent, e := service.Consent(ctx, source.ID, "s58-consent", statementextract.ConsentInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash})
	if e != nil {
		t.Fatal(e)
	}
	input := statementextract.ExtractionInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash, ConsentID: consent.ID}
	attempt, e := service.Extract(ctx, source.ID, "s58-extract", input)
	if e != nil || attempt.Status != "completed" {
		t.Fatal(attempt, e)
	}
	replay, e := service.Extract(ctx, source.ID, "s58-extract", input)
	if e != nil || replay.ID != attempt.ID || reader.calls != 1 {
		t.Fatal("repeat called", e)
	}
	source, e = repo.Statement(ctx, source.ID)
	if e != nil || source.Plain == nil || len(source.Plain.Candidate.Rows) != 5 || source.Plain.Candidate.RawAnswer == "" || source.Plain.ReviewedDigest != "" {
		t.Fatal("draft missing", e)
	}
	for i, row := range source.Review.Rows {
		if !strings.Contains(row.OperationDate.Local, fmt.Sprintf("12:%02d", i)) || len(row.Evidence) != 0 {
			t.Fatal("time lost or fake evidence")
		}
	}

	// Manual edits cannot forge the raw reply; a fresh model attempt owns a new original.
	firstRaw := source.Plain.Candidate.RawAnswer
	candidate := source.Plain.Candidate
	candidate.RawAnswer = "forged manual reply"
	source, e = repo.ManualStatementDraft(ctx, source.ID, "s58-manual", imports.StatementManualDraft{ExpectedRevision: source.Revision, SourceSHA256: source.SHA256, Plain: &candidate, Reason: "Verify immutable raw reply"})
	if e != nil {
		t.Fatal(e)
	}
	if source.Plain.Candidate.RawAnswer != firstRaw || source.Plain.Original.RawAnswer != firstRaw {
		t.Fatal("manual draft forged raw reply")
	}
	next, e := service.Preview(ctx, source.ID, statementextract.PreviewInput{ExpectedRevision: source.Revision})
	if e != nil {
		t.Fatal(e)
	}
	nextConsent, e := service.Consent(ctx, source.ID, "s58-consent-2", statementextract.ConsentInput{ExpectedRevision: source.Revision, PreviewID: next.ID, PreviewHash: next.Hash})
	if e != nil {
		t.Fatal(e)
	}
	nextAttempt, e := service.Extract(ctx, source.ID, "s58-extract-2", statementextract.ExtractionInput{ExpectedRevision: source.Revision, PreviewID: next.ID, PreviewHash: next.Hash, ConsentID: nextConsent.ID})
	if e != nil || nextAttempt.Status != "completed" {
		t.Fatal("second explicit attempt failed", e)
	}
	source, e = repo.Statement(ctx, source.ID)
	if e != nil {
		t.Fatal(e)
	}
	if reader.calls != 2 || source.Plain.Original.RawAnswer != source.Plain.Candidate.RawAnswer || source.Plain.Original.RawAnswer == firstRaw {
		t.Fatal("new draft displays stale first-attempt reply")
	}
}
