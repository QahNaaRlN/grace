package commitments

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/QahNaaRlN/grace/internal/loans"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Loan struct {
	Lender                string          `json:"lender"`
	AgreementNumber       string          `json:"agreementNumber"`
	AgreementOn           *string         `json:"agreementOn"`
	PrincipalMinor        *int64          `json:"principalMinor"`
	AnnualRateBasisPoints *int            `json:"annualRateBasisPoints"`
	RepaymentType         string          `json:"repaymentType"`
	Reminders             bool            `json:"reminders"`
	Revision              int             `json:"revision"`
	Documents             []Document      `json:"documents"`
	Drafts                []ScheduleDraft `json:"drafts"`
	// Debt is computed for GET (SPEC-0053) and ignored on save.
	Debt *loans.DebtStatus `json:"debt,omitempty"`
}
type Document struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Version   int       `json:"version"`
	SHA256    string    `json:"sha256"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Handler) GetLoan(w http.ResponseWriter, r *http.Request) {
	on := loans.Today()
	if value := r.URL.Query().Get("on"); value != "" {
		parsed, err := loans.ParseDay(value)
		if err != nil || loans.FormatDay(parsed) != value {
			web.WriteError(w, 400, "invalid_request", "Дата должна быть в формате ГГГГ-ММ-ДД.")
			return
		}
		on = parsed
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	id := chi.URLParam(r, "obligationId")
	if err = lockLoan(r.Context(), tx, id); err != nil {
		failure(w, err)
		return
	}
	loan := Loan{Documents: []Document{}, Drafts: []ScheduleDraft{}, RepaymentType: loans.TypeUnknown}
	err = tx.QueryRow(r.Context(), `SELECT lender,agreement_number,agreement_on::text,principal_minor,annual_rate_basis_points,repayment_type,reminders,revision FROM loan_details WHERE obligation_id::text=$1`, id).Scan(&loan.Lender, &loan.AgreementNumber, &loan.AgreementOn, &loan.PrincipalMinor, &loan.AnnualRateBasisPoints, &loan.RepaymentType, &loan.Reminders, &loan.Revision)
	if err != nil && err != pgx.ErrNoRows {
		failure(w, err)
		return
	}
	rows, err := tx.Query(r.Context(), `SELECT id::text,filename,version,sha256,octet_length(content),created_at FROM loan_documents WHERE obligation_id::text=$1 ORDER BY version DESC`, id)
	if err != nil {
		failure(w, err)
		return
	}
	for rows.Next() {
		var d Document
		if err = rows.Scan(&d.ID, &d.Filename, &d.Version, &d.SHA256, &d.Size, &d.CreatedAt); err != nil {
			rows.Close()
			failure(w, err)
			return
		}
		loan.Documents = append(loan.Documents, d)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		failure(w, err)
		return
	}
	rows, err = tx.Query(r.Context(), `SELECT id::text,payments,revision,applied_at FROM payment_schedule_drafts WHERE obligation_id::text=$1 ORDER BY created_at DESC LIMIT 50`, id)
	if err != nil {
		failure(w, err)
		return
	}
	for rows.Next() {
		var d ScheduleDraft
		if err = rows.Scan(&d.ID, &d.Payments, &d.Revision, &d.AppliedAt); err != nil {
			rows.Close()
			failure(w, err)
			return
		}
		loan.Drafts = append(loan.Drafts, d)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		failure(w, err)
		return
	}
	debt, err := loans.Debt(r.Context(), tx, id, on)
	if err != nil {
		failure(w, err)
		return
	}
	loan.Debt = &debt
	web.WriteJSON(w, 200, loan)
}
func (h *Handler) SaveLoan(w http.ResponseWriter, r *http.Request) {
	var loan Loan
	if !web.DecodeJSON(w, r, &loan) {
		return
	}
	loan.Documents = []Document{}
	loan.Drafts = []ScheduleDraft{}
	loan.Debt = nil
	if loan.RepaymentType == "" {
		loan.RepaymentType = loans.TypeUnknown
	}
	loan.Lender = strings.TrimSpace(loan.Lender)
	loan.AgreementNumber = strings.TrimSpace(loan.AgreementNumber)
	if !validText(loan.Lender, 160) || !validText(loan.AgreementNumber, 160) || (loan.AgreementOn != nil && !validDate(*loan.AgreementOn)) || (loan.PrincipalMinor != nil && (*loan.PrincipalMinor < 1 || *loan.PrincipalMinor > 9007199254740991)) || (loan.AnnualRateBasisPoints != nil && (*loan.AnnualRateBasisPoints < 0 || *loan.AnnualRateBasisPoints > 100000)) ||
		(loan.RepaymentType != loans.Annuity && loan.RepaymentType != loans.Differentiated && loan.RepaymentType != loans.TypeUnknown) {
		invalid(w)
		return
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	id := chi.URLParam(r, "obligationId")
	if err = lockLoan(r.Context(), tx, id); err != nil {
		failure(w, err)
		return
	}
	err = tx.QueryRow(r.Context(), `INSERT INTO loan_details(obligation_id,lender,agreement_number,agreement_on,principal_minor,annual_rate_basis_points,reminders,repayment_type) SELECT $1,$2,$3,$4,$5,$6,$7,$9 WHERE $8=0 ON CONFLICT(obligation_id) DO NOTHING RETURNING revision`, id, loan.Lender, loan.AgreementNumber, loan.AgreementOn, loan.PrincipalMinor, loan.AnnualRateBasisPoints, loan.Reminders, loan.Revision, loan.RepaymentType).Scan(&loan.Revision)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(r.Context(), `UPDATE loan_details SET lender=$2,agreement_number=$3,agreement_on=$4,principal_minor=$5,annual_rate_basis_points=$6,reminders=$7,repayment_type=$9,revision=revision+1 WHERE obligation_id::text=$1 AND revision=$8 RETURNING revision`, id, loan.Lender, loan.AgreementNumber, loan.AgreementOn, loan.PrincipalMinor, loan.AnnualRateBasisPoints, loan.Reminders, loan.Revision, loan.RepaymentType).Scan(&loan.Revision)
	}
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 200, loan)
}
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 17<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		invalid(w)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		invalid(w)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (16<<20)+1))
	if err != nil || len(data) > 16<<20 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		invalid(w)
		return
	}
	name := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	if name == "." || !validText(name, 180) || strings.TrimSpace(name) == "" {
		invalid(w)
		return
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	id := chi.URLParam(r, "obligationId")
	if err = lockLoan(r.Context(), tx, id); err != nil {
		failure(w, err)
		return
	}
	// Serialize the quota across all loans of the owner, not only this agreement.
	if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended('loan-documents:'||grace_owner_id()::text,0))`); err != nil {
		failure(w, err)
		return
	}
	var used int64
	if err = tx.QueryRow(r.Context(), `SELECT COALESCE(sum(octet_length(content)),0) FROM loan_documents`).Scan(&used); err != nil {
		failure(w, err)
		return
	}
	if used+int64(len(data)) > 100<<20 {
		web.WriteError(w, 422, "document_quota", "Лимит документов — 100 МиБ. Новый файл не сохранён.")
		return
	}
	digest := sha256.Sum256(data)
	d := Document{Filename: name, SHA256: hex.EncodeToString(digest[:]), Size: len(data)}
	err = tx.QueryRow(r.Context(), `INSERT INTO loan_documents(obligation_id,filename,version,sha256,content) SELECT $1,$2,COALESCE(max(version),0)+1,$3,$4 FROM loan_documents WHERE obligation_id=$1::uuid RETURNING id::text,version,created_at`, id, name, d.SHA256, data).Scan(&d.ID, &d.Version, &d.CreatedAt)
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 201, d)
}
func (h *Handler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	var name string
	var data []byte
	err := h.pool.QueryRow(r.Context(), `SELECT filename,content FROM loan_documents WHERE id::text=$1 AND obligation_id::text=$2`, chi.URLParam(r, "documentId"), chi.URLParam(r, "obligationId")).Scan(&name, &data)
	if err != nil {
		web.WriteError(w, 404, "not_found", "Документ недоступен.")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(data)
}
