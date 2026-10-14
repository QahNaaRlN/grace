package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/importreview"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *database.TenantPool
	now  func() time.Time
}

func NewRepository(p *pgxpool.Pool) *Repository { return &Repository{database.ForUsers(p), time.Now} }
func (r *Repository) begin(ctx context.Context, write bool) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	if write {
		if err = financialevents.LockOwner(ctx, tx); err != nil {
			_ = tx.Rollback(ctx)
			return nil, err
		}
	}
	return tx, nil
}
func request(ctx context.Context, tx pgx.Tx, scope, key string, payload any, out any) (bool, error) {
	if len(key) == 0 || len(key) > 160 {
		return false, fault("idempotency_key_required", 400)
	}
	var h string
	var b []byte
	err := tx.QueryRow(ctx, "SELECT request_hash,response FROM import_requests WHERE scope=$1 AND idempotency_key=$2", scope, key).Scan(&h, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if h != digest(payload) {
		return false, fault("idempotency_conflict", 409)
	}
	return true, json.Unmarshal(b, out)
}
func remember(ctx context.Context, tx pgx.Tx, scope, key string, payload, out any) error {
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO import_requests(scope,idempotency_key,request_hash,response) VALUES($1,$2,$3,$4)", scope, key, digest(payload), b)
	return err
}

func (r *Repository) Upload(ctx context.Context, raw []byte, key string) (Batch, error) {
	return r.UploadNamed(ctx, raw, key, "")
}

// UploadNamed keeps the uploaded file name for the history (SPEC-0044).
func (r *Repository) UploadNamed(ctx context.Context, raw []byte, key, fileName string) (Batch, error) {
	s, err := importreview.Load(bytes.NewReader(raw))
	if err != nil {
		return Batch{}, fault("invalid_snapshot", 400)
	}
	report, err := importreview.Review(s)
	if err != nil {
		return Batch{}, fault("review_limit", 422)
	}
	return r.uploadReport(ctx, raw, key, report, upload{"json", fileName})
}

type upload struct{ format, fileName string }

func (r *Repository) uploadReport(ctx context.Context, raw []byte, key string, report importreview.Report, file upload) (Batch, error) {
	tx, err := r.begin(ctx, true)
	if err != nil {
		return Batch{}, err
	}
	defer tx.Rollback(ctx)
	var b Batch
	payload := struct{ Hash, Source string }{hashBytes(raw), report.Source.SourceID}
	replay, err := request(ctx, tx, "upload", key, payload, &b)
	if err != nil || replay {
		return b, err
	}
	var id string
	created := false
	err = tx.QueryRow(ctx, "SELECT id::text FROM import_batches WHERE payload_hash=$1", payload.Hash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		created = true
		data, _ := json.Marshal(report)
		err = tx.QueryRow(ctx, "INSERT INTO import_batches(source_id,payload_hash,snapshot,report,format,file_name) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')) RETURNING id::text", report.Source.SourceID, payload.Hash, raw, data, file.format, cleanFileName(file.fileName)).Scan(&id)
		if err != nil {
			return b, err
		}
		involved := map[string]int{}
		for _, c := range report.Candidates {
			involved[c.RecordID]++
			if c.OtherRecordID != c.RecordID {
				involved[c.OtherRecordID]++
			}
		}
		for _, rec := range report.Records {
			source, _ := json.Marshal(rec)
			_, err = tx.Exec(ctx, "INSERT INTO import_records(batch_id,snapshot_record_id,domain,source,raw_hash,questions) VALUES($1,$2,$3,$4,$5,$6)", id, rec.ID, rec.Domain, source, digest(rec.Raw), len(rec.Issues)+involved[rec.ID])
			if err != nil {
				return b, err
			}
		}
	} else if err != nil {
		return b, err
	}
	b, err = loadBatchHeader(ctx, tx, id, false)
	if err != nil {
		return b, err
	}
	if b.Source.SourceID != report.Source.SourceID {
		return Batch{}, fault("source_conflict", 409)
	}
	if err = remember(ctx, tx, "upload", key, payload, b); err != nil {
		return b, err
	}
	b.created = created
	return b, tx.Commit(ctx)
}
func loadBatch(ctx context.Context, tx pgx.Tx, id string, lock bool) (Batch, importreview.Report, error) {
	var report importreview.Report
	b, err := loadBatchHeader(ctx, tx, id, lock)
	if err != nil {
		return b, report, err
	}
	var raw []byte
	if err = tx.QueryRow(ctx, "SELECT report FROM import_batches WHERE id=$1", id).Scan(&raw); err != nil {
		return b, report, err
	}
	return b, report, json.Unmarshal(raw, &report)
}

// loadBatchHeader reads a batch without parsing its report, which holds every
// source row; history and record pages stay fast on thousands of rows.
func loadBatchHeader(ctx context.Context, tx pgx.Tx, id string, lock bool) (Batch, error) {
	var b Batch
	var source, domains []byte
	query := "SELECT id::text,payload_hash,revision,status,created_at,report->'source',COALESCE(report->'domainCounts','{}'),file_name,format FROM import_batches WHERE id=$1"
	if lock {
		query += " FOR UPDATE"
	}
	err := tx.QueryRow(ctx, query, id).Scan(&b.ID, &b.PayloadHash, &b.Revision, &b.Status, &b.CreatedAt, &source, &domains, &b.FileName, &b.Format)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, ErrNotFound
	}
	if err != nil {
		return b, err
	}
	if err = json.Unmarshal(source, &b.Source); err != nil {
		return b, err
	}
	if err = json.Unmarshal(domains, &b.DomainCounts); err != nil {
		return b, err
	}
	b.Counts = map[string]int{}
	rows, err := tx.Query(ctx, "SELECT CASE WHEN published THEN 'published' ELSE decision->>'action' END,count(*) FROM import_records WHERE batch_id=$1 AND domain='transaction' GROUP BY 1", id)
	if err != nil {
		return b, err
	}
	for rows.Next() {
		var k string
		var n int
		if err = rows.Scan(&k, &n); err != nil {
			rows.Close()
			return b, err
		}
		b.Counts[k] = n
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return b, err
	}
	return b, summarize(ctx, tx, &b)
}
func (r *Repository) Batch(ctx context.Context, id string) (Batch, error) {
	tx, err := r.begin(ctx, false)
	if err != nil {
		return Batch{}, err
	}
	defer tx.Rollback(ctx)
	return loadBatchHeader(ctx, tx, id, false)
}
func (r *Repository) Batches(ctx context.Context, cursor string, limit int) (Page[Batch], error) {
	page := Page[Batch]{Items: []Batch{}}
	tx, err := r.begin(ctx, false)
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	offset, ok := parseOffset(cursor)
	if !ok {
		return page, fault("invalid_pagination", 400)
	}
	// Newest first: the history answers «what was loaded when».
	rows, err := tx.Query(ctx, "SELECT id::text FROM import_batches ORDER BY created_at DESC, id DESC OFFSET $1 LIMIT $2", offset, limit+1)
	if err != nil {
		return page, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return page, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return page, err
	}
	if len(ids) > limit {
		page.NextCursor = strconv.Itoa(offset + limit)
		ids = ids[:limit]
	}
	for _, id := range ids {
		b, err := loadBatchHeader(ctx, tx, id, false)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, b)
	}
	return page, nil
}
func readRecord(row pgx.Row) (Record, error) {
	rec := Record{Candidates: []Candidate{}, History: []Audit{}}
	var source, decision []byte
	err := row.Scan(&rec.ID, &source, &decision, &rec.Published, &rec.EventID, &rec.Questions)
	if errors.Is(err, pgx.ErrNoRows) {
		return rec, ErrNotFound
	}
	if err != nil {
		return rec, err
	}
	if err = json.Unmarshal(source, &rec.Source); err != nil {
		return rec, err
	}
	return rec, json.Unmarshal(decision, &rec.Decision)
}

