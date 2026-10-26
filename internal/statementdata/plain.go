package statementdata

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const PlainVersion = "statement.plain.0058.v1"

var PlainFields = []string{"operationDate", "bookingDate", "transactionAmount", "transactionCurrency", "accountAmount", "accountCurrency", "description", "cardLast4"}
var PlainHeaderFields = []string{"periodFrom", "periodTo", "availableBalanceDate", "availableBalanceObserved", "creditTurnover", "debitTurnover", "currency"}

type PlainRow struct {
	ID                string            `json:"id"`
	Raw               map[string]string `json:"raw"`
	OperationDate     *LocalDate        `json:"operationDate"`
	BookingDate       *LocalDate        `json:"bookingDate"`
	TransactionAmount *Money            `json:"transactionAmount"`
	AccountAmount     *Money            `json:"accountAmount"`
	Description       *string           `json:"description"`
	CardLast4         *string           `json:"cardLast4"`
	Issues            []string          `json:"issues"`
}
type PlainReading struct {
	Version     string            `json:"version"`
	RawAnswer   string            `json:"rawAnswer"`
	ParseStatus string            `json:"parseStatus"`
	Rows        []PlainRow        `json:"rows"`
	Header      map[string]string `json:"header"`
}
type PlainState struct {
	SourceSHA256   string       `json:"sourceSHA256"`
	Candidate      PlainReading `json:"candidate"`
	Original       PlainReading `json:"original"`
	Digest         string       `json:"digest"`
	ReviewedDigest string       `json:"reviewedDigest"`
}

func (p PlainState) CurrentDigest() string {
	return VisualHash(struct {
		Source  string
		Reading PlainReading
	}{p.SourceSHA256, p.Candidate})
}
func PlainEmpty() PlainReading {
	return PlainReading{Version: PlainVersion, ParseStatus: "manual", Rows: []PlainRow{}, Header: map[string]string{}}
}
func absent(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "—", "–", "-", "null", "не указано", "не указан", "неразборчиво", "нет":
		return true
	}
	return false
}
func signedMoney(raw, ccy string) (*Money, error) {
	c, e := Currency(ccy)
	if e != nil {
		return nil, e
	}
	m, d, e := ParseMoney(raw, c)
	if e != nil {
		return nil, e
	}
	if d == "debit" {
		m.AmountMinor = -m.AmountMinor
	}
	return &m, nil
}

