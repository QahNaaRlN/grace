package imports

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/jackc/pgx/v5"
)

const monthZone = "Asia/Yekaterinburg"

type MonthRow struct {
	RecordID       string   `json:"recordId"`
	Sheet          string   `json:"sheet"`
	Row            int      `json:"row"`
	SourceDate     string   `json:"sourceDate"`
	SourceKind     string   `json:"sourceKind"`
	SourceAmount   *int64   `json:"sourceAmountMinor"`
	SourceCurrency string   `json:"sourceCurrency"`
	Description    string   `json:"description"`
	Action         string   `json:"action"`
	Published      bool     `json:"published"`
	Reason         string   `json:"reason"`
	TargetDate     string   `json:"targetDate"`
	TargetKind     string   `json:"targetKind"`
	TargetAmount   *int64   `json:"targetAmountMinor"`
	TargetCurrency string   `json:"targetCurrency"`
	Differences    []string `json:"differences"`
}
type MonthTotal struct {
	Currency   string `json:"currency"`
	SourceRows int    `json:"sourceRows"`
	Source     *int64 `json:"sourceMinor"`
	Prepared   *int64 `json:"preparedMinor"`
	Confirmed  *int64 `json:"confirmedMinor"`
}
type MonthReport struct {
	BatchID         string       `json:"batchId"`
	Revision        int64        `json:"batchRevision"`
	Month           string       `json:"month"`
	TimeZone        string       `json:"timeZone"`
	Complete        bool         `json:"complete"`
	OutsidePeriod   int          `json:"outsidePeriod"`
	Pending         int          `json:"pendingRows"`
	Totals          []MonthTotal `json:"totals"`
	Rows            []MonthRow   `json:"rows"`
	UnknownDateRows []MonthRow   `json:"unknownDateRows"`
}

func validMonth(month string) bool {
	t, err := time.Parse("2006-01", month)
	return err == nil && len(month) == 7 && t.Year() >= 1 && t.Format("2006-01") == month
}

func sumMinor(total **int64, amount *int64) {
	if *total == nil {
		return
	}
	if amount == nil || *amount <= 0 {
		*total = nil
		return
	}
	v, ok := add(**total, *amount)
	if !ok {
		*total = nil
	} else {
		*total = &v
	}
}

func monthRow(rec Record, event *financialevents.Event) MonthRow {
	n := rec.Source.Normalized
	row := MonthRow{RecordID: rec.ID, Sheet: rec.Source.Sheet, Row: rec.Source.Row, SourceKind: n.Kind, SourceAmount: n.AmountMinor, SourceCurrency: n.Currency, Description: n.Description, Action: rec.Decision.Action, Published: rec.Published, Reason: rec.Decision.Reason, Differences: []string{}}
	if n.Date != nil {
		row.SourceDate = n.Date.Local
	}
	if rec.Decision.Action == "create" && rec.Decision.Fields != nil {
		f := rec.Decision.Fields
		row.TargetDate, row.TargetKind, row.TargetCurrency = f.OccurredAt, f.Kind, f.Currency
		amount := f.AmountMinor
		row.TargetAmount = &amount
	}
	if event != nil {
		row.TargetDate, row.TargetKind = event.OccurredAt.Format(time.RFC3339), event.Kind
		for _, post := range event.Postings {
			if post.Direction == "debit" {
				amount := post.Amount
				row.TargetAmount, row.TargetCurrency = &amount, post.Currency
				break
			}
		}
	}
	if rec.Decision.Action == "exclude" || rec.Decision.Action == "duplicate_of" {
		row.Differences = append(row.Differences, rec.Decision.Action)
	}
	if rec.Decision.Action == "unresolved" {
		row.Differences = append(row.Differences, "unresolved")
	}
	if row.TargetKind != "" {
		if row.TargetKind != row.SourceKind {
			row.Differences = append(row.Differences, "kind_changed")
		}
		if row.SourceAmount == nil || row.TargetAmount == nil || *row.SourceAmount != *row.TargetAmount {
			row.Differences = append(row.Differences, "amount_changed")
		}
		if row.SourceCurrency != row.TargetCurrency {
			row.Differences = append(row.Differences, "currency_changed")
		}
		if date := targetLocalDate(row.TargetDate); len(row.SourceDate) < 10 || date == "" || date != row.SourceDate[:10] {
			row.Differences = append(row.Differences, "date_changed")
		}
	}
	return row
}

func targetLocalDate(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ""
	}
	zone, err := time.LoadLocation(monthZone)
	if err != nil {
		return ""
	}
	return t.In(zone).Format("2006-01-02")
}