const recordSQL = "SELECT r.id::text,r.source,r.decision,r.published,p.event_id::text,r.questions FROM import_records r LEFT JOIN import_provenance p ON p.record_id=r.id AND p.owner_id=r.owner_id "

func loadRecord(ctx context.Context, tx pgx.Tx, batch, id string) (Record, error) {
	return readRecord(tx.QueryRow(ctx, recordSQL+"WHERE r.batch_id=$1 AND r.id=$2", batch, id))
}
func (r *Repository) Records(ctx context.Context, batch, cursor, domain, status string, limit int) (Page[Record], error) {
	return r.RecordsQuery(ctx, batch, RecordQuery{Cursor: cursor, Limit: limit, RecordFilter: RecordFilter{Domain: domain, Status: status}})
}
func (r *Repository) Record(ctx context.Context, batch, id string) (Record, error) {
	tx, err := r.begin(ctx, false)
	if err != nil {
		return Record{}, err
	}
	defer tx.Rollback(ctx)
	_, report, err := loadBatch(ctx, tx, batch, false)
	if err != nil {
		return Record{}, err
	}
	rec, err := loadRecord(ctx, tx, batch, id)
	if err != nil {
		return rec, err
	}
	rec.Candidates, err = candidates(ctx, tx, batch, report, rec)
	if err != nil {
		return rec, err
	}
	rows, err := tx.Query(ctx, "SELECT revision,actor_id::text,created_at,decision FROM import_decisions WHERE record_id=$1 ORDER BY revision", id)
	if err != nil {
		return rec, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Audit
		var raw []byte
		if err = rows.Scan(&a.Revision, &a.ActorID, &a.CreatedAt, &raw); err != nil {
			return rec, err
		}
		if err = json.Unmarshal(raw, &a.Decision); err != nil {
			return rec, err
		}
		rec.History = append(rec.History, a)
	}
	return rec, rows.Err()
}
func (r *Repository) Edit(ctx context.Context, batch, id, key string, input Edit) (Batch, error) {
	if err := validateDecision(input.Decision); err != nil {
		return Batch{}, err
	}
	tx, err := r.begin(ctx, true)
	if err != nil {
		return Batch{}, err
	}
	defer tx.Rollback(ctx)
	scope := "edit:" + batch + ":" + id
	var b Batch
	replay, err := request(ctx, tx, scope, key, input, &b)
	if err != nil || replay {
		return b, err
	}
	b, _, err = loadBatch(ctx, tx, batch, true)
	if err != nil {
		return b, err
	}
	if b.Revision != input.ExpectedRevision {
		return b, fault("revision_conflict", 409)
	}
	rec, err := loadRecord(ctx, tx, batch, id)
	if err != nil {
		return b, err
	}
	if rec.Published {
		return b, fault("already_published", 409)
	}
	if rec.Source.Domain != "transaction" && input.Decision.Action != "unresolved" {
		return b, fault("unsupported_domain", 422)
	}
	if input.Decision.Action == "duplicate_of" {
		canonical, err := loadRecord(ctx, tx, batch, input.Decision.CanonicalRecordID)
		if err != nil {
			return b, err
		}
		if canonical.ID == id || canonical.Source.Domain != "transaction" || canonical.Decision.Action == "duplicate_of" || canonical.Decision.Action == "exclude" {
			return b, fault("invalid_duplicate", 422)
		}
		var incoming bool
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM import_records WHERE batch_id=$1 AND decision->>'canonicalRecordId'=$2)", batch, id).Scan(&incoming); err != nil {
			return b, err
		}
		if incoming {
			return b, fault("duplicate_chain", 422)
		}
	}
	raw, _ := json.Marshal(input.Decision)
	if _, err = tx.Exec(ctx, "UPDATE import_records SET decision=$1 WHERE id=$2", raw, id); err != nil {
		return b, err
	}
	if _, err = tx.Exec(ctx, "UPDATE import_batches SET revision=revision+1 WHERE id=$1", batch); err != nil {
		return b, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO import_decisions(record_id,revision,decision) VALUES($1,$2,$3)", id, b.Revision+1, raw); err != nil {
		return b, err
	}
	b, _, err = loadBatch(ctx, tx, batch, false)
	if err != nil {
		return b, err
	}
	if err = remember(ctx, tx, scope, key, input, b); err != nil {
		return b, err
	}
	return b, tx.Commit(ctx)
}

