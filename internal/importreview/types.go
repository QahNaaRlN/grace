// Package importreview reviews operator-provided snapshots entirely offline.
// It never creates financial facts or connects to a provider or database.
package importreview

import "encoding/json"

const SnapshotVersion = "grace.sheet-snapshot.v1"
const ReportVersion = "grace.import-review.v1"
const MaxBytes = 16 << 20
const MaxRows = 10000
const MaxCells = 100000
const MaxCandidates = 50000

type Value struct {
	NumberValue  json.RawMessage `json:"numberValue,omitempty"`
	StringValue  *string         `json:"stringValue,omitempty"`
	BoolValue    *bool           `json:"boolValue,omitempty"`
	FormulaValue *string         `json:"formulaValue,omitempty"`
}
type Cell struct {
	UserEnteredValue *Value          `json:"userEnteredValue,omitempty"`
	EffectiveValue   *Value          `json:"effectiveValue,omitempty"`
	FormattedValue   string          `json:"formattedValue,omitempty"`
	Note             string          `json:"note,omitempty"`
	DataValidation   json.RawMessage `json:"dataValidation,omitempty"`
}
type Row struct {
	Values []Cell `json:"values"`
}
type Range struct {
	StartRow    int   `json:"startRow"`
	StartColumn int   `json:"startColumn"`
	Rows        []Row `json:"rows"`
}
type Sheet struct {
	SheetID     int64   `json:"sheetId"`
	Title       string  `json:"title"`
	RowCount    int     `json:"rowCount"`
	ColumnCount int     `json:"columnCount"`
	Ranges      []Range `json:"ranges"`
}
type Funding struct {
	Kind       string `json:"kind"`
	FacilityID string `json:"facilityId,omitempty"`
}
type Mappings struct {
	Accounts   map[string]string  `json:"accounts,omitempty"`
	Categories map[string]string  `json:"categories,omitempty"`
	Funding    map[string]Funding `json:"funding,omitempty"`
}
type Snapshot struct {
	SchemaVersion     string   `json:"schemaVersion"`
	SourceID          string   `json:"sourceId"`
	CapturedAt        string   `json:"capturedAt"`
	Locale            string   `json:"locale"`
	DeclaredTimeZone  string   `json:"declaredTimeZone"`
	ConfirmedTimeZone string   `json:"confirmedTimeZone"`
	Sheets            []Sheet  `json:"sheets"`
	Mappings          Mappings `json:"mappings,omitempty"`
}
type Issue struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Coordinate string `json:"coordinate"`
	Message    string `json:"message"`
}
type Date struct {
	Local     string `json:"local"`
	Precision string `json:"precision"`
	TimeZone  string `json:"timeZone"`
	UTC       string `json:"utc,omitempty"`
}
type Fields struct {
	Kind        string   `json:"kind,omitempty"`
	Date        *Date    `json:"date,omitempty"`
	Currency    string   `json:"currency,omitempty"`
	AmountMinor *int64   `json:"amountMinor,omitempty"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Subcategory string   `json:"subcategory,omitempty"`
	CategoryID  string   `json:"categoryId,omitempty"`
	Account     string   `json:"account,omitempty"`
	AccountID   string   `json:"accountId,omitempty"`
	Funding     *Funding `json:"funding,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Recurring   *bool    `json:"recurring,omitempty"`
	Recipient   string   `json:"recipient,omitempty"`
	// Named amounts retain the source meaning; null means unknown.
	Amounts map[string]*int64 `json:"amounts,omitempty"`
	Dates   map[string]*Date  `json:"dates,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}
type Record struct {
	ID         string          `json:"id"`
	SheetID    int64           `json:"sheetId"`
	Sheet      string          `json:"sheet"`
	Row        int             `json:"row"`
	Domain     string          `json:"domain"`
	OriginalID string          `json:"originalId,omitempty"`
	Raw        map[string]Cell `json:"raw"`
	Normalized Fields          `json:"normalized"`
	Issues     []Issue         `json:"issues"`
}
type Candidate struct {
	RecordID      string   `json:"recordId"`
	OtherRecordID string   `json:"otherRecordId"`
	Rule          string   `json:"rule"`
	Matched       []string `json:"matched"`
	Differences   []string `json:"differences"`
	Decision      string   `json:"decision"`
}
type Metadata struct {
	SourceID          string `json:"sourceId"`
	CapturedAt        string `json:"capturedAt"`
	Locale            string `json:"locale"`
	DeclaredTimeZone  string `json:"declaredTimeZone"`
	ConfirmedTimeZone string `json:"confirmedTimeZone"`
}
type Report struct {
	SchemaVersion string         `json:"schemaVersion"`
	Source        Metadata       `json:"source"`
	PayloadSHA256 string         `json:"payloadSha256"`
	Records       []Record       `json:"records"`
	Candidates    []Candidate    `json:"candidates"`
	Issues        []Issue        `json:"issues"`
	DomainCounts  map[string]int `json:"domainCounts"`
}
