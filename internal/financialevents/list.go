package financialevents

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SPEC-0043: the operations list pages by cursor, filters on the server and
// returns per-day totals computed here, not in the browser.

var ErrInvalidCursor = errors.New("invalid cursor")

const dateLayout = "2006-01-02"

type ListFilter struct {
	Limit     int
	Cursor    string
	Category  string
	From      string
	To        string
	AccountID string
	Kind      string
	Query     string
	// Voided lists only voided operations; otherwise they are excluded.
	Voided bool
}

type DayTotal struct {
	Date         string `json:"date"`
	Currency     string `json:"currency"`
	InflowMinor  int64  `json:"inflowMinor"`
	OutflowMinor int64  `json:"outflowMinor"`
}

type Page struct {
	Items      []Event    `json:"items"`
	NextCursor *string    `json:"nextCursor"`
	DayTotals  []DayTotal `json:"dayTotals"`
}

// Date-only events sort at the start of their local day, after timed ones in
// descending order; id breaks ties so the cursor is stable.
const (
	sortKey   = `COALESCE(e.occurred_at, e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg')`
	localDate = `COALESCE(e.occurred_date, (e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)`
)

type cursor struct {
	at      time.Time
	created time.Time
	id      string
}

func encodeCursor(event Event) string {
	at := event.CreatedAt
	if event.OccurredAt != nil {
		at = *event.OccurredAt
	} else if event.OccurredDate != nil {
		if day, err := time.ParseInLocation(dateLayout, *event.OccurredDate, yekaterinburg()); err == nil {
			at = day
		}
	}
	raw := strings.Join([]string{at.UTC().Format(time.RFC3339Nano), event.CreatedAt.UTC().Format(time.RFC3339Nano), event.ID}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(value string) (cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursor{}, ErrInvalidCursor
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return cursor{}, ErrInvalidCursor
	}
	at, errAt := time.Parse(time.RFC3339Nano, parts[0])
	created, errCreated := time.Parse(time.RFC3339Nano, parts[1])
	if errAt != nil || errCreated != nil || len(parts[2]) != 36 {
		return cursor{}, ErrInvalidCursor
	}
	return cursor{at: at, created: created, id: parts[2]}, nil
}

func yekaterinburg() *time.Location {
	location, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		return time.FixedZone("Asia/Yekaterinburg", 5*60*60)
	}
	return location
}

// where builds the filter shared by the page and its day totals.
func (filter ListFilter) where(args *[]any) string {
	add := func(value any) string {
		*args = append(*args, value)
		return fmt.Sprintf("$%d", len(*args))
	}
	conditions := []string{"e.voided_at IS NULL"}
	if filter.Voided {
		conditions[0] = "e.voided_at IS NOT NULL"
	}
	switch filter.Category {
	case "":
	case "uncategorized":
		conditions = append(conditions, "e.category_id IS NULL AND e.kind='purchase'")
	default:
		conditions = append(conditions, "e.category_id::text="+add(filter.Category))
	}
	if filter.From != "" {
		conditions = append(conditions, localDate+">="+add(filter.From)+"::date")
	}
	if filter.To != "" {
		conditions = append(conditions, localDate+"<"+add(filter.To)+"::date")
	}
	if filter.AccountID != "" {
		conditions = append(conditions, "(SELECT bool_or(f.account_id::text="+add(filter.AccountID)+") FROM (SELECT account_id FROM postings WHERE financial_event_id=e.id UNION ALL SELECT account_id FROM voided_postings WHERE financial_event_id=e.id) f)")
	}
	if filter.Kind != "" {
		conditions = append(conditions, "e.kind::text="+add(filter.Kind))
	}
	if query := strings.TrimSpace(filter.Query); query != "" {
		pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(query)) + "%"
		placeholder := add(pattern)
		conditions = append(conditions, "(lower(e.merchant) LIKE "+placeholder+" OR lower(e.description) LIKE "+placeholder+")")
	}
	return strings.Join(conditions, " AND ")
}

func (repo *Repository) ListPage(ctx context.Context, filter ListFilter) (Page, error) {
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 50
	}
	args := []any{}
	where := filter.where(&args)
	if filter.Cursor != "" {
		position, err := decodeCursor(filter.Cursor)
		if err != nil {
			return Page{}, err
		}
		args = append(args, position.at, position.created, position.id)
		where += fmt.Sprintf(" AND (%s, e.created_at, e.id) < ($%d, $%d, $%d::uuid)", sortKey, len(args)-2, len(args)-1, len(args))
	}
	args = append(args, filter.Limit+1)
	rows, err := repo.pool.Query(ctx, eventQuery+`
		WHERE e.id IN (
			SELECT e.id FROM financial_events e WHERE `+where+`
			ORDER BY `+sortKey+` DESC, e.created_at DESC, e.id DESC
			LIMIT $`+fmt.Sprint(len(args))+`
		)
		ORDER BY `+sortKey+` DESC, e.created_at DESC, e.id DESC, p.created_at, p.id`, args...)
	if err != nil {
		return Page{}, fmt.Errorf("list financial events page: %w", err)
	}
	items, err := collectEvents(rows)
	rows.Close()
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: items, DayTotals: []DayTotal{}}
	if len(items) > filter.Limit {
		page.Items = items[:filter.Limit]
		next := encodeCursor(page.Items[len(page.Items)-1])
		page.NextCursor = &next
	}
	if len(page.Items) == 0 {
		return page, nil
	}
	page.DayTotals, err = repo.dayTotals(ctx, filter, page.Items)
	return page, err
}

// dayTotals sums whole days present on the page over the full filter, so a day
// split across pages still shows its complete total. Internal transfers move
// money between own accounts and are neither inflow nor outflow.
func (repo *Repository) dayTotals(ctx context.Context, filter ListFilter, items []Event) ([]DayTotal, error) {
	days := make([]string, 0)
	seen := map[string]bool{}
	for _, item := range items {
		day := ""
		switch {
		case item.OccurredDate != nil:
			day = *item.OccurredDate
		case item.OccurredAt != nil:
			day = item.OccurredAt.In(yekaterinburg()).Format(dateLayout)
		}
		if day != "" && !seen[day] {
			seen[day] = true
			days = append(days, day)
		}
	}
	args := []any{}
	where := filter.where(&args)
	args = append(args, days)
	daysParam := fmt.Sprintf("$%d", len(args))
	postingFilter := ""
	if filter.AccountID != "" {
		args = append(args, filter.AccountID)
		postingFilter = fmt.Sprintf(" AND p.account_id::text=$%d", len(args))
	}
	rows, err := repo.pool.Query(ctx, `
		SELECT `+localDate+`::text AS day, p.currency,
			COALESCE(sum(p.amount_minor) FILTER (WHERE p.direction='credit'),0)::bigint,
			COALESCE(sum(p.amount_minor) FILTER (WHERE p.direction='debit'),0)::bigint
		FROM financial_events e JOIN postings p ON p.financial_event_id=e.id
		WHERE `+where+` AND e.kind<>'internal_transfer' AND `+localDate+`::text = ANY(`+daysParam+`)`+postingFilter+`
		GROUP BY day, p.currency
		ORDER BY day DESC, p.currency`, args...)
	if err != nil {
		return nil, fmt.Errorf("day totals: %w", err)
	}
	defer rows.Close()
	totals := make([]DayTotal, 0)
	for rows.Next() {
		var total DayTotal
		if err := rows.Scan(&total.Date, &total.Currency, &total.InflowMinor, &total.OutflowMinor); err != nil {
			return nil, err
		}
		totals = append(totals, total)
	}
	return totals, rows.Err()
}