func candidates(ctx context.Context, tx pgx.Tx, batch string, report importreview.Report, rec Record) ([]Candidate, error) {
	result := []Candidate{}
	seen := map[string]bool{}
	for _, c := range report.Candidates {
		other := ""
		if c.RecordID == rec.Source.ID {
			other = c.OtherRecordID
		}
		if c.OtherRecordID == rec.Source.ID {
			other = c.RecordID
		}
		if other == "" {
			continue
		}
		var id string
		if err := tx.QueryRow(ctx, "SELECT id::text FROM import_records WHERE batch_id=$1 AND snapshot_record_id=$2", batch, other).Scan(&id); err != nil {
			return nil, err
		}
		cid := "record:" + id
		if !seen[cid] {
			result = append(result, Candidate{ID: cid, RecordID: id, Rule: c.Rule, Differences: c.Differences})
			seen[cid] = true
		}
	}
	n := rec.Source.Normalized
	amount := n.AmountMinor
	kind := n.Kind
	date := n.Date
	if rec.Decision.Fields != nil {
		f := rec.Decision.Fields
		amount = &f.AmountMinor
		kind = f.Kind
		if f.OccurredDate != "" {
			date = &importreview.Date{Local: f.OccurredDate, Precision: "date"}
		} else if t, err := time.Parse(time.RFC3339, f.OccurredAt); err == nil {
			zone, _ := time.LoadLocation("Asia/Yekaterinburg")
			date = &importreview.Date{Local: t.In(zone).Format("2006-01-02"), Precision: "date"}
		}
	}
	if amount != nil && date != nil && len(date.Local) >= 10 {
		// A month migration also reviews matches within the main source block.
		batchRows, err := tx.Query(ctx, `SELECT id::text,source->'normalized'->'date'->>'local',decision
		FROM import_records WHERE batch_id=$1 AND id<>$2 AND domain='transaction'
		AND COALESCE(decision->'fields'->>'kind',source->'normalized'->>'kind')=$3
		AND COALESCE(decision->'fields'->>'amountMinor',source->'normalized'->>'amountMinor')=$4`, batch, rec.ID, kind, strconv.FormatInt(*amount, 10))
		if err != nil {
			return nil, err
		}
		for batchRows.Next() {
			var id string
			var local *string
			var decisionRaw []byte
			if err = batchRows.Scan(&id, &local, &decisionRaw); err != nil {
				batchRows.Close()
				return nil, err
			}
			var decision Decision
			if err = json.Unmarshal(decisionRaw, &decision); err != nil {
				batchRows.Close()
				return nil, err
			}
			otherDate := ""
			if local != nil && len(*local) >= 10 {
				otherDate = (*local)[:10]
			}
			if decision.Fields != nil {
				otherDate = targetLocalDate(decision.Fields.OccurredAt)
				if decision.Fields.OccurredDate != "" {
					otherDate = decision.Fields.OccurredDate
				}
			}
			cid := "record:" + id
			if otherDate == date.Local[:10] && !seen[cid] {
				result = append(result, Candidate{ID: cid, RecordID: id, Rule: "batch_date_amount_kind", Differences: []string{"Compare source and reviewed fields; equal totals do not prove a duplicate"}})
				seen[cid] = true
			}
		}
		err = batchRows.Err()
		batchRows.Close()
		if err != nil {
			return nil, err
		}
		rows, err := tx.Query(ctx, `SELECT DISTINCT e.id::text FROM financial_events e JOIN postings p ON p.financial_event_id=e.id AND p.owner_id=e.owner_id WHERE p.amount_minor=$1 AND e.kind::text=$2 AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)=$3::date ORDER BY e.id::text LIMIT 1001`, *amount, kind, date.Local[:10])
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			cid := "event:" + id
			result = append(result, Candidate{ID: cid, EventID: id, Rule: "date_amount_kind", Differences: []string{"Compare account, recipient and source evidence"}})
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	// Exact raw evidence already published in another snapshot is never a new fact silently.
	rows, err := tx.Query(ctx, `SELECT DISTINCT p.event_id::text FROM import_records r JOIN import_provenance p ON p.record_id=r.id AND p.owner_id=r.owner_id WHERE r.batch_id<>$1 AND r.raw_hash=$2 AND p.event_id IS NOT NULL`, batch, digest(rec.Source.Raw))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		cid := "event:" + id
		exists := false
		for _, c := range result {
			if c.ID == cid {
				exists = true
			}
		}
		if !exists {
			result = append(result, Candidate{ID: cid, EventID: id, Rule: "published_source", Differences: []string{}})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(result) > 1000 {
		return nil, fault("review_limit", 422)
	}
	for i := range result {
		if result[i].RecordID != "" {
			other, err := loadRecord(ctx, tx, batch, result[i].RecordID)
			if err != nil {
				return nil, err
			}
			result[i].Source = &other.Source
		} else if result[i].EventID != "" {
			event, err := financialevents.LoadInTx(ctx, tx, result[i].EventID)
			if err != nil {
				return nil, err
			}
			result[i].Event = &event
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
