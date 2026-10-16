package obligations

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/calendar"
)

// SPEC-0037: a schedule preview turns a pattern, pasted bank table or edited
// rows into explicit payments and their total. It never writes: the person
// reviews the result and saves it through the regular create/draft APIs.

const (
	maxSchedulePayments = 600
	maxPasteBytes       = 64 * 1024
)

var ErrPreviewInvalid = errors.New("invalid schedule preview request")

type previewRequest struct {
	Mode             string `json:"mode"`
	FirstDueOn       string `json:"firstDueOn,omitempty"`
	Count            int    `json:"count,omitempty"`
	Interval         string `json:"interval,omitempty"`
	AmountMinor      int64  `json:"amountMinor,omitempty"`
	FirstAmountMinor *int64 `json:"firstAmountMinor,omitempty"`
	LastAmountMinor  *int64 `json:"lastAmountMinor,omitempty"`
	Text             string `json:"text,omitempty"`
	AmountColumn     int    `json:"amountColumn,omitempty"`
	// SPEC-0053: money columns of a bank loan schedule; banks order them differently.
	PrincipalColumn int                    `json:"principalColumn,omitempty"`
	InterestColumn  int                    `json:"interestColumn,omitempty"`
	OtherColumn     int                    `json:"otherColumn,omitempty"`
	Payments        []createPaymentRequest `json:"payments,omitempty"`
	DayOfMonth      int                    `json:"dayOfMonth,omitempty"`
	FromMonth       string                 `json:"fromMonth,omitempty"`
	ToMonth         string                 `json:"toMonth,omitempty"`
}

type RejectedLine struct {
	Line   int    `json:"line"`
	Text   string `json:"text"`
	Reason string `json:"reason"`
}

type SchedulePreview struct {
	Payments      []createPaymentRequest `json:"payments"`
	TotalMinor    int64                  `json:"totalMinor"`
	FirstDueOn    string                 `json:"firstDueOn,omitempty"`
	LastDueOn     string                 `json:"lastDueOn,omitempty"`
	RejectedLines []RejectedLine         `json:"rejectedLines"`
}

// PreviewSchedule validates the request shape and builds the preview.
// ErrPreviewInvalid wraps a person-readable reason for a malformed request.
func PreviewSchedule(request previewRequest) (SchedulePreview, error) {
	switch request.Mode {
	case "equal":
		return previewEqual(request)
	case "period":
		return previewPeriod(request)
	case "paste":
		return previewPaste(request.Text, pasteColumns{request.AmountColumn, request.PrincipalColumn, request.InterestColumn, request.OtherColumn})
	case "rows":
		return previewRows(request.Payments)
	default:
		return SchedulePreview{}, fmt.Errorf("%w: неизвестный способ заполнения графика", ErrPreviewInvalid)
	}
}

func previewEqual(request previewRequest) (SchedulePreview, error) {
	first, err := time.Parse(dateLayout, strings.TrimSpace(request.FirstDueOn))
	if err != nil || first.Format(dateLayout) != strings.TrimSpace(request.FirstDueOn) {
		return SchedulePreview{}, fmt.Errorf("%w: укажите дату первого платежа", ErrPreviewInvalid)
	}
	if request.Count < 1 || request.Count > maxSchedulePayments {
		return SchedulePreview{}, fmt.Errorf("%w: количество платежей — от 1 до %d", ErrPreviewInvalid, maxSchedulePayments)
	}
	if request.Interval != "monthly" {
		return SchedulePreview{}, fmt.Errorf("%w: поддерживаются только ежемесячные платежи", ErrPreviewInvalid)
	}
	dates := make([]time.Time, request.Count)
	for index := range dates {
		dates[index] = calendar.Occurrence(first, "monthly", index)
	}
	return equalPayments(dates, request)
}

