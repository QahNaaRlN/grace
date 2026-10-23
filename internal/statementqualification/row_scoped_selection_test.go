package statementqualification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type rowScopedMeter struct {
	reserves, settles int
	reserved          int64
	actual            *int64
	fail              string
}

func (m *rowScopedMeter) Reserve(ctx context.Context, n int64, p statementextract.Policy) (string, error) {
	m.reserves++
	m.reserved = n
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 390*time.Second {
		return "", errors.New("document deadline missing or expanded")
	}
	if m.fail == "reserve" {
		return "", errors.New("PRIVATE_RESERVE")
	}
	return "fake-reservation", nil
}
func (m *rowScopedMeter) Settle(ctx context.Context, _ string, n *int64) error {
	m.settles++
	if ctx.Err() != nil {
		return errors.New("cancelled settlement context")
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 5*time.Second {
		return errors.New("settlement deadline missing or expanded")
	}
	if n != nil {
		v := *n
		m.actual = &v
	}
	if m.fail == "settle" {
		return errors.New("PRIVATE_SETTLE")
	}
	return nil
}

type rowScopedFake struct {
	t                             *testing.T
	meter                         *rowScopedMeter
	calls                         int
	groupFault, fieldFault, fault string
	failAt                        int
	orders                        []string
	requests                      []statementextract.RowScopedRequest
	cancel                        context.CancelFunc
}

func rowScopedPayload(t *testing.T, v any) []byte {
	t.Helper()
	raw, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var x any
	_ = json.Unmarshal(raw, &x)
	var normalize func(any)
	normalize = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				if v == nil {
					if k == "labels" {
						x[k] = map[string]any{}
					} else {
						x[k] = []any{}
					}
				} else {
					normalize(v)
				}
			}
		case []any:
			for _, v := range x {
				normalize(v)
			}
		}
	}
	normalize(x)
	if o, ok := x.(map[string]any); ok {
		if h, ok := o["header"].(map[string]any); ok {
			labels := h["labels"].(map[string]any)
			for _, k := range []string{"openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover"} {
				if _, ok := labels[k]; !ok {
					labels[k] = []any{}
				}
			}
		}
	}
	raw, e = json.Marshal(x)
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func rowScopedDelivered(t *testing.T, r statementextract.RowScopedRequest) (statementextract.Chunk, map[string]bool) {
	t.Helper()
	var body struct {
		Index          int
		Context, Spans [][]json.RawMessage
	}
	if len(r.Messages) != 2 || json.Unmarshal([]byte(r.Messages[1]["content"]), &body) != nil {
		t.Fatal("actual wire body")
	}
	decode := func(xs [][]json.RawMessage) []statementextract.RedactedSpan {
		out := []statementextract.RedactedSpan{}
		for _, tuple := range xs {
			if len(tuple) != 5 {
				t.Fatal("tuple")
			}
			var s statementextract.RedactedSpan
			for i, v := range []any{&s.ID, &s.Page, &s.X, &s.Y, &s.Text} {
				if json.Unmarshal(tuple[i], v) != nil {
					t.Fatal("tuple value")
				}
			}
			out = append(out, s)
		}
		return out
	}
	var schema map[string]any
	if json.Unmarshal(r.Schema, &schema) != nil {
		t.Fatal("actual schema")
	}
	defs := schema["$defs"].(map[string]any)
	money := defs["moneySelection"].(map[string]any)
	numeric := map[string]bool{}
	if ids, ok := money["items"].(map[string]any)["enum"].([]any); ok {
		for _, id := range ids {
			numeric[id.(string)] = true
		}
	}
	return statementextract.Chunk{Index: body.Index, Context: decode(body.Context), Spans: decode(body.Spans)}, numeric
}
func rowScopedFieldMapping(c statementextract.Chunk, numeric map[string]bool) statementdata.RowMapping {
	date := regexp.MustCompile(`^[0-9]{2}\.[0-9]{2}\.[0-9]{4}$`)
	clock := regexp.MustCompile(`^[0-9]{2}:[0-9]{2}(?::[0-9]{2})?$`)
	amount := regexp.MustCompile(`^[+−-]?[0-9]+[.,][0-9]{2}$`)
	dates, clocks, money := []string{}, []string{}, []string{}
	for _, s := range c.Spans {
		if date.MatchString(s.Text) {
			dates = append(dates, s.ID)
		}
		if clock.MatchString(s.Text) {
			clocks = append(clocks, s.ID)
		}
		if numeric[s.ID] && amount.MatchString(s.Text) {
			money = append(money, s.ID)
		}
	}
	currency := func(text string) []string {
		for _, s := range append(append([]statementextract.RedactedSpan{}, c.Context...), c.Spans...) {
			if s.Text == text {
				return []string{s.ID}
			}
		}
		return nil
	}
	r := statementdata.RowMapping{}
	if len(dates) > 0 {
		r.OperationDate = []string{dates[0]}
	}
	if len(dates) > 1 {
		r.BookingDate = []string{dates[1]}
	}
	if len(clocks) > 0 {
		r.OperationDate = append(r.OperationDate, clocks[0])
	}
	if len(clocks) > 1 {
		r.BookingDate = append(r.BookingDate, clocks[1])
	}
	if len(money) > 0 {
		r.AccountAmount = []string{money[0]}
		r.AccountCurrency = currency("RUB")
	}
	if len(money) > 1 {
		r.TransactionAmount = []string{money[1]}
		r.TransactionCurrency = currency("USD")
	}
	selected := map[string]bool{}
	for _, ids := range [][]string{r.OperationDate, r.BookingDate, r.AccountAmount, r.AccountCurrency, r.TransactionAmount, r.TransactionCurrency} {
		for _, id := range ids {
			selected[id] = true
		}
	}
	// The fake explicitly selects remaining owned description aliases. The server
	// never fills this selection. No previous grouping answer/golden is consulted.
	for _, s := range c.Spans {
		if !selected[s.ID] {
			r.Description = append(r.Description, s.ID)
		}
	}
	return r
}
func rowScopedRU(c statementextract.Chunk, numeric map[string]bool) statementdata.Proposal {
	find := func(text string) statementextract.RedactedSpan {
		for _, s := range c.Spans {
			if s.Text == text {
				return s
			}
		}
		return statementextract.RedactedSpan{}
	}
	line := func(a statementextract.RedactedSpan, delta int, onlyNumeric bool) []string {
		out := []string{}
		for _, s := range c.Spans {
			if s.Page == a.Page && math.Abs(float64(s.Y-a.Y-delta)) <= 1 && (!onlyNumeric || numeric[s.ID]) {
				out = append(out, s.ID)
			}
		}
		return out
	}
	p := statementdata.Proposal{Header: statementdata.HeaderMapping{Labels: map[string]statementdata.Selection{}}}
	for _, text := range []string{"12.10.2026", "13.10.2026"} {
		a := find(text)
		p.Rows = append(p.Rows, statementdata.RowMapping{OperationDate: []string{a.ID}, AccountAmount: line(a, 0, true), AccountCurrency: []string{find("RUB").ID}, Description: line(a, 25, false)})
	}
	p.Header.Currency = []string{find("RUB").ID}
	p.Header.AvailableBalanceObserved = line(find("Доступный"), 0, true)
	p.Header.Labels["availableBalanceObserved"] = []string{}
	a := find("Доступный")
	for _, span := range c.Spans {
		if span.Page == a.Page && span.Y == a.Y && (span.Text == "Доступный" || span.Text == "остаток") {
			p.Header.Labels["availableBalanceObserved"] = append(p.Header.Labels["availableBalanceObserved"], span.ID)
		}
	}
	for _, pair := range [][2]string{{"debitTurnover", "Расходы:"}, {"creditTurnover", "Пополнения:"}} {
		a := find(pair[1])
		ids := line(a, 0, true)
		if pair[0] == "debitTurnover" {
			p.Header.DebitTurnover = ids
		} else {
			p.Header.CreditTurnover = ids
		}
		p.Header.Labels[pair[0]] = []string{a.ID}
	}
	return p
}
func rowScopedGroupingMapping(t *testing.T, c statementextract.Chunk, numeric map[string]bool, fault string) statementextract.RowGrouping {
	t.Helper()
	p := rowScopedDeliveredProposal(t, c, numeric)
	// Controls are mutations of the delivered-alias answer, never fabricated Review.
	if fault == "prefix" || fault == "suffix" || fault == "separator" || fault == "all_description" {
		ids := p.Rows[1].Description
		drop := map[string]bool{}
		if fault == "prefix" {
			drop[ids[0]] = true
		}
		if fault == "suffix" {
			drop[ids[len(ids)-1]] = true
		}
		if fault == "all_description" {
			for _, id := range ids {
				drop[id] = true
			}
		}
		if fault == "separator" {
			for _, s := range c.Spans {
				if s.Text == "-" {
					for _, id := range ids {
						if id == s.ID {
							drop[id] = true
						}
					}
				}
			}
		}
		kept := []string{}
		for _, id := range ids {
			if !drop[id] {
				kept = append(kept, id)
			}
		}
		p.Rows[1].Description = kept
	}
	if fault == "missing_available_label" {
		p.Header.Labels["availableBalanceObserved"] = []string{}
	}
	if fault == "identifier" {
		for _, s := range c.Spans {
			if s.Text == "БИК" {
				p.Header.Labels["debitTurnover"] = []string{s.ID}
				p.Header.DebitTurnover = []string{}
				for _, v := range c.Spans {
					if v.Page == s.Page && v.Y == s.Y && numeric[v.ID] {
						p.Header.DebitTurnover = append(p.Header.DebitTurnover, v.ID)
					}
				}
			}
		}
	}
	if fault == "wrong_fx" {
		p.Rows[0].TransactionCurrency = p.Rows[0].AccountCurrency
	}
	contextIDs, used := map[string]bool{}, map[string]bool{}
	add := func(ids []string) {
		for _, id := range ids {
			contextIDs[id] = true
			used[id] = true
		}
	}
	for _, ids := range [][]string{p.Header.OpeningBookDate, p.Header.ClosingBookDate, p.Header.OpeningBookBalance, p.Header.ClosingBookBalance, p.Header.AvailableBalanceObserved, p.Header.DebitTurnover, p.Header.CreditTurnover, p.Header.Currency} {
		add(ids)
	}
	for _, ids := range p.Header.Labels {
		add(ids)
	}
	for _, r := range p.Rows {
		add(r.AccountCurrency)
		add(r.TransactionCurrency)
	}
	g := statementextract.RowGrouping{Header: p.Header, Rows: []statementextract.RowEnvelope{}, Context: []string{}, Service: []statementdata.ServiceMapping{}, Unresolved: []string{}}
	for _, r := range p.Rows {
		owned := map[string]bool{}
		for _, ids := range [][]string{r.OperationDate, r.BookingDate, r.AccountAmount, r.TransactionAmount, r.Description, r.DirectionEvidence} {
			for _, id := range ids {
				if !contextIDs[id] {
					owned[id] = true
					used[id] = true
				}
			}
		}
		ids := []string{}
		for _, s := range c.Spans {
			if owned[s.ID] {
				ids = append(ids, s.ID)
			}
		}
		g.Rows = append(g.Rows, statementextract.RowEnvelope{Spans: ids})
	}
	service := []string{}
	for _, s := range c.Spans {
		if contextIDs[s.ID] {
			g.Context = append(g.Context, s.ID)
		}
		if !used[s.ID] {
			service = append(service, s.ID)
		}
	}
	if len(service) > 0 {
		g.Service = append(g.Service, statementdata.ServiceMapping{Spans: service, Reason: "Explicit public fixture headings and unused fragments"})
	}
	if fault == "missing_owner" {
		g.Rows[0].Spans = g.Rows[0].Spans[1:]
	}
	if fault == "duplicate_owner" {
		g.Context = append(g.Context, g.Rows[0].Spans[0])
	}
	if fault == "unresolved" {
		id := g.Rows[1].Spans[len(g.Rows[1].Spans)-1]
		g.Rows[1].Spans = g.Rows[1].Spans[:len(g.Rows[1].Spans)-1]
		g.Unresolved = append(g.Unresolved, id)
	}
	if fault == "missing_row" {
		g.Service = append(g.Service, statementdata.ServiceMapping{Spans: g.Rows[1].Spans, Reason: "Deliberately wrong record ownership control"})
		g.Rows = g.Rows[:1]
	}
	return g
}
func (f *rowScopedFake) complete(ctx context.Context, r statementextract.RowScopedRequest) ([]byte, *statementextract.Billing, error) {
	f.t.Helper()
	if f.meter.reserves != 1 || f.meter.settles != 0 || f.meter.reserved != 161700 {
		f.t.Fatal("reservation must precede stage", f.meter)
	}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 120*time.Second {
		f.t.Fatal("call deadline")
	}
	f.calls++
	f.orders = append(f.orders, fmt.Sprintf("%s%d", r.Stage, r.RowOrdinal))
	f.requests = append(f.requests, r)
	c, numeric := rowScopedDelivered(f.t, r)
	var raw []byte
	if r.Stage == "grouping" {
		if len(c.Context) != 0 {
			f.t.Fatal("group context")
		}
		raw = rowScopedPayload(f.t, rowScopedGroupingMapping(f.t, c, numeric, f.groupFault))
	} else {
		row := rowScopedFieldMapping(c, numeric)
		if f.fieldFault == "missing_currency" {
			row.AccountCurrency = nil
		}
		if f.fieldFault == "missing_description" {
			row.Description = nil
		}
		if f.fieldFault == "wrong_fx" && r.RowOrdinal == 1 {
			row.TransactionCurrency = row.AccountCurrency
		}
		if f.fieldFault == "missing_clock" && len(row.OperationDate) > 1 {
			row.OperationDate = row.OperationDate[:1]
		}
		raw = rowScopedPayload(f.t, row)
	}
	bill := &statementextract.Billing{GenerationID: fmt.Sprintf("fake-s121-%d", f.calls), CostMicros: 10, InputTokens: 100, OutputTokens: 100}
	if f.failAt == f.calls {
		switch f.fault {
		case "provider":
			return nil, bill, errors.New("PRIVATE_PROVIDER")
		case "unknown":
			return raw, nil, errors.New("PRIVATE_BILL")
		case "cost":
			bill.CostMicros = 53901
		case "duplicate":
			bill.GenerationID = "fake-s121-1"
		case "overflow":
			raw = []byte(strings.Repeat("X", statementextract.RowScopedResponseLimit+1))
		case "json":
			raw = []byte(`{"PRIVATE":null}`)
		case "cancel_in_call":
			f.cancel()
			<-ctx.Done()
			return raw, bill, ctx.Err()
		}
	}
	return raw, bill, nil
}
func rowScopedRun(t *testing.T, d statementdata.Document, name string, f *rowScopedFake) (RowScopedOutcome, error) {
	t.Helper()
	return RunRowScopedOffline(context.Background(), Sample{name, d, FixtureHash(name)}, RowScopedStages{f.complete, f.complete}, f.meter)
}
func TestRowScopedOfflineCausalAndAccounting(t *testing.T) {
	d := clockTestDocument(t)
	for _, mode := range []string{"good", "missing_owner", "duplicate_owner", "unresolved", "prefix", "suffix", "separator", "all_description", "missing_row", "missing_description", "missing_clock", "missing_currency", "wrong_fx"} {
		t.Run(mode, func(t *testing.T) {
			m := &rowScopedMeter{}
			f := &rowScopedFake{t: t, meter: m}
			switch mode {
			case "missing_description", "missing_clock", "missing_currency":
				f.fieldFault = mode
			case "wrong_fx":
				f.groupFault = mode
				f.fieldFault = mode
			default:
				if mode != "good" {
					f.groupFault = mode
				}
			}
			out, e := rowScopedRun(t, d, "cards", f)
			if m.reserves != 1 || m.settles != 1 || m.actual == nil || *m.actual != int64(f.calls*10) || out.ActualMicros == nil || !out.SettlementRecorded || f.calls > 3 {
				t.Fatal("accounting", m, out, e)
			}
			if mode == "good" {
				if e != nil || !out.FullQualityPassed || out.Result == nil || out.Result.Correct != 20 || len(out.Checks) != 20 || len(out.Precision) != 2 {
					t.Fatal(out, e)
				}
				if strings.Join(f.orders, ",") != "grouping0,row1,row2" {
					t.Fatal("order", f.orders)
				}
				return
			}
			if out.FullQualityPassed {
				t.Fatal("negative accepted", mode, out)
			}
			if mode == "prefix" || mode == "suffix" || mode == "separator" || mode == "all_description" || mode == "missing_row" || mode == "wrong_fx" || mode == "missing_currency" {
				if e != nil || out.Result == nil || out.Result.Passed {
					t.Fatal("golden failure not observed", mode, out, e)
				}
			} else {
				if e == nil || out.Result != nil {
					t.Fatal("invalid stage got fabricated quality", mode, out, e)
				}
			}
			t.Logf("%s status=%s stage=%s cause=%s calls=%d", mode, out.Status, out.Stage, out.Cause, f.calls)
		})
	}
}
func TestRowScopedOfflineStopAndUncertainty(t *testing.T) {
	for _, mode := range []string{"reserve", "settle", "provider", "unknown", "cost", "duplicate", "overflow", "json", "cancel_before", "deadline_before", "cancel_in_call"} {
		t.Run(mode, func(t *testing.T) {
			m := &rowScopedMeter{fail: mode}
			f := &rowScopedFake{t: t, meter: m, fault: mode, failAt: 2}
			if mode == "overflow" {
				f.failAt = 1
			}
			ctx := context.Background()
			cancel := func() {}
			if mode == "cancel_before" || mode == "cancel_in_call" {
				ctx, cancel = context.WithCancel(ctx)
				f.cancel = cancel
				if mode == "cancel_before" {
					cancel()
				}
			}
			if mode == "deadline_before" {
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
			}
			defer cancel()
			out, e := RunRowScopedOffline(ctx, Sample{"cards", clockTestDocument(t), FixtureHash("cards")}, RowScopedStages{f.complete, f.complete}, m)
			if e == nil || out.FullQualityPassed || strings.Contains(out.Cause, "PRIVATE") {
				t.Fatal(out, e)
			}
			if mode == "reserve" || mode == "cancel_before" || mode == "deadline_before" {
				if f.calls != 0 || m.settles != 0 {
					t.Fatal("early refusal", f.calls, m)
				}
				return
			}
			if m.settles != 1 {
				t.Fatal("settlement", m)
			}
			if mode == "unknown" || mode == "cost" || mode == "duplicate" {
				if m.actual != nil || out.ActualMicros != nil || f.calls != 2 {
					t.Fatal("invented known billing", out, m)
				}
			} else if m.actual == nil || *m.actual != int64(f.calls*10) {
				t.Fatal("known prefix lost", out, m)
			}
			if mode == "settle" {
				if f.calls != 3 || out.SettlementRecorded {
					t.Fatal(out, m)
				}
			} else if f.calls > 2 || out.Result != nil {
				t.Fatal("continued after refusal", out, m)
			}
		})
	}
}

