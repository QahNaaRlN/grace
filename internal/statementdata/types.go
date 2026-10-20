// Package statementdata validates bank statement evidence without AI, SQL or
// financial writes. A proposal selects source spans; it never supplies money.
package statementdata

const (
	ExtractorVersion       = "poppler-25.04.0/bbox-v1"
	SchemaVersion          = "grace.bank-statement.v1"
	MaxPDFBytes            = 10 << 20
	MaxTextBytes           = 8 << 20
	MaxXMLBytes            = 64 << 20
	MaxPages               = 50
	MaxSpans               = 200000
	MaxRows                = 5000
	MaxWireMinor     int64 = 9007199254740991
)

type Box struct {
	XMin float64 `json:"xMin"`
	YMin float64 `json:"yMin"`
	XMax float64 `json:"xMax"`
	YMax float64 `json:"yMax"`
}
type Span struct {
	ID    string `json:"id"`
	Page  int    `json:"page"`
	Block int    `json:"block"`
	Line  int    `json:"line"`
	Word  int    `json:"word"`
	Box   Box    `json:"box"`
	Text  string `json:"text"`
}
type Page struct {
	Number int     `json:"number"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
type Document struct {
	Extractor string `json:"extractor"`
	Pages     []Page `json:"pages"`
	Spans     []Span `json:"spans"`
}

// Selection contains locators only. There is no model-supplied value to trust.
type Selection []string
type RowMapping struct {
	OperationDate       Selection `json:"operationDate"`
	BookingDate         Selection `json:"bookingDate"`
	AccountAmount       Selection `json:"accountAmount"`
	AccountCurrency     Selection `json:"accountCurrency"`
	TransactionAmount   Selection `json:"transactionAmount"`
	TransactionCurrency Selection `json:"transactionCurrency"`
	Description         Selection `json:"description"`
	// Direction is required for unsigned amount columns and requires grounded
	// column evidence, subsequently confirmed by the owner as a field role.
	Direction         string    `json:"direction"`
	DirectionEvidence Selection `json:"directionEvidence"`
}
type HeaderMapping struct {
	OpeningBookDate          Selection            `json:"openingBookDate"`
	ClosingBookDate          Selection            `json:"closingBookDate"`
	Labels                   map[string]Selection `json:"labels"`
	OpeningBookBalance       Selection            `json:"openingBookBalance"`
	ClosingBookBalance       Selection            `json:"closingBookBalance"`
	AvailableBalanceObserved Selection            `json:"availableBalanceObserved"`
	DebitTurnover            Selection            `json:"debitTurnover"`
	CreditTurnover           Selection            `json:"creditTurnover"`
	Currency                 Selection            `json:"currency"`
}
type ServiceMapping struct {
	Spans  Selection `json:"spans"`
	Reason string    `json:"reason"`
}
type Proposal struct {
	Rows    []RowMapping     `json:"rows"`
	Header  HeaderMapping    `json:"header"`
	Service []ServiceMapping `json:"service"`
}
type Evidence struct {
	Role    string `json:"role"`
	Spans   []Span `json:"spans"`
	Literal string `json:"literal"`
}
type Money struct {
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency"`
}
type LocalDate struct {
	Local     string `json:"local"`
	Precision string `json:"precision"`
	// UTC is absent until a bank timezone/offset is explicitly supplied.
	UTC      string `json:"utc,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
}
type Row struct {
	Ordinal           int        `json:"ordinal"`
	OperationDate     *LocalDate `json:"operationDate"`
	BookingDate       *LocalDate `json:"bookingDate"`
	AccountAmount     *Money     `json:"accountAmount"`
	TransactionAmount *Money     `json:"transactionAmount"`
	Direction         string     `json:"direction"`
	Description       string     `json:"description"`
	Evidence          []Evidence `json:"evidence"`
	Issues            []string   `json:"issues"`
}
type Header struct {
	OpeningBookDate          *LocalDate `json:"openingBookDate"`
	ClosingBookDate          *LocalDate `json:"closingBookDate"`
	OpeningBookBalance       *Money     `json:"openingBookBalance"`
	ClosingBookBalance       *Money     `json:"closingBookBalance"`
	AvailableBalanceObserved *Money     `json:"availableBalanceObserved"`
	DebitTurnover            *Money     `json:"debitTurnover"`
	CreditTurnover           *Money     `json:"creditTurnover"`
	Evidence                 []Evidence `json:"evidence"`
}
type Check struct {
	Status  string   `json:"status"`
	Reasons []string `json:"reasons"`
}
type Reconciliation struct {
	FinancialCoverage Check    `json:"financialCoverage"`
	Turnover          Check    `json:"turnoverCheck"`
	BookBalance       Check    `json:"bookBalanceCheck"`
	RowCoverage       Check    `json:"rowCoverage"`
	LedgerBalance     Check    `json:"ledgerBalanceCheck"`
	DebitMinor        *int64   `json:"debitMinor"`
	CreditMinor       *int64   `json:"creditMinor"`
	Blockers          []string `json:"blockers"`
	Warnings          []string `json:"warnings"`
}
type Review struct {
	unmappedSpanIDs []string
	Rows            []Row          `json:"rows"`
	Header          Header         `json:"header"`
	Reconciliation  Reconciliation `json:"reconciliation"`
}