// Normalize distrusts all supplied typed fields and issues. Only raw cells are editable.
func (p *PlainReading) Normalize() error {
	if p.Version != PlainVersion || len(p.RawAnswer) > 256<<10 || len(p.Rows) > 200 || len(p.Header) > len(PlainHeaderFields) {
		return errors.New("plain_draft_invalid")
	}
	if p.ParseStatus != "parsed" && p.ParseStatus != "parse_error" && p.ParseStatus != "manual" {
		return errors.New("plain_draft_invalid")
	}
	for k, v := range p.Header {
		if !plainKey(k, PlainHeaderFields) || len(v) > 2000 {
			return errors.New("plain_draft_invalid")
		}
	}
	seen := map[string]bool{}
	for i := range p.Rows {
		row := &p.Rows[i]
		raw := row.Raw
		id := row.ID
		if id == "" || len(id) > 100 || seen[id] || len(raw) > len(PlainFields) {
			return errors.New("plain_draft_invalid")
		}
		seen[id] = true
		for k, v := range raw {
			if !plainKey(k, PlainFields) || len(v) > 4000 {
				return errors.New("plain_draft_invalid")
			}
		}
		*row = PlainRow{ID: id, Raw: raw, Issues: []string{}}
		for _, f := range []struct {
			k   string
			dst **LocalDate
		}{{"operationDate", &row.OperationDate}, {"bookingDate", &row.BookingDate}} {
			if absent(raw[f.k]) {
				if f.k == "operationDate" {
					row.Issues = append(row.Issues, "operation_date_required")
				}
				continue
			}
			d, e := ParseLocalDate(raw[f.k], "")
			if e != nil {
				row.Issues = append(row.Issues, f.k+":invalid_date")
			} else {
				*f.dst = d
			}
		}
		for _, f := range []struct {
			k, c string
			dst  **Money
		}{{"transactionAmount", "transactionCurrency", &row.TransactionAmount}, {"accountAmount", "accountCurrency", &row.AccountAmount}} {
			if absent(raw[f.k]) && absent(raw[f.c]) && f.k == "transactionAmount" {
				continue
			}
			m, e := signedMoney(raw[f.k], raw[f.c])
			if e != nil {
				row.Issues = append(row.Issues, f.k+":invalid_money")
			} else {
				*f.dst = m
			}
		}
		if row.AccountAmount != nil && (row.AccountAmount.AmountMinor == 0 || row.AccountAmount.Currency != "RUB") {
			row.Issues = append(row.Issues, "account_amount_required")
		}
		// Unsigned statements need a human to supply the printed sign, never an inferred debit.
		if row.AccountAmount != nil && !strings.HasPrefix(strings.TrimSpace(raw["accountAmount"]), "+") && !strings.HasPrefix(strings.TrimSpace(raw["accountAmount"]), "-") && !strings.HasPrefix(strings.TrimSpace(raw["accountAmount"]), "−") {
			row.Issues = append(row.Issues, "direction_required")
		}
		if !absent(raw["description"]) {
			v := raw["description"]
			row.Description = &v
		} else {
			row.Issues = append(row.Issues, "description_required")
		}
		if !absent(raw["cardLast4"]) {
			v := strings.TrimSpace(raw["cardLast4"])
			row.CardLast4 = &v
		}
	}
	return nil
}
func plainKey(k string, keys []string) bool {
	for _, s := range keys {
		if k == s {
			return true
		}
	}
	return false
}
func (p PlainReading) Review() (Review, error) {
	if e := p.Normalize(); e != nil {
		return Review{}, e
	}
	headerIssues := []string{}
	out := Review{Rows: []Row{}, Header: Header{Evidence: []Evidence{}}}
	for i, r := range p.Rows {
		row := Row{Ordinal: i + 1, OperationDate: r.OperationDate, BookingDate: r.BookingDate, Issues: r.Issues, Evidence: []Evidence{}}
		if r.Description != nil {
			row.Description = *r.Description
		}
		for _, f := range []struct {
			src *Money
			dst **Money
		}{{r.AccountAmount, &row.AccountAmount}, {r.TransactionAmount, &row.TransactionAmount}} {
			if f.src != nil {
				m := *f.src
				if m.AmountMinor < 0 {
					m.AmountMinor = -m.AmountMinor
				}
				*f.dst = &m
			}
		}
		if r.AccountAmount != nil {
			row.Direction = "credit"
			if r.AccountAmount.AmountMinor < 0 {
				row.Direction = "debit"
			}
		}
		out.Rows = append(out.Rows, row)
	}
	for _, f := range []struct {
		k   string
		dst **Money
	}{{"availableBalanceObserved", &out.Header.AvailableBalanceObserved}, {"creditTurnover", &out.Header.CreditTurnover}, {"debitTurnover", &out.Header.DebitTurnover}} {
		if absent(p.Header[f.k]) {
			continue
		}
		m, e := signedMoney(p.Header[f.k], p.Header["currency"])
		if e != nil {
			headerIssues = append(headerIssues, "plain_header_invalid")
			continue
		}
		if f.k != "availableBalanceObserved" && m.AmountMinor < 0 {
			headerIssues = append(headerIssues, "negative_turnover")
			continue
		}
		*f.dst = m
	}
	out.Reconciliation = Reconcile(out.Rows, out.Header)
	out.Reconciliation.Blockers = append(out.Reconciliation.Blockers, headerIssues...)
	if p.ParseStatus == "parse_error" {
		out.Reconciliation.Blockers = append(out.Reconciliation.Blockers, "parse_error")
	}
	return out, nil
}

// Split only unescaped Markdown pipes; never merge ambiguous rows or values.
func markdownCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	line = line[1 : len(line)-1]
	out := []string{}
	var cell strings.Builder
	escaped := false
	for _, r := range line {
		if escaped {
			if r != '|' && r != '\\' {
				cell.WriteRune('\\')
			}
			cell.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '|' {
			out = append(out, strings.TrimSpace(cell.String()))
			cell.Reset()
		} else {
			cell.WriteRune(r)
		}
	}
	if escaped {
		cell.WriteRune('\\')
	}
	return append(out, strings.TrimSpace(cell.String()))
}

var plainDateRE = regexp.MustCompile(`\b(?:[0-9]{2}\.[0-9]{2}\.[0-9]{4}|[0-9]{4}-[0-9]{2}-[0-9]{2})\b`)
var plainCurrencySuffix = regexp.MustCompile(`(?:^|\s)([A-Z]{3})$`)
var plainMoneyRE = regexp.MustCompile(`[+−-]?[0-9][0-9 \x{00a0}\x{202f}]*[.,][0-9]{2}\s*(?:₽|RUB|руб\.?)`)