// buildMonth processes the entire batch, never just a page of source records.
func buildMonth(batch Batch, month string, records []Record, events map[string]financialevents.Event) MonthReport {
	report := MonthReport{BatchID: batch.ID, Revision: batch.Revision, Month: month, TimeZone: monthZone, Totals: []MonthTotal{}, Rows: []MonthRow{}, UnknownDateRows: []MonthRow{}}
	totals := map[string]*MonthTotal{}
	totalFor := func(currency string) *MonthTotal {
		if totals[currency] == nil {
			a, b, c := int64(0), int64(0), int64(0)
			totals[currency] = &MonthTotal{Currency: currency, Source: &a, Prepared: &b, Confirmed: &c}
		}
		return totals[currency]
	}
	// Count an existing linked event only once, including across prepared and
	// confirmed decisions. Publication takes precedence over prepared evidence.
	seen := map[string]bool{}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Published != records[j].Published {
			return records[i].Published
		}
		return records[i].ID < records[j].ID
	})
	for _, rec := range records {
		if rec.Source.Domain != "transaction" {
			continue
		}
		var e *financialevents.Event
		eventID := rec.Decision.EventID
		if rec.EventID != nil {
			eventID = *rec.EventID
		}
		if event, ok := events[eventID]; ok {
			e = &event
		}
		row := monthRow(rec, e)
		var sourceDate time.Time
		var err error
		if len(row.SourceDate) >= 10 {
			sourceDate, err = time.Parse("2006-01-02", row.SourceDate[:10])
		} else {
			err = fault("invalid_date", 400)
		}
		// An explicit reviewed date may resolve a missing/invalid source date,
		// while the immutable source and its uncertainty stay visible in the row.
		if err != nil && rec.Decision.Reason != "" && (rec.Decision.Action == "create" && rec.Decision.Fields != nil && rec.Decision.Fields.TimeReason != "" || rec.Decision.Action == "link_existing" && e != nil) {
			sourceDate, err = time.Parse("2006-01-02", targetLocalDate(row.TargetDate))
		}
		if err != nil {
			report.UnknownDateRows = append(report.UnknownDateRows, row)
			continue
		}
		if sourceDate.Format("2006-01") != month {
			report.OutsidePeriod++
			continue
		}
		if !row.Published {
			report.Pending++
		}
		if row.SourceKind == "purchase" {
			t := totalFor(row.SourceCurrency)
			t.SourceRows++
			sumMinor(&t.Source, row.SourceAmount)
		}
		if row.Action == "create" || row.Action == "link_existing" {
			if row.TargetKind == "purchase" && strings.HasPrefix(targetLocalDate(row.TargetDate), month+"-") {
				identity := "record:" + rec.ID
				if eventID != "" {
					identity = "event:" + eventID
				}
				if !seen[identity] {
					t := totalFor(row.TargetCurrency)
					if row.Published {
						sumMinor(&t.Confirmed, row.TargetAmount)
					} else {
						sumMinor(&t.Prepared, row.TargetAmount)
					}
					seen[identity] = true
				}
			} else if row.TargetKind == "" || row.TargetKind == "purchase" && targetLocalDate(row.TargetDate) == "" {
				row.Differences = append(row.Differences, "target_unknown")
				if row.SourceKind == "purchase" {
					t := totalFor(row.SourceCurrency)
					if row.Published {
						t.Confirmed = nil
					} else {
						t.Prepared = nil
					}
				}
			}
		}
		report.Rows = append(report.Rows, row)
	}
	for _, t := range totals {
		report.Totals = append(report.Totals, *t)
	}
	sort.Slice(report.Totals, func(i, j int) bool { return report.Totals[i].Currency < report.Totals[j].Currency })
	less := func(a, b MonthRow) bool {
		if a.Sheet != b.Sheet {
			return a.Sheet < b.Sheet
		}
		return a.Row < b.Row
	}
	sort.Slice(report.Rows, func(i, j int) bool { return less(report.Rows[i], report.Rows[j]) })
	sort.Slice(report.UnknownDateRows, func(i, j int) bool { return less(report.UnknownDateRows[i], report.UnknownDateRows[j]) })
	unknownBlocked := false
	for _, row := range report.UnknownDateRows {
		if !row.Published || row.Action != "exclude" || row.Reason == "" {
			unknownBlocked = true
		}
	}
	report.Complete = len(report.Rows) > 0 && report.Pending == 0 && !unknownBlocked
	return report
}

func (r *Repository) Month(ctx context.Context, batchID, month string) (MonthReport, error) {
	if !validMonth(month) {
		return MonthReport{}, fault("invalid_month", 400)
	}
	// One consistent read for revision, decisions, provenance and linked events.
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return MonthReport{}, err
	}
	defer tx.Rollback(ctx)
	batch, _, err := loadBatch(ctx, tx, batchID, false)
	if err != nil {
		return MonthReport{}, err
	}
	rows, err := tx.Query(ctx, recordSQL+"WHERE r.batch_id=$1 AND r.domain='transaction' ORDER BY r.id", batchID)
	if err != nil {
		return MonthReport{}, err
	}
	records := []Record{}
	for rows.Next() {
		rec, e := readRecord(rows)
		if e != nil {
			rows.Close()
			return MonthReport{}, e
		}
		records = append(records, rec)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return MonthReport{}, err
	}
	events := map[string]financialevents.Event{}
	for _, rec := range records {
		id := rec.Decision.EventID
		if rec.EventID != nil {
			id = *rec.EventID
		}
		if id == "" {
			continue
		}
		if _, ok := events[id]; ok {
			continue
		}
		event, err := financialevents.LoadInTx(ctx, tx, id)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return MonthReport{}, err
		}
		events[id] = event
	}
	return buildMonth(batch, month, records, events), nil
}