func TestRowScopedOfflineRUControl(t *testing.T) {
	f, e := os.Open("../pdfextract/testdata/turnover-roles-ru.xml")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	d, e := statementdata.DecodeLayout(f)
	if e != nil {
		t.Fatal(e)
	}
	meter := &rowScopedMeter{}
	fake := &rowScopedFake{t: t, meter: meter}
	out, e := rowScopedRun(t, d, "turnover-roles-ru", fake)
	if e != nil || !out.FullQualityPassed {
		t.Fatalf("result=%+v checks=%+v err=%v", out.Result, out.Checks, e)
	}
}

func rowScopedMany(n int) statementdata.Document {
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}}
	for i := 0; i < n; i++ {
		y := 1 + float64(i)/10
		d.Spans = append(d.Spans, statementdata.Span{ID: fmt.Sprintf("public-%d", i), Page: 1, Block: 1, Line: i + 1, Word: 1, Text: "1.00", Box: statementdata.Box{XMin: 1, YMin: y, XMax: 10, YMax: y + 1}})
	}
	return d
}
func TestRowScopedOfflineLateRequestLimit(t *testing.T) {
	// Inline synthetic boundary input, not a new corpus entry or a quality sample.
	n := 1900
	d := rowScopedMany(n)
	group, e := statementextract.BuildRowGrouping(d)
	if e != nil {
		t.Fatal("late boundary precondition", e)
	}
	c, _ := rowScopedDelivered(t, group)
	ids := []string{}
	for _, s := range c.Spans {
		ids = append(ids, s.ID)
	}
	g := statementextract.RowGrouping{Rows: []statementextract.RowEnvelope{{Spans: ids}}, Context: []string{}, Service: []statementdata.ServiceMapping{}, Unresolved: []string{}}
	encoded := rowScopedPayload(t, g)
	g, e = statementextract.DecodeRowGrouping(d, encoded)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = statementextract.BuildRowFields(d, g, 1); statementextract.RowScopedCause(e) != "request_limit_exceeded" {
		t.Fatal("late boundary precondition", e)
	}
	meter := &rowScopedMeter{}
	calls := 0
	stages := RowScopedStages{Grouping: func(context.Context, statementextract.RowScopedRequest) ([]byte, *statementextract.Billing, error) {
		calls++
		return encoded, &statementextract.Billing{GenerationID: "fake-bound", CostMicros: 10, InputTokens: 1, OutputTokens: 1}, nil
	}, Row: func(context.Context, statementextract.RowScopedRequest) ([]byte, *statementextract.Billing, error) {
		t.Fatal("oversized row request delivered")
		return nil, nil, nil
	}}
	out, e := RunRowScopedOffline(context.Background(), Sample{"cards", d, FixtureHash("cards")}, stages, meter)
	if e == nil || out.Cause != "request_limit_exceeded" || out.Stage != "row1" || out.Result != nil || calls != 1 || meter.settles != 1 || meter.actual == nil || *meter.actual != 10 {
		t.Fatal(out, meter, e)
	}
	t.Logf("late-cap spans=%d groupingBytes=%d calls=%d knownPrefix=%d", n, group.InputBytes, calls, *meter.actual)
}
func TestRowScopedOfflineInitialRequestLimit(t *testing.T) {
	meter := &rowScopedMeter{}
	never := func(context.Context, statementextract.RowScopedRequest) ([]byte, *statementextract.Billing, error) {
		t.Fatal("oversize input delivered")
		return nil, nil, nil
	}
	out, e := RunRowScopedOffline(context.Background(), Sample{"cards", rowScopedMany(2001), FixtureHash("cards")}, RowScopedStages{never, never}, meter)
	if e == nil || out.Cause != "request_limit_exceeded" || meter.reserves != 0 || len(out.Calls) != 0 {
		t.Fatal(out, meter, e)
	}
}

