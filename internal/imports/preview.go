package imports

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/importreview"
	"github.com/jackc/pgx/v5"
)

func command(f *Fields, key string) (financialevents.Command, error) {
	if f == nil {
		return financialevents.Command{}, fault("fields_required", 422)
	}
	t, err := time.Parse(time.RFC3339, f.OccurredAt)
	if f.OccurredDate != "" && f.OccurredAt == "" && financialevents.ValidCalendarDate(f.OccurredDate) {
		t = time.Time{}
		err = nil
	}
	if err != nil || f.OccurredDate != "" && f.OccurredAt != "" || f.AmountMinor <= 0 || len(f.Currency) != 3 || f.Currency != strings.ToUpper(f.Currency) || len(f.Merchant) > 2000 || len(f.Description) > 4000 || len(f.TimeReason) > 2000 {
		return financialevents.Command{}, fault("invalid_fields", 422)
	}
	if f.Kind == "purchase" && strings.TrimSpace(f.Merchant) == "" && strings.TrimSpace(f.Description) == "" {
		return financialevents.Command{}, fault("purchase_description_required", 422)
	}
	for _, id := range []string{f.AccountID, f.DestinationAccountID, f.OriginalEventID} {
		if id != "" && !uuid.MatchString(id) {
			return financialevents.Command{}, fault("invalid_reference", 422)
		}
	}
	if f.CategoryID != nil && !uuid.MatchString(*f.CategoryID) {
		return financialevents.Command{}, fault("invalid_reference", 422)
	}
	if f.Kind == "refund" {
		if f.OriginalEventID == "" {
			return financialevents.Command{}, fault("original_purchase_required", 422)
		}
	} else if f.AccountID == "" {
		return financialevents.Command{}, fault("account_required", 422)
	}
	if f.Kind == "internal_transfer" && f.DestinationAccountID == "" {
		return financialevents.Command{}, fault("destination_required", 422)
	}
	return financialevents.Command{AllowUncategorized: true, Kind: f.Kind, IdempotencyKey: key, RequestHash: digest(f), AccountID: f.AccountID, DestinationAccountID: f.DestinationAccountID, CategoryID: f.CategoryID, OriginalEventID: f.OriginalEventID, AmountMinor: f.AmountMinor, Merchant: f.Merchant, Description: f.Description, OccurredAt: t, OccurredDate: f.OccurredDate}, nil
}
func add(a, b int64) (int64, bool) {
	if b > 0 && a > math.MaxInt64-b || b < 0 && a < math.MinInt64-b {
		return 0, false
	}
	return a + b, true
}
func selectionValid(ids []string) bool {
	if len(ids) == 0 || len(ids) > 100 {
		return false
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !uuid.MatchString(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
func planHash(p Preview) string {
	return digest(struct {
		Batch                        string
		Revision                     int64
		Lines                        []Line
		Changes                      []Change
		Blockers                     []string
		NewEvents, Links, Exclusions int
		StatementRevision            int64
	}{p.BatchID, p.Revision, p.Lines, p.Changes, p.Blockers, p.NewEvents, p.Links, p.Exclusions, p.StatementRevision})
}
func resolved(rec Record, c Candidate, records map[string]Record) bool {
	d := rec.Decision
	if d.Action == "exclude" {
		return true
	}
	if d.Action == "link_existing" && c.EventID == d.EventID {
		return true
	}
	if d.Action == "duplicate_of" {
		if c.RecordID == d.CanonicalRecordID {
			return true
		}
		if canon, ok := records[d.CanonicalRecordID]; ok && c.EventID != "" && (canon.Decision.EventID == c.EventID || canon.EventID != nil && *canon.EventID == c.EventID) {
			return true
		}
	}
	if other, ok := records[c.RecordID]; ok && other.Decision.Action == "duplicate_of" && other.Decision.CanonicalRecordID == rec.ID {
		return true
	}
	if other, ok := records[c.RecordID]; ok && other.Decision.Action == "exclude" && other.Decision.Reason != "" {
		return true
	}
	for _, resolution := range d.Resolutions {
		if resolution.CandidateID == c.ID && resolution.Reason != "" {
			return true
		}
	}
	return false
}
func (r *Repository) plan(ctx context.Context, tx pgx.Tx, batch Batch, report importreview.Report, ids []string) (Preview, error) {
	if batch.Format != nil && (*batch.Format == "bank_statement_pdf" || *batch.Format == "bank_statement_image") && len(ids) > 100 {
		return Preview{}, fault("statement_confirmation_limit", 422)
	}

	p := Preview{BatchID: batch.ID, Revision: batch.Revision, Lines: []Line{}, Changes: []Change{}, Blockers: []string{}}
	statementBlockers, statementWarnings, sourceRevision, err := r.statementPlanGuards(ctx, tx, batch.ID)
	if err != nil {
		return p, err
	}
	p.StatementRevision = sourceRevision
	p.Blockers = append(p.Blockers, statementBlockers...)
	records := map[string]Record{}
	selected := map[string]bool{}
	ids = append([]string(nil), ids...)
	sort.Strings(ids)
	for _, id := range ids {
		rec, err := loadRecord(ctx, tx, batch.ID, id)
		if err != nil {
			return p, err
		}
		records[id] = rec
		selected[id] = true
	}
	for _, rec := range records {
		if rec.Decision.Action == "duplicate_of" && !selected[rec.Decision.CanonicalRecordID] {
			canon, err := loadRecord(ctx, tx, batch.ID, rec.Decision.CanonicalRecordID)
			if err != nil {
				return p, err
			}
			records[canon.ID] = canon
		}
	}
	deltas := map[string]Change{}
	refundTotals := map[string]int64{}
	for _, id := range ids {
		rec := records[id]
		line := Line{RecordID: id, SourceSheet: rec.Source.Sheet, SourceRow: rec.Source.Row, Decision: rec.Decision, SourceDate: rec.Source.Normalized.Date, Blockers: []string{}, Warnings: []string{}, Postings: []Change{}, Candidates: []Candidate{}}
		if rec.Source.PDF != nil {
			ordinal := rec.Source.PDF.Ordinal
			line.PDFOrdinal = &ordinal
			line.SourceKind = rec.Source.SourceKind
		}
		line.Warnings = append(line.Warnings, statementWarnings...)
		if rec.Published {
			line.Blockers = append(line.Blockers, "already_published")
		}
		if rec.Source.Domain != "transaction" {
			line.Blockers = append(line.Blockers, "unsupported_domain")
		}
		for _, issue := range rec.Source.Issues {
			line.Warnings = append(line.Warnings, issue.Code+": "+issue.Message)
		}
		if rec.Source.Normalized.Funding != nil || rec.Source.Normalized.Subcategory != "" || rec.Source.Normalized.Priority != "" || rec.Source.Normalized.Recurring != nil {
			line.Warnings = append(line.Warnings, "Source metadata is retained as evidence; funding/subcategory/priority/recurrence are not financial-core fields")
		}
		cs, err := candidates(ctx, tx, batch.ID, report, rec)
		if err != nil {
			return p, err
		}
		line.Candidates = cs
		for _, c := range cs {
			if c.RecordID != "" {
				if _, exists := records[c.RecordID]; !exists {
					other, err := loadRecord(ctx, tx, batch.ID, c.RecordID)
					if err != nil {
						return p, err
					}
					records[c.RecordID] = other
				}
			}
		}
		for _, c := range cs {
			if !resolved(rec, c, records) {
				line.Blockers = append(line.Blockers, "unresolved_candidate:"+c.ID)
			}
		}
		d := rec.Decision
		n := rec.Source.Normalized
		switch d.Action {
		case "unresolved":
			line.Blockers = append(line.Blockers, "unresolved")
		case "exclude":
			p.Exclusions++
		case "duplicate_of":
			p.Links++
			canon, ok := records[d.CanonicalRecordID]
			if !ok || canon.ID == id || canon.Decision.Action != "create" && canon.Decision.Action != "link_existing" || !canon.Published && !selected[canon.ID] {
				line.Blockers = append(line.Blockers, "canonical_not_confirmed")
			}
			if ok && rec.Source.PDF != nil {
				var posts []financialevents.Posting
				if canon.EventID != nil {
					event, e := financialevents.LoadInTx(ctx, tx, *canon.EventID)
					if e != nil {
						return p, e
					}
					if event.VoidedAt != nil {
						line.Blockers = append(line.Blockers, "statement_event_voided")
					}
					if n.Funding != nil && n.Funding.Kind == "borrowed" && event.Kind == "income_received" {
						line.Blockers = append(line.Blockers, "borrowed_is_not_income")
					}
					posts = event.Postings
				} else if canon.Decision.Action == "link_existing" {
					event, e := financialevents.LoadInTx(ctx, tx, canon.Decision.EventID)
					if e != nil {
						return p, e
					}
					if event.VoidedAt != nil {
						line.Blockers = append(line.Blockers, "statement_event_voided")
					}
					if n.Funding != nil && n.Funding.Kind == "borrowed" && event.Kind == "income_received" {
						line.Blockers = append(line.Blockers, "borrowed_is_not_income")
					}
					posts = event.Postings
				} else if canon.Decision.Fields != nil {
					if n.Funding != nil && n.Funding.Kind == "borrowed" && canon.Decision.Fields.Kind == "income_received" {
						line.Blockers = append(line.Blockers, "borrowed_is_not_income")
					}
					cmd, e := command(canon.Decision.Fields, "preview:"+canon.ID)
					if e == nil {
						posts, e = financialevents.ValidateInTx(ctx, tx, cmd)
					}
					if e != nil {
						line.Blockers = append(line.Blockers, "invalid_canonical_event")
					}
				}
				if !statementPostingsMatch(rec.Source.PDF.AccountID, rec.Source.PDF.Row, posts) {
					line.Blockers = append(line.Blockers, "statement_posting_mismatch")
				}
			}
			if ok && rec.Source.PDF == nil && (canon.Source.Normalized.Kind != n.Kind || canon.Source.Normalized.Currency != n.Currency) {
				line.Blockers = append(line.Blockers, "incompatible_duplicate")
			}
		case "link_existing":
			p.Links++
			e, err := financialevents.LoadInTx(ctx, tx, d.EventID)
			if err != nil {
				if !errors.Is(err, pgx.ErrNoRows) {
					return p, err
				}
				line.Blockers = append(line.Blockers, "event_not_found")
			} else {
				if rec.Source.PDF != nil {
					if e.VoidedAt != nil {
						line.Blockers = append(line.Blockers, "statement_event_voided")
					}
					if n.Funding != nil && n.Funding.Kind == "borrowed" && e.Kind == "income_received" {
						line.Blockers = append(line.Blockers, "borrowed_is_not_income")
					}
					if !statementPostingsMatch(rec.Source.PDF.AccountID, rec.Source.PDF.Row, e.Postings) {
						line.Blockers = append(line.Blockers, "statement_posting_mismatch")
					}
				}
				if rec.Source.PDF == nil && (n.Kind == "" || e.Kind != n.Kind) {
					line.Blockers = append(line.Blockers, "incompatible_kind")
				}
				for _, post := range e.Postings {
					if post.Currency != n.Currency {
						line.Blockers = append(line.Blockers, "incompatible_currency")
					}
				}
				// Full confirmed event in preview allows inspecting every source difference.
				b, _ := json.Marshal(e)
				line.Warnings = append(line.Warnings, "Existing event: "+string(b))
			}
		case "create":
			p.NewEvents++
			f := d.Fields
			if rec.Source.PDF == nil && (n.Kind == "" || n.Kind == "unresolved_requirement") {
				line.Blockers = append(line.Blockers, "unsupported_semantics")
			}
			if d.Reason == "" {
				line.Blockers = append(line.Blockers, "review_reason_required")
			}
			if f != nil {
				if n.Kind != f.Kind && d.Reason == "" {
					line.Blockers = append(line.Blockers, "kind_conflict")
				}
				if n.Funding != nil && n.Funding.Kind == "borrowed" && f.Kind == "income_received" {
					line.Blockers = append(line.Blockers, "borrowed_is_not_income")
				}
				if n.Date == nil || n.Date.Precision == "date" || n.Date.Precision == "serial_fraction" {
					if f.TimeReason == "" && f.OccurredDate == "" {
						line.Blockers = append(line.Blockers, "time_confirmation_required")
					}
				}
			}
			cmd, err := command(f, "preview:"+id)
			if err != nil {
				line.Blockers = append(line.Blockers, asFault(err).Code)
			} else {
				postings, err := financialevents.ValidateInTx(ctx, tx, cmd)
				if errors.Is(err, financialevents.ErrInvalid) {
					line.Blockers = append(line.Blockers, "invalid_financial_event")
				} else if err != nil {
					return p, err
				} else {
					if rec.Source.PDF != nil && !statementPostingsMatch(rec.Source.PDF.AccountID, rec.Source.PDF.Row, postings) {
						line.Blockers = append(line.Blockers, "statement_posting_mismatch")
					}
					for _, post := range postings {
						if post.Currency != f.Currency {
							line.Blockers = append(line.Blockers, "currency_mismatch")
						}
						delta := post.Amount
						if post.Direction == "debit" {
							delta = -delta
						}
						change := Change{post.AccountID, post.Currency, delta, post.Account}
						line.Postings = append(line.Postings, change)
						k := change.AccountID + ":" + change.Currency
						previous := deltas[k]
						total, ok := add(previous.DeltaMinor, delta)
						if !ok {
							line.Blockers = append(line.Blockers, "amount_overflow")
						} else {
							change.DeltaMinor = total
							deltas[k] = change
						}
					}
					if cmd.Kind == "refund" {
						sum, ok := add(refundTotals[cmd.OriginalEventID], cmd.AmountMinor)
						if !ok {
							line.Blockers = append(line.Blockers, "amount_overflow")
						} else {
							refundTotals[cmd.OriginalEventID] = sum
						}
					}
				}
			}
		default:
			line.Blockers = append(line.Blockers, "invalid_action")
		}
		for _, blocker := range line.Blockers {
			p.Blockers = append(p.Blockers, id+":"+blocker)
		}
		p.Lines = append(p.Lines, line)
	}
	for original, total := range refundTotals {
		var remaining int64
		if err := tx.QueryRow(ctx, `SELECT p.amount_minor-COALESCE((SELECT sum(rp.amount_minor) FROM financial_events re JOIN postings rp ON rp.financial_event_id=re.id WHERE re.original_event_id=e.id AND re.kind='refund'),0) FROM financial_events e JOIN postings p ON p.financial_event_id=e.id WHERE e.id=$1 AND e.kind='purchase'`, original).Scan(&remaining); err != nil {
			return p, err
		}
		if total > remaining {
			p.Blockers = append(p.Blockers, "refund_group_exceeds_purchase:"+original)
		}
	}
	for _, change := range deltas {
		p.Changes = append(p.Changes, change)
	}
	sort.Slice(p.Changes, func(i, j int) bool {
		return p.Changes[i].AccountID+":"+p.Changes[i].Currency < p.Changes[j].AccountID+":"+p.Changes[j].Currency
	})
	sort.Strings(p.Blockers)
	p.Hash = planHash(p)
	return p, nil
}
func (r *Repository) Preview(ctx context.Context, batch, key string, input Selection) (Preview, error) {
	// Either explicit ids (a page selection) or a filter (SPEC-0044), never both.
	byFilter := input.Filter != nil && len(input.RecordIDs) == 0
	if !byFilter && (input.Filter != nil || !selectionValid(input.RecordIDs)) {
		return Preview{}, fault("invalid_selection", 400)
	}
	tx, err := r.begin(ctx, true)
	if err != nil {
		return Preview{}, err
	}
	defer tx.Rollback(ctx)
	var p Preview
	scope := "preview:" + batch
	replay, err := request(ctx, tx, scope, key, input, &p)
	if err != nil || replay {
		return p, err
	}
	b, report, err := loadBatch(ctx, tx, batch, true)
	if err != nil {
		return p, err
	}
	if b.Revision != input.ExpectedRevision {
		return p, fault("revision_conflict", 409)
	}
	ids := input.RecordIDs
	if byFilter {
		// The revision lock makes the filter resolve to the rows the person saw.
		if ids, err = filteredIDs(ctx, tx, batch, *input.Filter); err != nil {
			return p, err
		}
	}
	p, err = r.plan(ctx, tx, b, report, ids)
	if err != nil {
		return p, err
	}
	p.ExpiresAt = r.now().UTC().Add(15 * time.Minute)
	raw, _ := json.Marshal(p)
	err = tx.QueryRow(ctx, "INSERT INTO import_previews(batch_id,revision,content,content_hash,expires_at) VALUES($1,$2,$3,$4,$5) RETURNING id::text", batch, p.Revision, raw, p.Hash, p.ExpiresAt).Scan(&p.ID)
	if err != nil {
		return p, err
	}
	if err = remember(ctx, tx, scope, key, input, p); err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}
func (r *Repository) Confirm(ctx context.Context, batch, key string, input Confirmation) (Receipt, error) {
	result := Receipt{Outcomes: []Outcome{}}
	if !uuid.MatchString(input.PreviewID) || len(input.PreviewHash) != 64 {
		return result, fault("invalid_preview", 400)
	}
	tx, err := r.begin(ctx, true)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	// Replay is checked before expiry, revision and candidate changes.
	scope := "confirm"
	payload := struct {
		Batch string
		Input Confirmation
	}{batch, input}
	replay, err := request(ctx, tx, scope, key, payload, &result)
	if err != nil || replay {
		return result, err
	}
	b, report, err := loadBatch(ctx, tx, batch, true)
	if err != nil {
		return result, err
	}
	var raw []byte
	var hash string
	var expires time.Time
	var revision int64
	err = tx.QueryRow(ctx, "SELECT content,content_hash,expires_at,revision FROM import_previews WHERE id=$1 AND batch_id=$2", input.PreviewID, batch).Scan(&raw, &hash, &expires, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	}
	if err != nil {
		return result, err
	}
	if hash != input.PreviewHash {
		return result, fault("preview_changed", 409)
	}
	if !r.now().Before(expires) {
		return result, fault("preview_expired", 409)
	}
	if revision != b.Revision {
		return result, fault("revision_conflict", 409)
	}
	var previous Preview
	if err = json.Unmarshal(raw, &previous); err != nil {
		return result, err
	}
	ids := []string{}
	for _, line := range previous.Lines {
		ids = append(ids, line.RecordID)
	}
	current, err := r.plan(ctx, tx, b, report, ids)
	if err != nil {
		return result, err
	}
	if current.Hash != hash {
		return result, fault("preview_changed", 409)
	}
	if len(current.Blockers) > 0 {
		return result, fault("preview_blocked", 422)
	}
	result.BatchID = batch
	result.PreviewID = input.PreviewID
	result.CreatedAt = r.now().UTC()
	if err = tx.QueryRow(ctx, "INSERT INTO import_confirmations(batch_id,preview_id,idempotency_key,request_hash,receipt) VALUES($1,$2,$3,$4,'{}') RETURNING id::text", batch, input.PreviewID, key, digest(payload)).Scan(&result.ID); err != nil {
		return result, err
	}
	events := map[string]*string{}
	for _, line := range current.Lines {
		d := line.Decision
		switch d.Action {
		case "create":
			cmd, err := command(d.Fields, "import:"+line.RecordID)
			if err != nil {
				return result, err
			}
			event, _, err := financialevents.CreateInTx(ctx, tx, cmd, "import")
			if err != nil {
				return result, err
			}
			id := event.ID
			events[line.RecordID] = &id
		case "link_existing":
			id := d.EventID
			events[line.RecordID] = &id
		case "exclude":
			events[line.RecordID] = nil
		}
	}
	for _, line := range current.Lines {
		if line.Decision.Action == "duplicate_of" {
			canonical := line.Decision.CanonicalRecordID
			if e, ok := events[canonical]; ok {
				events[line.RecordID] = e
			} else {
				var e string
				if err = tx.QueryRow(ctx, "SELECT event_id::text FROM import_provenance WHERE record_id=$1 AND event_id IS NOT NULL", canonical).Scan(&e); err != nil {
					return result, err
				}
				events[line.RecordID] = &e
			}
		}
		eventID := events[line.RecordID]
		decision, _ := json.Marshal(line.Decision)
		if _, err = tx.Exec(ctx, "INSERT INTO import_provenance(record_id,batch_id,confirmation_id,event_id,action,decision) VALUES($1,$2,$3,$4,$5,$6)", line.RecordID, batch, result.ID, eventID, line.Decision.Action, decision); err != nil {
			return result, err
		}
		if _, err = tx.Exec(ctx, "UPDATE import_records SET published=true WHERE id=$1 AND NOT published", line.RecordID); err != nil {
			return result, err
		}
		outcome := Outcome{RecordID: line.RecordID, Action: line.Decision.Action, EventID: eventID}
		if eventID != nil {
			event, err := financialevents.LoadInTx(ctx, tx, *eventID)
			if err != nil {
				return result, err
			}
			outcome.Event = &event
		}
		result.Outcomes = append(result.Outcomes, outcome)
	}
	if err = r.statementAfterConfirm(ctx, tx, batch); err != nil {
		return result, err
	}
	// Only the in-flight receipt can be updated; readers cannot mutate confirmations.
	receipt, _ := json.Marshal(result)
	if _, err = tx.Exec(ctx, "UPDATE import_confirmations SET receipt=$1 WHERE id=$2", receipt, result.ID); err != nil {
		return result, err
	}
	if _, err = tx.Exec(ctx, "UPDATE import_batches SET revision=revision+1,status=CASE WHEN EXISTS(SELECT 1 FROM import_records WHERE batch_id=$1 AND domain='transaction' AND NOT published) THEN 'reviewing' ELSE 'completed' END WHERE id=$1", batch); err != nil {
		return result, err
	}
	if err = remember(ctx, tx, scope, key, payload, result); err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}
func (r *Repository) Receipt(ctx context.Context, batch, id string) (Receipt, error) {
	tx, err := r.begin(ctx, false)
	if err != nil {
		return Receipt{}, err
	}
	defer tx.Rollback(ctx)
	var result Receipt
	var raw []byte
	err = tx.QueryRow(ctx, "SELECT receipt FROM import_confirmations WHERE batch_id=$1 AND id=$2", batch, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, ErrNotFound
	}
	if err != nil {
		return result, err
	}
	return result, json.Unmarshal(raw, &result)
}