// SPEC-0045: one payment per month on a day of month, from a month to a month
// inclusive. Each month clamps the day on its own (31 → 28/29 Feb → 31 Mar),
// so day 31 is "the last day of every month".
func previewPeriod(request previewRequest) (SchedulePreview, error) {
	from, errFrom := time.Parse("2006-01", strings.TrimSpace(request.FromMonth))
	to, errTo := time.Parse("2006-01", strings.TrimSpace(request.ToMonth))
	if errFrom != nil || errTo != nil {
		return SchedulePreview{}, fmt.Errorf("%w: укажите месяц начала и месяц окончания", ErrPreviewInvalid)
	}
	if request.DayOfMonth < 1 || request.DayOfMonth > 31 {
		return SchedulePreview{}, fmt.Errorf("%w: число месяца — от 1 до 31", ErrPreviewInvalid)
	}
	if to.Before(from) {
		return SchedulePreview{}, fmt.Errorf("%w: срок окончания раньше начала", ErrPreviewInvalid)
	}
	months := (to.Year()-from.Year())*12 + int(to.Month()-from.Month()) + 1
	if months > maxSchedulePayments {
		return SchedulePreview{}, fmt.Errorf("%w: не больше %d месяцев", ErrPreviewInvalid, maxSchedulePayments)
	}
	dates := make([]time.Time, months)
	for index := range dates {
		month := from.AddDate(0, index, 0)
		last := month.AddDate(0, 1, -1).Day()
		day := min(request.DayOfMonth, last)
		dates[index] = month.AddDate(0, 0, day-1)
	}
	return equalPayments(dates, request)
}

// equalPayments puts the same amount on every date, with optional first and
// last amounts, and validates the amounts for both pattern modes.
func equalPayments(dates []time.Time, request previewRequest) (SchedulePreview, error) {
	if request.AmountMinor <= 0 ||
		(request.FirstAmountMinor != nil && *request.FirstAmountMinor <= 0) ||
		(request.LastAmountMinor != nil && *request.LastAmountMinor <= 0) {
		return SchedulePreview{}, fmt.Errorf("%w: суммы платежей должны быть больше нуля", ErrPreviewInvalid)
	}
	if len(dates) == 1 && request.FirstAmountMinor != nil && request.LastAmountMinor != nil &&
		*request.FirstAmountMinor != *request.LastAmountMinor {
		return SchedulePreview{}, fmt.Errorf("%w: при одном платеже первая и последняя суммы совпадают", ErrPreviewInvalid)
	}
	payments := make([]createPaymentRequest, len(dates))
	for index, date := range dates {
		amount := request.AmountMinor
		if index == 0 && request.FirstAmountMinor != nil {
			amount = *request.FirstAmountMinor
		}
		if index == len(dates)-1 && request.LastAmountMinor != nil {
			amount = *request.LastAmountMinor
		}
		payments[index] = createPaymentRequest{DueOn: date.Format(dateLayout), AmountMinor: amount}
	}
	return summarize(payments, nil)
}

var (
	dottedDate = regexp.MustCompile(`(?:^|[^\d])(\d{1,2})\.(\d{1,2})\.(\d{4})(?:[^\d]|$)`)
	isoDate    = regexp.MustCompile(`(?:^|[^\d])(\d{4})-(\d{2})-(\d{2})(?:[^\d]|$)`)
	amountCell = regexp.MustCompile(`^-?\d+(?:[.,]\d{1,2})?$`)
	// Without tabs or semicolons, amounts are found by grouping: "25 000,00".
	spacedAmount = regexp.MustCompile(`-?\d{1,3}(?:[ \x{00a0}\x{202f}]\d{3})+(?:[.,]\d{1,2})?|-?\d+(?:[.,]\d{1,2})?`)
	currencyMark = regexp.MustCompile(`(?i)\s*(₽|руб\.?|rub)\s*`)
)

// pasteColumns are 1-based money columns after the date; 0 means "not in the table".
type pasteColumns struct{ amount, principal, interest, other int }

