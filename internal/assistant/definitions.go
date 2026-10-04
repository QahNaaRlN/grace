package assistant

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
	ReadOnly    bool        `json:"readOnly"`
}

type InputSchema struct {
	Schema               string                    `json:"$schema"`
	Type                 string                    `json:"type"`
	Properties           map[string]PropertySchema `json:"properties"`
	Required             []string                  `json:"required"`
	AdditionalProperties bool                      `json:"additionalProperties"`
}

type PropertySchema struct {
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	Description string `json:"description"`
	Minimum     *int   `json:"minimum,omitempty"`
	Maximum     *int   `json:"maximum,omitempty"`
}

const schemaDraft = "https://json-schema.org/draft/2020-12/schema"

func Definitions() []ToolDefinition {
	minLimit, maxLimit := 1, 100
	return []ToolDefinition{
		{
			Name: "get_accounts", Description: "Return all Grace accounts with balances computed from postings.", ReadOnly: true,
			InputSchema: objectSchema(map[string]PropertySchema{}, nil),
		},
		{
			Name: "get_period_summary", Description: "Return actual cash flow and boundary balances for a half-open period, separated by currency.", ReadOnly: true,
			InputSchema: objectSchema(map[string]PropertySchema{
				"from": {Type: "string", Format: "date-time", Description: "Inclusive RFC 3339 boundary with an explicit UTC offset."},
				"to":   {Type: "string", Format: "date-time", Description: "Exclusive RFC 3339 boundary with an explicit UTC offset."},
			}, []string{"from", "to"}),
		},
		{
			Name: "get_cash_position", Description: "Return current liquid balances less outstanding obligations through a calendar date.", ReadOnly: true,
			InputSchema: objectSchema(map[string]PropertySchema{
				"asOf":    {Type: "string", Format: "date", Description: "Date used to classify overdue and upcoming payments; defaults to current UTC date."},
				"through": {Type: "string", Format: "date", Description: "Inclusive obligation horizon."},
			}, []string{"through"}),
		},
		{
			Name: "get_obligations", Description: "Return fixed obligations and canonical derived schedule state.", ReadOnly: true,
			InputSchema: objectSchema(map[string]PropertySchema{
				"asOf": {Type: "string", Format: "date", Description: "Date used to derive schedule state; defaults to current UTC date."},
			}, nil),
		},
		{
			Name: "get_receipts", Description: "Return recent confirmed receipts with ordered items.", ReadOnly: true,
			InputSchema: objectSchema(map[string]PropertySchema{
				"limit": {Type: "integer", Description: "Maximum number of receipts.", Minimum: &minLimit, Maximum: &maxLimit},
			}, nil),
		},
		{
			Name: "get_period_since_income", Description: "Return the confirmed income anchor and actual cash flow since it, separated by currency.", ReadOnly: true,
			InputSchema: objectSchema(map[string]PropertySchema{
				"asOf":       {Type: "string", Format: "date-time", Description: "Exclusive RFC 3339 boundary with an explicit UTC offset."},
				"categoryId": {Type: "string", Format: "uuid", Description: "Optional exact Grace income-category identifier."},
			}, []string{"asOf"}),
		},
	}
}

func objectSchema(properties map[string]PropertySchema, required []string) InputSchema {
	if required == nil {
		required = make([]string, 0)
	}
	return InputSchema{
		Schema: schemaDraft, Type: "object", Properties: properties,
		Required: required, AdditionalProperties: false,
	}
}
