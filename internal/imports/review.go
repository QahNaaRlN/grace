package imports

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// SPEC-0044: review hundreds or thousands of rows by filter, sort and search
// on the server, and walk the unresolved ones without scrolling.

// MaxFilterSelection bounds a preview built from a filter (decision 1).
const MaxFilterSelection = 1000

type RecordQuery struct {
	RecordFilter
	Cursor string
	Limit  int
	// Sort: row (default), date, -date, amount, -amount.
	Sort string
}

var sortColumns = map[string]string{
	"row":     "r.source->>'sheet', (r.source->>'row')::int, r.id",
	"date":    "r.source->'normalized'->'date'->>'local' NULLS LAST, (r.source->>'row')::int, r.id",
	"-date":   "r.source->'normalized'->'date'->>'local' DESC NULLS LAST, (r.source->>'row')::int, r.id",
	"amount":  "(r.source->'normalized'->>'amountMinor')::bigint NULLS LAST, (r.source->>'row')::int, r.id",
	"-amount": "(r.source->'normalized'->>'amountMinor')::bigint DESC NULLS LAST, (r.source->>'row')::int, r.id",
}

var statuses = map[string]bool{"": true, "needs_review": true, "unresolved": true, "create": true, "link_existing": true, "duplicate_of": true, "exclude": true, "published": true}

// cleanFileName keeps only the base name, never a client path, up to 255 runes.
func cleanFileName(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	if utf8.RuneCountInString(name) > 255 {
		name = string([]rune(name)[:255])
	}
	return name
}

func parseOffset(cursor string) (int, bool) {
	if cursor == "" {
		return 0, true
	}
	n, err := strconv.Atoi(cursor)
	return n, err == nil && n >= 0 && n <= 1_000_000
}

func (f RecordFilter) valid() bool {
	return statuses[f.Status] && (f.Domain == "" || f.Domain == "transaction" || f.Domain == "unsupported") && utf8.RuneCountInString(f.Q) <= 100
}

// where builds the shared filter; $1 is always the batch id.
func (f RecordFilter) where() (string, []any) {
	args := []any{nil}
	add := func(v any) string { args = append(args, v); return fmt.Sprintf("$%d", len(args)) }
	conditions := []string{"r.batch_id = $1"}
	switch f.Domain {
	case "transaction":
		conditions = append(conditions, "r.domain = 'transaction'")
	case "unsupported":
		conditions = append(conditions, "r.domain <> 'transaction'")
	}
	switch f.Status {
	case "":
	case "needs_review":
		conditions = append(conditions, "NOT r.published AND r.questions > 0")
	case "published":
		conditions = append(conditions, "r.published")
	default:
		conditions = append(conditions, "NOT r.published AND r.decision->>'action' = "+add(f.Status))
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(q)) + "%"
		p := add(pattern)
		conditions = append(conditions, "(lower(COALESCE(r.source->'normalized'->>'description','')) LIKE "+p+
			" OR lower(COALESCE(r.source->'normalized'->>'recipient','')) LIKE "+p+")")
	}
	return strings.Join(conditions, " AND "), args
}

func (r *Repository) RecordsQuery(ctx context.Context, batch string, query RecordQuery) (Page[Record], error) {
	page := Page[Record]{Items: []Record{}}
	offset, ok := parseOffset(query.Cursor)
	order, sorted := sortColumns[query.Sort]
	if query.Sort == "" {
		order, sorted = sortColumns["row"], true
	}
	if !ok || !sorted || !query.valid() || query.Limit < 1 || query.Limit > 100 {
		return page, fault("invalid_pagination", 400)
	}
	tx, err := r.begin(ctx, false)
	if err != nil {
		return page, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM import_batches WHERE id=$1)", batch).Scan(&exists); err != nil {
		return page, err
	}
	if !exists {
		return page, ErrNotFound
	}
	where, args := query.where()
	args[0] = batch
	var total int
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM import_records r WHERE "+where, args...).Scan(&total); err != nil {
		return page, err
	}
	page.Total = &total
	args = append(args, offset, query.Limit+1)
	rows, err := tx.Query(ctx, recordSQL+"WHERE "+where+" ORDER BY "+order+fmt.Sprintf(" OFFSET $%d LIMIT $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		rec, err := readRecord(rows)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, rec)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	if len(page.Items) > query.Limit {
		page.Items = page.Items[:query.Limit]
		page.NextCursor = strconv.Itoa(offset + query.Limit)
	}
	return page, nil
}

