package receiptdrafts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/receipts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *database.TenantPool }

func NewRepository(p *pgxpool.Pool) *Repository { return &Repository{database.ForUsers(p)} }
func Hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

const draftSelect = `SELECT id::text,revision,status,fields,extractions,photo_status,expires_at,receipt_id::text,event_id::text,image_id,image_hash,extraction_key,source_kind,source_pages,reading_consent FROM receipt_drafts WHERE id::text=$1`

func load(ctx context.Context, db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string, lock bool) (Draft, error) {
	var d Draft
	var f, e, pages, consent []byte
	q := draftSelect
	if lock {
		q += " FOR UPDATE"
	}
	err := db.QueryRow(ctx, q, id).Scan(&d.ID, &d.Revision, &d.Status, &f, &e, &d.PhotoStatus, &d.ExpiresAt, &d.ReceiptID, &d.EventID, &d.ImageID, &d.ImageHash, &d.ExtractionKey, &d.SourceKind, &pages, &consent)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, errors.New("not_found")
	}
	if err != nil {
		return d, err
	}
	if json.Unmarshal(f, &d.Fields) != nil || json.Unmarshal(e, &d.Extractions) != nil {
		return d, errors.New("invalid_draft")
	}
	if json.Unmarshal(pages, &d.Pages) != nil {
		return d, errors.New("invalid_draft")
	}
	if len(consent) > 0 {
		if json.Unmarshal(consent, &d.ReadingConsent) != nil {
			return d, errors.New("invalid_draft")
		}
	}
	d.Warnings = []string{}
	if _, err := Validate(d.Fields); err != nil {
		d.Warnings = append(d.Warnings, err.Error())
	}
	return d, nil
}
func editable(d Draft) error {
	if time.Now().After(d.ExpiresAt) {
		return errors.New("image_expired")
	}
	if d.Status == "confirmed" || d.Status == "cancelled" || d.Status == "expired" || d.Status == "extracting" {
		return errors.New("draft_not_editable")
	}
	return nil
}
func (r *Repository) Get(ctx context.Context, id string) (Draft, error) {
	return load(ctx, r.pool, id, false)
}
func (r *Repository) Create(ctx context.Context, imageID, imageHash string) (Draft, bool, error) {
	return r.createDocument(ctx, imageID, imageHash, "image", []documentdata.Page{})
}
func (r *Repository) createDocument(ctx context.Context, imageID, imageHash, kind string, pages []documentdata.Page) (Draft, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Draft{}, false, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('receipt-upload:' || grace_owner_id()::text,0))"); err != nil {
		return Draft{}, false, err
	}
	var previousID string
	findErr := tx.QueryRow(ctx, "SELECT id::text FROM receipt_drafts WHERE image_hash=$1", imageHash).Scan(&previousID)
	if findErr == nil {
		d, err := load(ctx, tx, previousID, false)
		if err != nil {
			return d, false, err
		}
		return d, false, tx.Commit(ctx)
	}
	if !errors.Is(findErr, pgx.ErrNoRows) {
		return Draft{}, false, findErr
	}
	var active int
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM receipt_drafts WHERE status NOT IN ('confirmed','cancelled','expired') AND expires_at>now()").Scan(&active); err != nil {
		return Draft{}, false, err
	}
	if active >= 20 {
		return Draft{}, false, errors.New("draft_limit")
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO receipt_drafts(image_id,image_hash,source_kind,source_pages) VALUES($1,$2,$3,$4) ON CONFLICT(owner_id,image_hash) DO NOTHING RETURNING id::text`, imageID, imageHash, kind, marshal(pages)).Scan(&id)
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, "SELECT id::text FROM receipt_drafts WHERE image_hash=$1", imageHash).Scan(&id)
	}
	if err != nil {
		return Draft{}, false, err
	}
	d, err := load(ctx, tx, id, false)
	if err != nil {
		return d, false, err
	}
	return d, created, tx.Commit(ctx)
}
func (r *Repository) Edit(ctx context.Context, id string, in Edit) (Draft, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	d, err := load(ctx, tx, id, true)
	if err != nil {
		return d, err
	}
	if err = editable(d); err != nil {
		return d, err
	}
	if d.Revision != in.Revision {
		return d, errors.New("stale_revision")
	}
	if len(in.Fields.Items) > 50 || len(marshal(in.Fields)) > 64000 {
		return d, errors.New("invalid_request")
	}
	// Suggestions are provider evidence, never accepted from the client.
	for i := range in.Fields.Items {
		in.Fields.Items[i].GroupSuggestion = nil
	}
	_, err = tx.Exec(ctx, `UPDATE receipt_drafts SET fields=$2,revision=revision+1,status='reviewed' WHERE id::text=$1`, id, marshal(in.Fields))
	if err != nil {
		return d, err
	}
	d, err = load(ctx, tx, id, false)
	if err != nil {
		return d, err
	}
	return d, tx.Commit(ctx)
}
func marshal(v any) []byte { b, _ := json.Marshal(v); return b }
func (r *Repository) BeginExtraction(ctx context.Context, id, key string) (Draft, bool, error) {
	return r.beginDocumentExtraction(ctx, id, key, nil)
}
func (r *Repository) beginDocumentExtraction(ctx context.Context, id, key string, plan *ReadingPlan) (Draft, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Draft{}, false, err
	}
	defer tx.Rollback(ctx)
	d, err := load(ctx, tx, id, true)
	if err != nil {
		return d, false, err
	}
	if d.ExtractionKey == key && d.Status != "extracting" {
		return d, false, nil
	}
	if plan != nil && (plan.Revision != d.Revision || plan.SourceSHA256 != d.ImageHash) {
		return d, false, errors.New("stale_revision")
	}
	if len(d.Extractions) >= 10 {
		return d, false, errors.New("extraction_limit")
	}
	if err = editable(d); err != nil {
		return d, false, err
	}
	if d.PhotoStatus != "stored" {
		return d, false, errors.New("image_expired")
	}
	_, err = tx.Exec(ctx, `UPDATE receipt_drafts SET status='extracting',extraction_key=$2,extraction_started_at=now(),reading_consent=$3 WHERE id::text=$1`, id, key, optionalReadingPlan(plan))
	if err != nil {
		return d, false, err
	}
	return d, true, tx.Commit(ctx)
}
func (r *Repository) FinishExtraction(ctx context.Context, id, key string, out *Extracted) (Draft, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	d, err := load(ctx, tx, id, true)
	if err != nil {
		return d, err
	}
	if d.Status != "extracting" || d.ExtractionKey != key {
		return d, errors.New("stale_revision")
	}
	status := "uploaded"
	fields := d.Fields
	history := d.Extractions
	if out != nil {
		status = "extracted"
		fields = out.Fields
		history = append(history, *out)
	}
	if len(history) > 10 {
		return d, errors.New("extraction_limit")
	}
	_, err = tx.Exec(ctx, `UPDATE receipt_drafts SET status=$2,fields=$3,extractions=$4,revision=revision+1 WHERE id::text=$1`, id, status, marshal(fields), marshal(history))
	if err != nil {
		return d, err
	}
	d, err = load(ctx, tx, id, false)
	if err != nil {
		return d, err
	}
	return d, tx.Commit(ctx)
}

type previewPayload struct {
	Fields      Fields      `json:"fields"`
	Fingerprint string      `json:"fingerprint"`
	Candidates  []Candidate `json:"candidates"`
}

func candidateList(ctx context.Context, tx pgx.Tx, v Validated) ([]Candidate, error) {
	day := v.Event.OccurredDate
	if day == "" {
		zone, _ := time.LoadLocation("Asia/Yekaterinburg")
		day = v.Event.OccurredAt.In(zone).Format("2006-01-02")
	}
	events, err := financialevents.PurchaseCandidatesInTx(ctx, tx, v.Receipt.TotalMinor, "RUB", day)
	if err != nil {
		return nil, err
	}
	out := []Candidate{}
	for _, e := range events {
		has, err := receipts.HasPurchaseReceiptInTx(ctx, tx, e.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, Candidate{e.ID, e.Merchant + " · " + e.Description, strconv.FormatInt(v.Receipt.TotalMinor, 10), "RUB", has})
	}
	return out, nil
}
func validateDecision(ctx context.Context, tx pgx.Tx, v Validated, candidates []Candidate) error {
	if v.Fields.Action == "create" {
		if len(candidates) > 0 && strings.TrimSpace(v.Fields.CandidateReason) == "" {
			return errors.New("candidate_review_required")
		}
		postings, err := financialevents.ValidateInTx(ctx, tx, v.Event)
		if err != nil {
			return errors.New("invalid_account")
		}
		if len(postings) != 1 || postings[0].Currency != "RUB" {
			return errors.New("invalid_account")
		}
		return nil
	}
	found := false
	for _, c := range candidates {
		if c.EventID == v.Fields.ExistingEventID && !c.HasReceipt {
			found = true
		}
	}
	if !found {
		return errors.New("invalid_existing_purchase")
	}
	return nil
}
func (r *Repository) Preview(ctx context.Context, id string, revision int64) (Preview, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Preview{}, err
	}
	defer tx.Rollback(ctx)
	if err = financialevents.LockOwner(ctx, tx); err != nil {
		return Preview{}, err
	}
	d, err := load(ctx, tx, id, true)
	if err != nil {
		return Preview{}, err
	}
	if err = editable(d); err != nil {
		return Preview{}, err
	}
	if d.Revision != revision {
		return Preview{}, errors.New("stale_revision")
	}
	v, err := Validate(d.Fields)
	if err != nil {
		return Preview{}, err
	}
	candidates, err := candidateList(ctx, tx, v)
	if err != nil {
		return Preview{}, err
	}
	// A preview may show matches before the user decides. Confirmation still
	// refuses unresolved matches; the preview itself never writes the ledger.
	decisionErr := validateDecision(ctx, tx, v, candidates)
	if decisionErr != nil && decisionErr.Error() != "candidate_review_required" {
		return Preview{}, decisionErr
	}
	fingerprint, err := financialevents.LedgerFingerprintInTx(ctx, tx)
	if err != nil {
		return Preview{}, err
	}
	payload := marshal(previewPayload{d.Fields, fingerprint, candidates})
	p := Preview{Revision: revision, Action: d.Fields.Action, AmountMinor: strconv.FormatInt(v.Receipt.TotalMinor, 10), Currency: "RUB", AccountID: d.Fields.AccountID, ExistingEventID: d.Fields.ExistingEventID, Candidates: candidates, Warnings: []string{}, Payload: payload}
	if decisionErr != nil {
		p.Warnings = append(p.Warnings, decisionErr.Error())
	}
	p.Hash = Hash(json.RawMessage(payload))
	err = tx.QueryRow(ctx, `INSERT INTO receipt_draft_previews(draft_id,revision,hash,payload) VALUES($1::uuid,$2,$3,$4) RETURNING id::text,expires_at`, id, revision, p.Hash, payload).Scan(&p.ID, &p.ExpiresAt)
	if err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}

type Confirmation struct {
	PreviewID string `json:"previewId"`
	Hash      string `json:"hash"`
}

func (r *Repository) Confirm(ctx context.Context, id, key string, in Confirmation) (Draft, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	if err = financialevents.LockOwner(ctx, tx); err != nil {
		return Draft{}, err
	}
	d, err := load(ctx, tx, id, true)
	if err != nil {
		return d, err
	}
	requestHash := Hash(struct {
		ID    string
		Input Confirmation
	}{id, in})
	var oldHash, oldID string
	err = tx.QueryRow(ctx, `SELECT request_hash,draft_id::text FROM receipt_draft_confirmations WHERE request_key=$1`, key).Scan(&oldHash, &oldID)
	if err == nil {
		if oldHash != requestHash || oldID != id {
			return d, errors.New("idempotency_conflict")
		}
		return d, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return d, err
	}
	if err = editable(d); err != nil {
		return d, err
	}
	var p previewPayload
	var raw []byte
	var revision int64
	var hash string
	var expiry time.Time
	err = tx.QueryRow(ctx, `SELECT revision,hash,payload,expires_at FROM receipt_draft_previews WHERE id::text=$1 AND draft_id::text=$2`, in.PreviewID, id).Scan(&revision, &hash, &raw, &expiry)
	if err != nil || hash != in.Hash || revision != d.Revision || time.Now().After(expiry) || json.Unmarshal(raw, &p) != nil {
		return d, errors.New("stale_preview")
	}
	fingerprint, err := financialevents.LedgerFingerprintInTx(ctx, tx)
	if err != nil {
		return d, err
	}
	if fingerprint != p.Fingerprint || Hash(p.Fields) != Hash(d.Fields) {
		return d, errors.New("stale_preview")
	}
	v, err := Validate(p.Fields)
	if err != nil {
		return d, err
	}
	candidates, err := candidateList(ctx, tx, v)
	if err != nil {
		return d, err
	}
	if err = validateDecision(ctx, tx, v, candidates); err != nil {
		return d, err
	}
	eventID := v.Fields.ExistingEventID
	if v.Fields.Action == "create" {
		v.Event.IdempotencyKey = "receipt-draft:" + id
		v.Event.RequestHash = requestHash
		e, _, err := financialevents.CreateInTx(ctx, tx, v.Event, "manual")
		if err != nil {
			return d, err
		}
		eventID = e.ID
	}
	v.Receipt.PurchaseEventID = eventID
	v.Receipt.IdempotencyKey = "receipt-draft:" + id
	v.Receipt.RequestHash = requestHash
	receipt, _, err := receipts.CreateInTx(ctx, tx, v.Receipt)
	if err != nil {
		return d, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO receipt_draft_confirmations(request_key,request_hash,draft_id,preview_id,receipt_id,event_id) VALUES($1,$2,$3::uuid,$4::uuid,$5::uuid,$6::uuid)`, key, requestHash, id, in.PreviewID, receipt.ID, eventID)
	if err != nil {
		return d, err
	}
	_, err = tx.Exec(ctx, `UPDATE receipt_drafts SET status='confirmed',photo_status='pending_delete',receipt_id=$2::uuid,event_id=$3::uuid WHERE id::text=$1`, id, receipt.ID, eventID)
	if err != nil {
		return d, err
	}
	d, err = load(ctx, tx, id, false)
	if err != nil {
		return d, err
	}
	return d, tx.Commit(ctx)
}
func (r *Repository) Cancel(ctx context.Context, id string) (Draft, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	d, err := load(ctx, tx, id, true)
	if err != nil {
		return d, err
	}
	if d.Status == "confirmed" || d.Status == "extracting" {
		return d, errors.New("draft_not_editable")
	}
	_, err = tx.Exec(ctx, `UPDATE receipt_drafts SET status='cancelled',photo_status=CASE WHEN photo_status='deleted' THEN 'deleted' ELSE 'pending_delete' END WHERE id::text=$1`, id)
	if err != nil {
		return d, err
	}
	d, err = load(ctx, tx, id, false)
	if err != nil {
		return d, err
	}
	return d, tx.Commit(ctx)
}
func (r *Repository) DeletePhoto(ctx context.Context, d Draft, store *PhotoStore) error {
	if d.PhotoStatus != "pending_delete" {
		return nil
	}
	if err := store.Delete(d.ImageID); err != nil {
		return err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE receipt_drafts SET photo_status='deleted' WHERE id::text=$1 AND photo_status='pending_delete'`, d.ID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Sweep runs with a user's principal. It durably queues expiration before any
// filesystem deletion, so a storage failure or restart cannot lose the job.
func (r *Repository) Sweep(ctx context.Context, store *PhotoStore) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE receipt_drafts SET status='expired',photo_status='pending_delete' WHERE expires_at<=now() AND status NOT IN ('confirmed','cancelled','expired')`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE receipt_drafts SET status='uploaded' WHERE status='extracting' AND extraction_started_at<now()-interval '3 minutes'`)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,image_id FROM receipt_drafts WHERE photo_status='pending_delete' ORDER BY created_at LIMIT 100`)
	if err != nil {
		return err
	}
	var jobs []Draft
	for rows.Next() {
		d := Draft{PhotoStatus: "pending_delete"}
		if err = rows.Scan(&d.ID, &d.ImageID); err != nil {
			rows.Close()
			return err
		}
		jobs = append(jobs, d)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	for _, d := range jobs {
		if err = r.DeletePhoto(ctx, d, store); err != nil {
			return err
		}
	}
	return nil
}

func optionalReadingPlan(plan *ReadingPlan) any {
	if plan == nil {
		return nil
	}
	return marshal(plan)
}