func previewPaste(text string, columns pasteColumns) (SchedulePreview, error) {
	amountColumn := columns.amount
	if strings.TrimSpace(text) == "" {
		return SchedulePreview{}, fmt.Errorf("%w: вставьте строки графика", ErrPreviewInvalid)
	}
	if len(text) > maxPasteBytes {
		return SchedulePreview{}, fmt.Errorf("%w: текст больше 64 КиБ — вставьте график частями", ErrPreviewInvalid)
	}
	if amountColumn < 0 || amountColumn > 12 {
		return SchedulePreview{}, fmt.Errorf("%w: номер колонки суммы — от 1 до 12", ErrPreviewInvalid)
	}
	for _, column := range []int{columns.principal, columns.interest, columns.other} {
		if column < 0 || column > 12 {
			return SchedulePreview{}, fmt.Errorf("%w: номер колонки — от 1 до 12", ErrPreviewInvalid)
		}
	}
	if (columns.principal == 0) != (columns.interest == 0) {
		return SchedulePreview{}, fmt.Errorf("%w: укажите обе колонки — основной долг и проценты", ErrPreviewInvalid)
	}
	if amountColumn == 0 {
		amountColumn = 1
	}
	var payments []createPaymentRequest
	var rejected []RejectedLine
	var previous time.Time
	for index, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" {
			continue
		}
		reject := func(reason string) {
			shown := line
			if len([]rune(shown)) > 120 {
				shown = string([]rune(shown)[:120]) + "…"
			}
			rejected = append(rejected, RejectedLine{Line: index + 1, Text: shown, Reason: reason})
		}
		due, rest, ok := lineDate(line)
		if !ok {
			reject("нет даты платежа")
			continue
		}
		if due.IsZero() {
			reject("некорректная дата")
			continue
		}
		amounts := lineAmounts(rest)
		if len(amounts) == 0 {
			reject("нет суммы после даты")
			continue
		}
		if amountColumn > len(amounts) {
			reject(fmt.Sprintf("нет денежной колонки %d после даты", amountColumn))
			continue
		}
		amount := amounts[amountColumn-1]
		payment := createPaymentRequest{DueOn: due.Format(dateLayout), AmountMinor: amount}
		partsReason := ""
		if columns.principal > 0 {
			payment.PrincipalMinor, payment.InterestMinor, payment.OtherMinor, partsReason = lineParts(amounts, columns, amount)
		}
		switch {
		case amount <= 0:
			reject("сумма должна быть больше нуля")
		case partsReason != "":
			reject(partsReason)
		case !previous.IsZero() && !due.After(previous):
			reject("дата не позже предыдущего платежа")
		case len(payments) == maxSchedulePayments:
			reject(fmt.Sprintf("больше %d платежей", maxSchedulePayments))
		default:
			payments = append(payments, payment)
			previous = due
		}
	}
	return summarize(payments, rejected)
}

// lineParts reads principal, interest and other charges of a pasted row. Without
// an "other" column, other charges are the payment minus principal and interest.
func lineParts(amounts []int64, columns pasteColumns, amount int64) (*int64, *int64, *int64, string) {
	cell := func(column int) (int64, bool) {
		if column > len(amounts) {
			return 0, false
		}
		return amounts[column-1], true
	}
	principal, ok1 := cell(columns.principal)
	interest, ok2 := cell(columns.interest)
	if !ok1 || !ok2 {
		return nil, nil, nil, "нет колонок основного долга или процентов"
	}
	other := amount - principal - interest
	if columns.other > 0 {
		value, ok := cell(columns.other)
		if !ok {
			return nil, nil, nil, "нет колонки прочих платежей"
		}
		other = value
	}
	if !validParts(&principal, &interest, &other, amount) {
		return nil, nil, nil, "основной долг, проценты и прочее не равны платежу"
	}
	return &principal, &interest, &other, ""
}

// lineDate finds the first date in a line and returns the text after it.
// A zero time with ok=true means a date-shaped value that is not a real date.
func lineDate(line string) (time.Time, string, bool) {
	if match := dottedDate.FindStringSubmatchIndex(line); match != nil {
		day, month, year := line[match[2]:match[3]], line[match[4]:match[5]], line[match[6]:match[7]]
		return realDate(year, month, day), line[match[7]:], true
	}
	if match := isoDate.FindStringSubmatchIndex(line); match != nil {
		year, month, day := line[match[2]:match[3]], line[match[4]:match[5]], line[match[6]:match[7]]
		return realDate(year, month, day), line[match[7]:], true
	}
	return time.Time{}, "", false
}

func realDate(year, month, day string) time.Time {
	value := fmt.Sprintf("%s-%02s-%02s", year, month, day)
	value = strings.ReplaceAll(value, " ", "0")
	parsed, err := time.Parse(dateLayout, value)
	if err != nil || parsed.Format(dateLayout) != value {
		return time.Time{}
	}
	return parsed
}

