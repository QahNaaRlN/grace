package obligations

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func ptr(value int64) *int64 { return &value }

func TestPreviewEqualKeepsAnchorDayAcrossShortMonths(t *testing.T) {
	preview, err := PreviewSchedule(previewRequest{Mode: "equal", FirstDueOn: "2027-12-31", Count: 4, Interval: "monthly", AmountMinor: 100})
	if err != nil {
		t.Fatal(err)
	}
	var dates []string
	for _, payment := range preview.Payments {
		dates = append(dates, payment.DueOn)
	}
	if got := strings.Join(dates, ","); got != "2027-12-31,2028-01-31,2028-02-29,2028-03-31" {
		t.Fatalf("unexpected dates %s", got)
	}
	if preview.TotalMinor != 400 || preview.FirstDueOn != "2027-12-31" || preview.LastDueOn != "2028-03-31" {
		t.Fatalf("unexpected summary %#v", preview)
	}
}

func TestPreviewEqualMortgage(t *testing.T) {
	preview, err := PreviewSchedule(previewRequest{
		Mode: "equal", FirstDueOn: "2027-01-15", Count: 120, Interval: "monthly",
		AmountMinor: 2_500_000, FirstAmountMinor: ptr(1_200_000), LastAmountMinor: ptr(2_412_345),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Payments) != 120 || preview.LastDueOn != "2036-12-15" {
		t.Fatalf("unexpected schedule %d payments, last %s", len(preview.Payments), preview.LastDueOn)
	}
	if preview.Payments[0].AmountMinor != 1_200_000 || preview.Payments[119].AmountMinor != 2_412_345 {
		t.Fatalf("first/last amounts not applied: %#v %#v", preview.Payments[0], preview.Payments[119])
	}
	if want := int64(1_200_000 + 118*2_500_000 + 2_412_345); preview.TotalMinor != want {
		t.Fatalf("total %d, want %d", preview.TotalMinor, want)
	}
	created := createRequest{Name: "Ипотека", Kind: "loan", Currency: "RUB", TotalMinor: preview.TotalMinor, Payments: preview.Payments}
	if !normalizeAndValidateCreate(&created) {
		t.Fatal("preview must satisfy the create contract")
	}
}

func TestPreviewEqualRejectsInvalidRequests(t *testing.T) {
	tests := map[string]previewRequest{
		"no payments":     {Mode: "equal", FirstDueOn: "2027-01-15", Count: 0, Interval: "monthly", AmountMinor: 1},
		"over the limit":  {Mode: "equal", FirstDueOn: "2027-01-15", Count: 601, Interval: "monthly", AmountMinor: 1},
		"weekly":          {Mode: "equal", FirstDueOn: "2027-01-15", Count: 2, Interval: "weekly", AmountMinor: 1},
		"bad date":        {Mode: "equal", FirstDueOn: "2027-02-30", Count: 2, Interval: "monthly", AmountMinor: 1},
		"zero amount":     {Mode: "equal", FirstDueOn: "2027-01-15", Count: 2, Interval: "monthly"},
		"negative last":   {Mode: "equal", FirstDueOn: "2027-01-15", Count: 2, Interval: "monthly", AmountMinor: 1, LastAmountMinor: ptr(-1)},
		"one, two values": {Mode: "equal", FirstDueOn: "2027-01-15", Count: 1, Interval: "monthly", AmountMinor: 1, FirstAmountMinor: ptr(2), LastAmountMinor: ptr(3)},
		"overflow":        {Mode: "equal", FirstDueOn: "2027-01-15", Count: 3, Interval: "monthly", AmountMinor: math.MaxInt64 / 2},
		"unknown mode":    {Mode: "annuity"},
	}
	for name, request := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := PreviewSchedule(request); !errors.Is(err, ErrPreviewInvalid) {
				t.Fatalf("expected ErrPreviewInvalid, got %v", err)
			}
		})
	}
}

// A bank schedule copied from a spreadsheet: № | дата | платёж | долг | проценты | остаток,
// with a header row and a closing «Итого».
func bankTable(rows int) string {
	var builder strings.Builder
	builder.WriteString("№\tДата платежа\tСумма платежа\tОсновной долг\tПроценты\tОстаток\n")
	start := time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC)
	for index := 0; index < rows; index++ {
		date := start.AddDate(0, index, 0).Format("02.01.2006")
		fmt.Fprintf(&builder, "%d\t%s\t25 000,00\t18 000,00\t7 000,00\t%d 000,00\r\n", index+1, date, 5000-index)
	}
	builder.WriteString("Итого\t\t6 000 000,00\t4 320 000,00\t1 680 000,00\t\n")
	return builder.String()
}

func TestPreviewPasteBankTable(t *testing.T) {
	preview, err := PreviewSchedule(previewRequest{Mode: "paste", Text: bankTable(240)})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Payments) != 240 || len(preview.RejectedLines) != 2 {
		t.Fatalf("got %d payments and %d rejected lines: %#v", len(preview.Payments), len(preview.RejectedLines), preview.RejectedLines)
	}
	if preview.Payments[0].AmountMinor != 2_500_000 || preview.Payments[0].DueOn != "2027-01-15" {
		t.Fatalf("payment column must be the first amount after the date, got %#v", preview.Payments[0])
	}
	if preview.TotalMinor != 240*2_500_000 {
		t.Fatalf("unexpected total %d", preview.TotalMinor)
	}
	if preview.RejectedLines[0].Line != 1 || preview.RejectedLines[0].Reason != "нет даты платежа" ||
		preview.RejectedLines[1].Line != 242 || !strings.HasPrefix(preview.RejectedLines[1].Text, "Итого") {
		t.Fatalf("header and total must be reported, got %#v", preview.RejectedLines)
	}
}