func ParsePlain(raw string) PlainReading {
	p := PlainEmpty()
	p.RawAnswer = raw
	p.ParseStatus = "parse_error"
	if len(raw) > 256<<10 {
		return p
	}
	table := false
	bad := false
	columns := 0
	for _, line := range strings.Split(raw, "\n") {
		cells := markdownCells(line)
		if cells == nil && table && strings.Contains(line, "|") {
			bad = true
		}
		if cells != nil {
			lower := strings.ToLower(strings.Join(cells, " "))
			if !table {
				if strings.Contains(lower, "дата") && strings.Contains(lower, "описан") {
					columns = len(cells)
					header := cells
					if len(header) == 7 || len(header) == 9 {
						header = header[1:]
					}
					if !plainHeaderOrder(header) {
						bad = true
					}
					if columns != 6 && columns != 7 && columns != 8 && columns != 9 {
						bad = true
					}
					table = true
				}
				continue
			}
			if strings.Trim(strings.Join(cells, ""), " :-") == "" {
				continue
			}
			if len(cells) != columns || len(p.Rows) >= 200 {
				bad = true
				continue
			}
			if columns == 9 || columns == 7 {
				cells = cells[1:]
			}
			fields := map[string]string{}
			if len(cells) == 6 {
				fields["operationDate"], fields["bookingDate"] = cells[0], cells[1]
				for _, v := range []struct {
					idx  int
					k, c string
				}{{2, "transactionAmount", "transactionCurrency"}, {3, "accountAmount", "accountCurrency"}} {
					fields[v.k] = cells[v.idx]
					if m := plainCurrencySuffix.FindStringSubmatch(strings.TrimSpace(cells[v.idx])); len(m) == 2 {
						fields[v.c] = m[1]
					}
					for _, cc := range []string{"RUB", "₽", "руб.", "руб"} {
						if strings.HasSuffix(strings.TrimSpace(cells[v.idx]), cc) {
							fields[v.c] = "RUB"
							break
						}
					}
				}
				fields["description"], fields["cardLast4"] = cells[4], cells[5]
			} else if len(cells) == 8 {
				for i, k := range PlainFields {
					fields[k] = cells[i]
				}
			} else {
				bad = true
				continue
			}
			p.Rows = append(p.Rows, PlainRow{ID: fmt.Sprintf("row-%d", len(p.Rows)+1), Raw: fields})
			continue
		}
		lower := strings.ToLower(strings.ReplaceAll(line, "*", ""))
		dates := plainDateRE.FindAllString(line, -1)
		money := plainMoneyRE.FindAllString(line, -1)
		if strings.Contains(lower, "период") && len(dates) == 2 {
			p.Header["periodFrom"], p.Header["periodTo"] = dates[0], dates[1]
		}
		key := ""
		switch {
		case strings.Contains(lower, "доступ") && strings.Contains(lower, "остат"):
			key = "availableBalanceObserved"
			if len(dates) == 1 {
				p.Header["availableBalanceDate"] = dates[0]
			}
		case strings.Contains(lower, "пополн"):
			key = "creditTurnover"
		case strings.Contains(lower, "расход") || strings.Contains(lower, "списан"):
			key = "debitTurnover"
		}
		if key != "" && len(money) == 1 {
			p.Header[key] = money[0]
			p.Header["currency"] = "RUB"
		}
	}
	if table && !bad && len(p.Rows) > 0 {
		p.ParseStatus = "parsed"
	}
	if p.Normalize() != nil {
		p.ParseStatus = "parse_error"
		p.Rows = []PlainRow{}
		p.Header = map[string]string{}
	}
	return p
}

// Unsupported layouts remain explicit parse errors instead of silently swapping dates/amounts.
func plainHeaderOrder(cells []string) bool {
	if len(cells) != 6 && len(cells) != 8 {
		return false
	}
	has := func(i int, parts ...string) bool {
		v := strings.ToLower(cells[i])
		for _, p := range parts {
			if strings.Contains(v, p) {
				return true
			}
		}
		return false
	}
	if !has(0, "дат") || !has(0, "операц") || !has(1, "дат") || !has(1, "списан", "провод", "обработ", "зачисл", "учёт", "учет") || !has(2, "сумм") || !has(2, "операц") {
		return false
	}
	if len(cells) == 6 {
		return has(3, "сумм") && has(3, "карт", "счёт", "счет") && has(4, "описан") && has(5, "карт")
	}
	return has(3, "валют") && has(4, "сумм") && has(4, "карт", "счёт", "счет") && has(5, "валют") && has(6, "описан") && has(7, "карт")
}