// lineAmounts returns money values after the date, in column order. Tabs and
// semicolons (spreadsheet and CSV copies) delimit cells exactly; otherwise
// amounts are recognised by thousands grouping.
func lineAmounts(rest string) []int64 {
	rest = currencyMark.ReplaceAllString(rest, " ")
	var cells []string
	if strings.ContainsAny(rest, "\t;") {
		cells = strings.FieldsFunc(rest, func(r rune) bool { return r == '\t' || r == ';' })
	} else {
		cells = spacedAmount.FindAllString(rest, -1)
	}
	var amounts []int64
	for _, cell := range cells {
		compact := strings.Map(func(r rune) rune {
			if r == ' ' || r == ' ' || r == ' ' {
				return -1
			}
			return r
		}, strings.TrimSpace(cell))
		if compact == "" {
			continue
		}
		if value, ok := parseMinorText(compact); ok {
			amounts = append(amounts, value)
		}
	}
	return amounts
}

func parseMinorText(text string) (int64, bool) {
	if !amountCell.MatchString(text) {
		return 0, false
	}
	negative := strings.HasPrefix(text, "-")
	text = strings.TrimPrefix(text, "-")
	whole, fraction, _ := strings.Cut(strings.Replace(text, ",", ".", 1), ".")
	value, ok := new(big.Int).SetString(whole+(fraction + "00")[:2], 10)
	if !ok || !value.IsInt64() {
		return 0, false
	}
	if negative {
		return -value.Int64(), true
	}
	return value.Int64(), true
}

func previewRows(rows []createPaymentRequest) (SchedulePreview, error) {
	if len(rows) == 0 || len(rows) > maxSchedulePayments {
		return SchedulePreview{}, fmt.Errorf("%w: в графике от 1 до %d платежей", ErrPreviewInvalid, maxSchedulePayments)
	}
	var payments []createPaymentRequest
	var rejected []RejectedLine
	var previous time.Time
	for index, row := range rows {
		due, err := time.Parse(dateLayout, strings.TrimSpace(row.DueOn))
		text := fmt.Sprintf("%s — %d", row.DueOn, row.AmountMinor)
		switch {
		case err != nil || due.Format(dateLayout) != strings.TrimSpace(row.DueOn):
			rejected = append(rejected, RejectedLine{Line: index + 1, Text: text, Reason: "некорректная дата"})
		case row.AmountMinor <= 0:
			rejected = append(rejected, RejectedLine{Line: index + 1, Text: text, Reason: "сумма должна быть больше нуля"})
		case !previous.IsZero() && !due.After(previous):
			rejected = append(rejected, RejectedLine{Line: index + 1, Text: text, Reason: "дата не позже предыдущего платежа"})
		case (row.PrincipalMinor != nil || row.InterestMinor != nil || row.OtherMinor != nil) &&
			!validParts(row.PrincipalMinor, row.InterestMinor, row.OtherMinor, row.AmountMinor):
			rejected = append(rejected, RejectedLine{Line: index + 1, Text: text, Reason: "основной долг, проценты и прочее не равны платежу"})
		default:
			payments = append(payments, createPaymentRequest{DueOn: due.Format(dateLayout), AmountMinor: row.AmountMinor,
				PrincipalMinor: row.PrincipalMinor, InterestMinor: row.InterestMinor, OtherMinor: row.OtherMinor})
			previous = due
		}
	}
	return summarize(payments, rejected)
}

func summarize(payments []createPaymentRequest, rejected []RejectedLine) (SchedulePreview, error) {
	if rejected == nil {
		rejected = []RejectedLine{}
	}
	preview := SchedulePreview{Payments: payments, RejectedLines: rejected}
	if preview.Payments == nil {
		preview.Payments = []createPaymentRequest{}
	}
	for _, payment := range payments {
		if preview.TotalMinor > math.MaxInt64-payment.AmountMinor {
			return SchedulePreview{}, fmt.Errorf("%w: сумма графика слишком большая", ErrPreviewInvalid)
		}
		preview.TotalMinor += payment.AmountMinor
	}
	if len(payments) > 0 {
		preview.FirstDueOn = payments[0].DueOn
		preview.LastDueOn = payments[len(payments)-1].DueOn
	}
	return preview, nil
}