func rowScopedDeliveredProposal(t *testing.T, c statementextract.Chunk, numeric map[string]bool) statementdata.Proposal {
	t.Helper()
	var p statementdata.Proposal
	ru := false
	for _, s := range c.Spans {
		if s.Text == "12.10.2026" {
			ru = true
		}
	}
	if ru {
		p = rowScopedRU(c, numeric)
	} else {
		f := &activationFixtureReasoner{}
		var e error
		p, _, e = f.Complete(context.Background(), statementextract.ScreeningPolicy(), c)
		if e != nil {
			t.Fatal(e)
		}
	}
	used := map[string]bool{}
	for _, r := range p.Rows {
		for _, id := range rowSelections(r) {
			used[id] = true
		}
	}
	for _, ids := range [][]string{p.Header.OpeningBookDate, p.Header.ClosingBookDate, p.Header.OpeningBookBalance, p.Header.ClosingBookBalance, p.Header.AvailableBalanceObserved, p.Header.DebitTurnover, p.Header.CreditTurnover, p.Header.Currency} {
		for _, id := range ids {
			used[id] = true
		}
	}
	for _, ids := range p.Header.Labels {
		for _, id := range ids {
			used[id] = true
		}
	}
	service := []string{}
	for _, span := range c.Spans {
		if !used[span.ID] {
			service = append(service, span.ID)
		}
	}
	p.Service = []statementdata.ServiceMapping{{Spans: service, Reason: "Explicit public fixture unused headings and identifiers"}}
	return p
}
func rowScopedBaseline(t *testing.T, d statementdata.Document, name string) (Result, int) {
	t.Helper()
	plan, e := statementextract.BuildComparisonPlan(d, statementextract.ScreeningPolicy())
	if e != nil {
		t.Fatal(e)
	}
	request := statementextract.RowScopedRequest{Messages: statementextract.Messages(plan.Chunks[0]), Schema: statementextract.RequestSchema(plan.Chunks[0])}
	if statementextract.PromptSHA(request.Messages[0]["content"]) != statementextract.ClockBasePromptSHA {
		t.Fatal("base identity")
	}
	c, numeric := rowScopedDelivered(t, request)
	delivered := rowScopedDeliveredProposal(t, c, numeric)
	var decoded statementdata.Proposal
	if json.Unmarshal(rowScopedPayload(t, delivered), &decoded) != nil {
		t.Fatal("base response")
	}
	p, e := statementextract.CanonicalComparisonProposal(plan, decoded)
	if e != nil {
		t.Fatal(e)
	}
	p, e = statementextract.MergeProposals(plan, []statementdata.Proposal{p})
	if e != nil {
		t.Fatal(e)
	}
	pages := []int{}
	for _, page := range d.Pages {
		pages = append(pages, page.Number)
	}
	review, e := statementdata.ValidateProposal(d, p, pages, true, "")
	if e != nil {
		t.Fatal(e)
	}
	result, _ := ScreeningChecks(name, review)
	raw, _ := json.Marshal(request.Messages)
	return result, len(raw) + len(request.Schema)
}

func TestRowScopedPanicNeverSettlesFree(t *testing.T) {
	meter := &rowScopedMeter{}
	calls := 0
	defer func() {
		if recover() == nil {
			t.Fatal("test client did not panic")
		}
		if meter.reserves != 1 || meter.settles != 1 || meter.actual != nil || calls != 1 {
			t.Fatal("uncertain delivery settled as free", meter, calls)
		}
	}()
	panicClient := func(context.Context, statementextract.RowScopedRequest) ([]byte, *statementextract.Billing, error) {
		calls++
		panic("PRIVATE_CLIENT")
	}
	_, _ = RunRowScopedOffline(context.Background(), Sample{"cards", clockTestDocument(t), FixtureHash("cards")}, RowScopedStages{panicClient, panicClient}, meter)
}
