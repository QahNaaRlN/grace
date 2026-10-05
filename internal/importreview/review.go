package importreview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

type profile struct {
	domain string
	first  int
	key    string
}

var ErrReviewLimit = errors.New("review_limit")

var profiles = map[string]profile{
	"Операции": {"transaction", 5, "A"}, "Кредитные карты": {"credit_snapshot", 5, "A"},
	"Выписки карт": {"payment_requirement", 5, "A"}, "Задолженности": {"debt_snapshot", 5, "B"},
	"Зарплата": {"income_plan", 5, "A"}, "Категории": {"category", 4, "A"},
	"Сводка": {"projection", 2, "A"}, "Сверка счетов": {"bank_reconciliation", 5, "A"},
	"Первичные документы": {"document_reference", 5, "A"}, "Инструкция": {"instruction", 4, "A"},
}

func Review(s Snapshot) (Report, error) {
	if err := validate(s); err != nil {
		return Report{}, err
	}
	payload, _ := json.Marshal(s)
	hash := sha256.Sum256(payload)
	report := Report{SchemaVersion: ReportVersion, Source: Metadata{s.SourceID, s.CapturedAt, s.Locale, s.DeclaredTimeZone, s.ConfirmedTimeZone}, PayloadSHA256: hex.EncodeToString(hash[:]), Records: []Record{}, Candidates: []Candidate{}, Issues: []Issue{}, DomainCounts: map[string]int{}}
	if s.DeclaredTimeZone != s.ConfirmedTimeZone {
		report.Issues = append(report.Issues, Issue{"timezone_override", "warning", "metadata", "Confirmed operator timezone overrides source metadata; raw dates are retained."})
	}
	sheets := append([]Sheet(nil), s.Sheets...)
	sort.Slice(sheets, func(i, j int) bool { return sheets[i].SheetID < sheets[j].SheetID })
	for _, sh := range sheets {
		byRow := map[int]map[string]Cell{}
		for _, rg := range sh.Ranges {
			for i, row := range rg.Rows {
				n := rg.StartRow + i
				if byRow[n] == nil {
					byRow[n] = map[string]Cell{}
				}
				for j, c := range row.Values {
					byRow[n][string(rune('A'+rg.StartColumn+j-1))] = c
				}
			}
		}
		rowNumbers := []int{}
		for n := range byRow {
			rowNumbers = append(rowNumbers, n)
		}
		sort.Ints(rowNumbers)
		p, known := profiles[sh.Title]
		for _, n := range rowNumbers {
			raw := byRow[n]
			hasFact := false
			hasData := false
			for _, c := range raw {
				if meaningful(c) {
					hasData = true
				}
				if c.UserEnteredValue != nil && c.UserEnteredValue.FormulaValue == nil && meaningful(c) {
					hasFact = true
				}
			}
			if !hasData {
				continue
			}
			if known && n < p.first {
				continue
			}
			h := sha256.Sum256([]byte(s.SourceID + ":" + strconv.FormatInt(sh.SheetID, 10) + ":" + strconv.Itoa(n) + ":" + report.PayloadSHA256))
			r := Record{ID: hex.EncodeToString(h[:]), SheetID: sh.SheetID, Sheet: sh.Title, Row: n, Domain: p.domain, Raw: raw, Issues: []Issue{}}
			if !hasFact {
				r.issue("formula_derived_row", "warning", "A", "Nonempty cached formula results are retained as source evidence, not executed or confirmed.")
			}
			if !known {
				r.Domain = "unclassified"
				r.issue("unknown_sheet", "error", "A", "Unknown sheet profile; raw row is retained.")
			} else {
				// Annotation rows remain visible without being interpreted as balances/plans.
				populated := 0
				for _, c := range raw {
					if meaningful(c) {
						populated++
					}
				}
				if (p.domain == "bank_reconciliation" || p.domain == "income_plan") && populated == 1 {
					r.Domain = "note"
				} else {
					normalizeRecord(&r, s)
				}
			}
			report.DomainCounts[r.Domain]++
			report.Records = append(report.Records, r)
		}
	}
	if err := reconcile(&report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r *Record) issue(code, severity, col, message string) {
	r.Issues = append(r.Issues, Issue{code, severity, fmt.Sprintf("%s!%s%d", r.Sheet, col, r.Row), message})
}
func (r *Record) money(name, col string) {
	if r.Normalized.Amounts == nil {
		r.Normalized.Amounts = map[string]*int64{}
	}
	m, err := Money(r.Raw[col])
	r.Normalized.Amounts[name] = m
	if err != nil {
		r.issue("ambiguous_money", "error", col, "Amount cannot be converted exactly to integer minor units.")
	}
}
func (r *Record) date(name, col, zone string) {
	if r.Normalized.Dates == nil {
		r.Normalized.Dates = map[string]*Date{}
	}
	d, err := NormalizeDate(r.Raw[col], zone)
	r.Normalized.Dates[name] = d
	if err != nil {
		r.issue("invalid_date", "error", col, "Date cannot be interpreted using the confirmed timezone.")
	}
}
func (r *Record) label(name, col string) {
	if r.Normalized.Labels == nil {
		r.Normalized.Labels = map[string]string{}
	}
	r.Normalized.Labels[name] = text(r.Raw[col])
}

func normalizeRecord(r *Record, s Snapshot) {
	n := &r.Normalized
	n.Currency = "RUB"
	switch r.Domain {
	case "transaction":
		normalizeTransaction(r, s)
	case "credit_snapshot":
		n.Account = text(r.Raw["A"])
		r.label("organization", "B")
		r.label("instrumentType", "C")
		for name, col := range map[string]string{"creditLimit": "D", "reportedDebt": "E", "availableLimit": "F", "requiredPayment": "J", "minimumPayment": "K"} {
			r.money(name, col)
		}
		for name, col := range map[string]string{"periodStart": "G", "periodEnd": "H", "paymentDue": "I"} {
			r.date(name, col, s.ConfirmedTimeZone)
		}
		r.label("status", "L")
		r.label("comment", "M")
		r.issue("unresolved_debt_basis", "warning", "E", "Reported debt is not assumed to be principal; components and observation date require evidence.")
	case "payment_requirement":
		n.Account = text(r.Raw["A"])
		for name, col := range map[string]string{"periodStart": "B", "periodEnd": "C", "paymentDue": "D"} {
			r.date(name, col, s.ConfirmedTimeZone)
		}
		for name, col := range map[string]string{"periodPurchases": "E", "graceRequirement": "F", "reportedPaid": "G", "reportedRemaining": "H", "minimumPayment": "I"} {
			r.money(name, col)
		}
		r.label("reportedStatus", "J")
		r.label("comment", "K")
		n.Kind = "unresolved_requirement"
		if n.Amounts["graceRequirement"] == nil {
			r.issue("missing_requirement_amount", "warning", "F", "Grace requirement is unknown; a scheduled amount in another field/formula must not be treated as grace.")
		}
		if f := formula(r.Raw["E"]); strings.Contains(f, `"Кредитная карта"`) {
			r.issue("incompatible_formula", "warning", "E", "Formula uses a funding label different from the current profile.")
		}
	case "debt_snapshot":
		n.Description = text(r.Raw["B"])
		r.label("repaymentPriority", "A")
		r.label("debtType", "C")
		for name, col := range map[string]string{"reportedOutstanding": "D", "minimumPayment": "F", "extraPaymentPlan": "G", "paymentPlan": "H"} {
			r.money(name, col)
		}
		r.label("rateValue", "E")
		if meaningful(r.Raw["E"]) {
			r.issue("unresolved_rate_basis", "warning", "E", "A rate without explicit period/basis is not assumed to be annual APR.")
		}
		r.date("paymentDue", "I", s.ConfirmedTimeZone)
		r.label("reportedOverdue", "J")
		r.label("status", "K")
		r.label("estimatedMonths", "L")
		r.label("comment", "M")
		r.issue("unresolved_debt_basis", "warning", "D", "Outstanding debt and future scheduled payments are distinct; snapshot date/components require evidence.")
	case "income_plan":
		n.Description = text(r.Raw["A"])
		r.date("baseDate", "B", s.ConfirmedTimeZone)
		r.date("workingDate", "C", s.ConfirmedTimeZone)
		for name, col := range map[string]string{"plannedIncome": "D", "reportedReceived": "E", "reportedDifference": "F"} {
			r.money(name, col)
		}
		if meaningful(r.Raw["H"]) {
			r.date("holiday", "H", s.ConfirmedTimeZone)
		}
	case "category":
		n.Category = text(r.Raw["A"])
		r.label("role", "B")
		r.money("budgetLimit", "C")
		n.Subcategory = text(r.Raw["D"])
		r.label("defaultPriority", "E")
		r.label("explanation", "F")
	case "bank_reconciliation":
		n.Account = text(r.Raw["A"])
		for name, col := range map[string]string{"openingBalance": "B", "credits": "C", "debits": "D", "closingBalance": "E"} {
			r.money(name, col)
		}
		r.label("reportedControl", "F")
		r.label("comment", "G")
		a := n.Amounts
		if a["openingBalance"] == nil || a["credits"] == nil || a["debits"] == nil || a["closingBalance"] == nil {
			r.issue("incomplete_reconciliation", "warning", "B", "All four control amounts are required to verify a statement balance.")
		} else if !balanceEqual(*a["openingBalance"], *a["credits"], *a["debits"], *a["closingBalance"]) {
			r.issue("balance_mismatch", "error", "F", "Opening plus credits minus debits differs from closing balance.")
		}
	case "document_reference":
		r.OriginalID = text(r.Raw["A"])
		r.label("filename", "B")
		r.label("mimeType", "C")
		r.label("group", "D")
		r.label("messageRef", "E")
		r.label("reportedStatus", "F")
		r.label("comment", "G")
		r.issue("document_unavailable", "warning", "B", "A registry entry does not contain the original document; no file is fetched.")
	case "projection", "instruction":
		r.label("title", "A")
		n.Currency = ""
	}
	// Formulas are retained as evidence only. These known stale literals never execute.
	for col, c := range r.Raw {
		if strings.Contains(formula(c), `"Погашение кредитной карты"`) {
			r.issue("incompatible_formula", "warning", col, "Formula filters a repayment label not used by the current transaction profile.")
		}
	}
	// Sort every issue list after map traversal to keep reports deterministic.
	sort.Slice(r.Issues, func(i, j int) bool {
		a, b := r.Issues[i], r.Issues[j]
		return a.Coordinate+":"+a.Code < b.Coordinate+":"+b.Code
	})
}

func normalizeTransaction(r *Record, s Snapshot) {
	n := &r.Normalized
	r.OriginalID = text(r.Raw["O"])
	if r.OriginalID == "" {
		r.issue("missing_id", "warning", "O", "Original ID is missing; the snapshot-local record identity is retained.")
	} else if strings.Contains(formula(r.Raw["O"]), "ROW()") {
		r.issue("unstable_source_id", "warning", "O", "Formula ID depends on row order; it is not a cross-snapshot financial identity.")
	}
	if r.Row > 304 {
		r.issue("outside_formula_scope", "warning", "A", "Row lies outside the current summary formula range; review individually.")
	}
	kinds := map[string]string{"Расход": "purchase", "Доход": "income_received", "Возврат": "refund", "Перевод между своими счетами": "internal_transfer", "Перевод": "unclassified_transfer", "Погашение": "debt_payment", "Погашение долга": "debt_payment", "Погашение кредитной карты": "debt_payment", "Платёж по другому долгу": "debt_payment"}
	n.Kind = kinds[text(r.Raw["B"])]
	if n.Kind == "" {
		r.issue("unknown_kind", "error", "B", "Operation kind has no explicit mapping.")
	}
	if n.Kind == "unclassified_transfer" {
		r.issue("unresolved_transfer", "error", "B", "Transfer ownership, purpose and destination need confirmation.")
	}
	var err error
	n.Date, err = NormalizeDate(r.Raw["A"], s.ConfirmedTimeZone)
	if err != nil || n.Date == nil {
		r.issue("invalid_date", "error", "A", "A valid transaction date is required.")
	} else if n.Date.Precision == "date" || n.Date.Precision == "serial_fraction" {
		r.issue("date_precision", "info", "A", "Only a calendar date is known; no event time is invented.")
	}
	n.AmountMinor, err = Money(r.Raw["F"])
	if err != nil || n.AmountMinor == nil || *n.AmountMinor <= 0 {
		r.issue("ambiguous_money", "error", "F", "A positive exact amount in minor units is required.")
	} else if v := cellValue(r.Raw["F"]); v != nil && v.StringValue != nil {
		r.issue("text_money", "warning", "F", "Text amount was normalized exactly; spreadsheet aggregates may omit it.")
	}
	n.Description = text(r.Raw["C"])
	n.Category = text(r.Raw["D"])
	n.Subcategory = text(r.Raw["E"])
	n.Recipient = text(r.Raw["K"])
	n.Account = text(r.Raw["H"])
	n.AccountID = s.Mappings.Accounts[n.Account]
	if n.AccountID == "" {
		r.issue("unresolved_account", "error", "H", "A confirmed canonical account mapping is required; route text is not an account.")
	}
	n.CategoryID = s.Mappings.Categories[n.Category]
	if n.Kind == "purchase" && n.CategoryID == "" {
		r.issue("unresolved_category", "error", "D", "A confirmed expense category mapping is required.")
	}
	f, ok := s.Mappings.Funding[text(r.Raw["G"])]
	if !ok {
		switch text(r.Raw["G"]) {
		case "Собственные средства":
			f.Kind = "own"
		case "Кредитные средства":
			f.Kind = "borrowed"
		default:
			f.Kind = "unknown"
		}
	}
	n.Funding = &f
	if f.Kind == "unknown" || f.Kind == "borrowed" && f.FacilityID == "" {
		r.issue("unresolved_funding", "warning", "G", "Unknown funding or an unlinked credit facility needs operator evidence.")
	}
	priorities := map[string]string{"Обязательный": "essential", "Можно сократить": "reducible", "Необязательный": "optional", "Нужно проверить": "review"}
	raw := text(r.Raw["I"])
	n.Priority = priorities[raw]
	if raw != "" && n.Priority == "" {
		r.issue("unknown_priority", "warning", "I", "Expense priority has no explicit mapping.")
	}
	var valid bool
	n.Recurring, valid = boolCell(r.Raw["J"])
	if !valid {
		r.issue("invalid_boolean", "warning", "J", "Recurring hint is not a boolean or a recognized yes/no value.")
	}
	if n.Kind == "internal_transfer" {
		r.issue("unresolved_transfer", "error", "H", "Two canonical accounts are required; a description is not proof of destination.")
	}
	if n.Kind == "debt_payment" {
		r.issue("unresolved_debt_target", "warning", "K", "Debt target and principal/interest allocation require evidence.")
	}
	if n.Kind == "refund" {
		r.issue("unresolved_original_purchase", "error", "K", "Refund must link to a confirmed original purchase before publication.")
	}
}

func balanceEqual(open, credits, debits, close int64) bool {
	// Exact big integer arithmetic also handles adversarial near-int64 control amounts.
	return decimalBalance(open, credits, debits) == strconv.FormatInt(close, 10)
}

func decimalBalance(open, credits, debits int64) string {
	n := big.NewInt(open)
	n.Add(n, big.NewInt(credits))
	n.Sub(n, big.NewInt(debits))
	return n.String()
}

func reconcile(report *Report) error {
	primary := []int{}
	for i, r := range report.Records {
		if r.Domain == "transaction" && r.Row <= 304 {
			primary = append(primary, i)
		}
	}
	ids := map[string]int{}
	for i := range report.Records {
		r := &report.Records[i]
		if r.Domain != "transaction" {
			continue
		}
		if r.OriginalID != "" {
			if previous, ok := ids[r.OriginalID]; ok {
				r.issue("duplicate_source_id", "warning", "O", "Source ID repeats another row; neither record is merged.")
				report.Records[previous].issue("duplicate_source_id", "warning", "O", "Source ID repeats another row; neither record is merged.")
			} else {
				ids[r.OriginalID] = i
			}
		}
		if r.Row <= 304 {
			continue
		}
		count := 0
		for _, j := range primary {
			o := &report.Records[j]
			a, b := r.Normalized, o.Normalized
			if a.AmountMinor == nil || b.AmountMinor == nil || *a.AmountMinor != *b.AmountMinor || a.Kind == "" || a.Kind != b.Kind || a.Date == nil || b.Date == nil {
				continue
			}
			dayMatch := a.Date.Local[:10] == b.Date.Local[:10]
			descriptionMatch := canonical(a.Description) != "" && canonical(a.Description) == canonical(b.Description)
			if !dayMatch && !descriptionMatch {
				continue
			}
			rule := "date_kind_amount"
			matched := []string{"kind", "amountMinor"}
			if dayMatch {
				matched = append(matched, "localDate")
			} else {
				rule = "description_kind_amount_date_conflict"
				r.issue("date_conflict", "warning", "A", "A candidate has the same description/kind/amount but another date; source evidence is required.")
			}
			if descriptionMatch {
				matched = append(matched, "description")
			}
			differences := []string{}
			for _, col := range []string{"A", "C", "D", "E", "G", "H", "I", "J", "K", "L", "O"} {
				if !jsonEqual(r.Raw[col], o.Raw[col]) {
					differences = append(differences, col)
				}
			}
			if len(report.Candidates) >= MaxCandidates {
				return ErrReviewLimit
			}
			report.Candidates = append(report.Candidates, Candidate{r.ID, o.ID, rule, matched, differences, "needs_confirmation"})
			count++
		}
		if count == 0 {
			r.issue("no_candidate", "warning", "A", "No candidate under the declared rules; this does not establish a new unique operation.")
		} else {
			r.issue("duplicate_candidate", "warning", "A", "Possible overlap requires individual confirmation; records are not merged.")
		}
	}
	return nil
}
