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
				"question": "Which one deterministic argument policy below literally matches `state.utterance`? Infer the requested capability from the utterance itself; this question cannot see any route answer.",
				"rules": []string{
					"Classify only how code may obtain arguments. Do not calculate dates, extract free-form values, or judge whether the request is generally understandable.",
					"`state.reference_time` is context supplied by the application. Its presence does not mean the user stated a date or modifier.",
					"Accounts or balances with no modifier are none.",
					"Current obligations, debts, installments, schedules, due payments, or current overdue state with no user-stated date or filter are reference_time.",
					"Recent, latest, saved, or confirmed receipts with no number, merchant, date, period, or filter are defaults. Words such as recent, latest, last, line items, or positions are not explicit modifiers.",
					"A period since the latest income, latest receipt of income, or latest credit with no named income kind is reference_time. Salary, advance, bonus, pension, or another named income kind is needs_resolution.",
					"A period summary or cash-position request with a stated date, range, named period, relative period, or horizon is needs_resolution. The same capabilities with no period or horizon are missing.",
					"An unsupported write, advice, forecast, OCR request, or a genuinely ambiguous request is not_applicable even if it mentions a date, amount, or other value.",
				},
			},
			Criteria: map[string]string{
				string(ArgumentsNone):            "The utterance asks only for accounts or current account balances. It states no date, filter, account selection, or other modifier, so code must pass exactly an empty object.",
				string(ArgumentsReferenceTime):   "Either: (a) the utterance asks for current obligations, debts, installments, payment schedules, due payments, or current overdue state without a user-stated date or filter; or (b) it asks for the period since the latest generic income, receipt of income, or credit without naming salary, advance, bonus, pension, or another income kind. Code may use only state.reference_time or its local date.",
				string(ArgumentsDefaults):        "The utterance asks for recent, latest, saved, or confirmed receipts or their line items and states no numeric count, merchant, date, period, or filter. Code may omit optional arguments and use the fixed default limit.",
				string(ArgumentsNeedsResolution): "The supported request contains a value code is not allowed to guess or parse here: any date, range, period, or horizon for a period summary or cash position; a user-stated obligation date or filter; a receipt count, merchant, date, period, or filter; or a named income kind such as salary, advance, bonus, or pension.",
				string(ArgumentsMissing):         "The utterance asks for a period summary or cash position but states no period, range, relative period, or horizon at all. Required temporal information is absent.",
				string(ArgumentsNotApplicable):   "The utterance is an unsupported write, advice, forecast, or OCR request, or is genuinely ambiguous about which information is wanted. No tool argument policy applies; ignore any incidental date, amount, or value in such a request.",
			},
		},
	}
}
