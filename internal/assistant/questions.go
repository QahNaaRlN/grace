package assistant

const (
	QuestionRoute     = "q1"
	QuestionArguments = "q2"
)

type RoutingQuestion struct {
	Type         string            `json:"type"`
	Instructions any               `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

func RoutingQuestions() map[string]RoutingQuestion {
	return map[string]RoutingQuestion{
		QuestionRoute: {
			Type: "choice",
			Instructions: map[string]any{
				"question": "Which single Grace read-only route best matches state.utterance?",
				"rules":    []string{"Choose only one provided route.", "Never treat a write as read-only.", "Do not invent dates, identifiers, limits, financial facts, or unsupported capabilities.", "Choose clarify for a supported but genuinely ambiguous request; choose unsupported when no read-only route can answer."},
			},
			Criteria: map[string]string{
				string(RouteGetAccounts):          "List Grace accounts or their current deterministic balances.",
				string(RouteGetPeriodSummary):     "Summarize actual income, spending, refunds, debt payments, transfers, balances, or categories for a period.",
				string(RouteGetCashPosition):      "Show liquid balances less known obligations through a horizon.",
				string(RouteGetObligations):       "List obligations, debts, installments, schedules, due payments, or their current state.",
				string(RouteGetReceipts):          "List recent confirmed receipts and their ordered line items.",
				string(RouteGetPeriodSinceIncome): "Summarize actual cash flow from the latest confirmed income, optionally naming an income category such as salary or advance.",
				string(RouteClarify):              "A supported request is genuinely ambiguous between routes or does not identify what information is wanted.",
				string(RouteUnsupported):          "Creating, changing, deleting, recommending, forecasting, OCR, or anything unavailable through the six read-only routes.",
			},
		},
		QuestionArguments: {
			Type: "choice",
			Instructions: map[string]any{
				"question": "How may deterministic code obtain arguments for the read-only capability implied by state.utterance? Judge independently of the route answer.",
				"rules":    []string{"Do not calculate or extract arbitrary typed values.", "An explicitly named date, range, limit, category, or other modifier needs resolution even if understandable.", "reference_time is allowed only when the current request can safely use state.reference_time unchanged or its local calendar date."},
			},
			Criteria: map[string]string{
				string(ArgumentsNone):            "The accounts request requires exactly an empty object.",
				string(ArgumentsReferenceTime):   "Current obligations or latest-income-period can safely use state.reference_time without another user value.",
				string(ArgumentsDefaults):        "Recent receipts can safely use the deterministic default limit because no limit or filter was requested.",
				string(ArgumentsNeedsResolution): "The user stated or implied a date, range, horizon, limit, named income category, filter, or modifier that must be resolved before execution.",
				string(ArgumentsMissing):         "The requested period-sensitive capability lacks required temporal information.",
				string(ArgumentsNotApplicable):   "The utterance is clarify or unsupported rather than an executable tool request.",
			},
		},
	}
}