func TestPreviewPasteFormats(t *testing.T) {
	text := strings.Join([]string{
		"15.01.2027 25 000,00 ₽ 18 000,00",
		"2027-02-15;24999.5;18000",
		"15.03.2027  25 000 руб.",
		"31.02.2027 1 000,00",
		"15.03.2027 1 000,00",
		"15.04.2027 платёж",
		"15.05.2027 0,00",
	}, "\n")
	preview, err := PreviewSchedule(previewRequest{Mode: "paste", Text: text})
	if err != nil {
		t.Fatal(err)
	}
	want := []createPaymentRequest{{DueOn: "2027-01-15", AmountMinor: 2_500_000}, {DueOn: "2027-02-15", AmountMinor: 2_499_950}, {DueOn: "2027-03-15", AmountMinor: 2_500_000}}
	if fmt.Sprint(preview.Payments) != fmt.Sprint(want) {
		t.Fatalf("payments %v, want %v", preview.Payments, want)
	}
	var reasons []string
	for _, line := range preview.RejectedLines {
		reasons = append(reasons, fmt.Sprintf("%d:%s", line.Line, line.Reason))
	}
	if got := strings.Join(reasons, "|"); got != "4:некорректная дата|5:дата не позже предыдущего платежа|6:нет суммы после даты|7:сумма должна быть больше нуля" {
		t.Fatalf("unexpected rejections %s", got)
	}
}

func TestPreviewPasteChosenColumnAndLimits(t *testing.T) {
	preview, err := PreviewSchedule(previewRequest{Mode: "paste", Text: "15.01.2027\t25 000,00\t18 000,00", AmountColumn: 2})
	if err != nil || preview.Payments[0].AmountMinor != 1_800_000 {
		t.Fatalf("expected the second amount column, got %#v %v", preview, err)
	}
	preview, err = PreviewSchedule(previewRequest{Mode: "paste", Text: "15.01.2027\t25 000,00", AmountColumn: 3})
	if err != nil || len(preview.Payments) != 0 || preview.RejectedLines[0].Reason != "нет денежной колонки 3 после даты" {
		t.Fatalf("expected a missing column rejection, got %#v %v", preview, err)
	}
	for name, request := range map[string]previewRequest{
		"empty":     {Mode: "paste", Text: "  \n "},
		"too large": {Mode: "paste", Text: strings.Repeat("x", maxPasteBytes+1)},
		"column":    {Mode: "paste", Text: "15.01.2027 1", AmountColumn: 13},
	} {
		if _, err := PreviewSchedule(request); !errors.Is(err, ErrPreviewInvalid) {
			t.Fatalf("%s: expected ErrPreviewInvalid, got %v", name, err)
		}
	}
	preview, err = PreviewSchedule(previewRequest{Mode: "paste", Text: bankTable(601)})
	if err != nil || len(preview.Payments) != maxSchedulePayments || preview.RejectedLines[1].Reason != "больше 600 платежей" {
		t.Fatalf("expected the 601st payment to be rejected, got %d payments %v", len(preview.Payments), err)
	}
}

func TestPreviewRowsSummarizesEditedSchedule(t *testing.T) {
	preview, err := PreviewSchedule(previewRequest{Mode: "rows", Payments: []createPaymentRequest{
		{DueOn: "2027-01-15", AmountMinor: 100},
		{DueOn: "2027-02-15", AmountMinor: 250},
		{DueOn: "2027-02-01", AmountMinor: 100},
		{DueOn: "", AmountMinor: 100},
		{DueOn: "2027-03-15", AmountMinor: 0},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if preview.TotalMinor != 350 || len(preview.Payments) != 2 || len(preview.RejectedLines) != 3 {
		t.Fatalf("unexpected preview %#v", preview)
	}
	if _, err := PreviewSchedule(previewRequest{Mode: "rows"}); !errors.Is(err, ErrPreviewInvalid) {
		t.Fatal("empty rows must be rejected")
	}
}

func TestPreviewScheduleHandler(t *testing.T) {
	handler := NewHandler(nil)
	tests := []struct {
		body   string
		status int
		has    string
	}{
		{`{"mode":"equal","firstDueOn":"2027-01-15","count":2,"interval":"monthly","amountMinor":100}`, http.StatusOK, `"totalMinor":200`},
		{`{"mode":"equal","firstDueOn":"2027-01-15","count":0,"interval":"monthly","amountMinor":100}`, http.StatusBadRequest, "количество платежей"},
		{`{"mode":"paste","text":"Дата\tСумма\nИтого\t100"}`, http.StatusUnprocessableEntity, "no_payments"},
		{`{"mode":"paste","text":"15.01.2027 1","unknown":1}`, http.StatusBadRequest, "invalid_request"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		handler.PreviewSchedule(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/obligations/schedule-preview", strings.NewReader(test.body)))
		if recorder.Code != test.status || !strings.Contains(recorder.Body.String(), test.has) {
			t.Fatalf("%s: got %d %s", test.body, recorder.Code, recorder.Body.String())
		}
	}
}