type NextUnresolved struct {
	Record    *Record `json:"record"`
	Remaining int     `json:"remaining"`
}

// NextUnresolved returns the first unresolved transaction row after the given
// one in sheet order, wrapping around, so saving a decision leads straight to
// the next row that still needs one.
func (r *Repository) NextUnresolved(ctx context.Context, batch, after string) (NextUnresolved, error) {
	var result NextUnresolved
	tx, err := r.begin(ctx, false)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	const open = "r.batch_id=$1 AND r.domain='transaction' AND NOT r.published AND r.decision->>'action'='unresolved'"
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM import_batches WHERE id=$1)", batch).Scan(&exists); err != nil {
		return result, err
	}
	if !exists {
		return result, ErrNotFound
	}
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM import_records r WHERE "+open, batch).Scan(&result.Remaining); err != nil {
		return result, err
	}
	// Rows after `after` first (rank 0), then from the start (rank 1).
	rec, err := readRecord(tx.QueryRow(ctx, recordSQL+`
		LEFT JOIN import_records a ON a.id::text = $2 AND a.batch_id = $1
		WHERE `+open+` AND r.id::text <> $2
		ORDER BY CASE WHEN a.id IS NOT NULL AND (r.source->>'sheet', (r.source->>'row')::int, r.id::text) > (a.source->>'sheet', (a.source->>'row')::int, a.id::text) THEN 0 ELSE 1 END,
			r.source->>'sheet', (r.source->>'row')::int, r.id
		LIMIT 1`, batch, after))
	if errors.Is(err, ErrNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Record = &rec
	return result, nil
}

// filteredIDs resolves a filter selection to selectable rows.
func filteredIDs(ctx context.Context, tx pgx.Tx, batch string, filter RecordFilter) ([]string, error) {
	if !filter.valid() {
		return nil, fault("invalid_selection", 400)
	}
	where, args := filter.where()
	args[0] = batch
	rows, err := tx.Query(ctx, "SELECT r.id::text FROM import_records r WHERE "+where+
		" AND r.domain='transaction' AND NOT r.published ORDER BY "+sortColumns["row"]+fmt.Sprintf(" LIMIT %d", MaxFilterSelection+1), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) > MaxFilterSelection {
		return nil, fault("selection_too_large", 422)
	}
	if len(ids) == 0 {
		return nil, fault("invalid_selection", 400)
	}
	return ids, nil
}

// summarize fills the history and summary fields of a batch header.
func summarize(ctx context.Context, tx pgx.Tx, b *Batch) error {
	err := tx.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE domain='transaction'),
		count(*) FILTER (WHERE domain='transaction' AND (published OR decision->>'action'<>'unresolved')),
		count(*) FILTER (WHERE domain='transaction' AND NOT published AND questions>0),
		min(left(source->'normalized'->'date'->>'local', 10)) FILTER (WHERE domain='transaction'),
		max(left(source->'normalized'->'date'->>'local', 10)) FILTER (WHERE domain='transaction')
		FROM import_records WHERE batch_id=$1`, b.ID).Scan(&b.Total, &b.Reviewed, &b.NeedsReview, &b.PeriodFrom, &b.PeriodTo)
	if err != nil {
		return err
	}
	var last ConfirmationSummary
	err = tx.QueryRow(ctx, `SELECT c.id::text, c.created_at,
		count(p.record_id) FILTER (WHERE p.action='create'), count(p.record_id)
		FROM import_confirmations c LEFT JOIN import_provenance p ON p.confirmation_id=c.id AND p.owner_id=c.owner_id
		WHERE c.batch_id=$1 GROUP BY c.id, c.created_at ORDER BY c.created_at DESC, c.id DESC LIMIT 1`, b.ID).
		Scan(&last.ID, &last.At, &last.CreatedEvents, &last.Outcomes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	b.LastConfirmation = &last
	return nil
}
