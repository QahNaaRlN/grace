package jeveval

const (
	questionTool      = "tool"
	questionPeriod    = "period_shape"
	questionArguments = "argument_presence"
)

func Questions() map[string]Question {
	return map[string]Question{
		questionTool: {
			Type: "choice",
			Instructions: map[string]any{
				"question": "Which single Grace read-only route best handles `state.utterance`? Choose clarify when a supported request lacks information required to execute safely. Choose unsupported when none of the five capabilities can answer it.",
				"rules": []string{
					"Select only from the provided routes.",
					"Do not invent dates, account data, receipts, obligations, or calculations.",
					"A request to change financial data is unsupported.",
				},
			},
			Criteria: map[string]string{
				string(ToolGetAccounts):      "List Grace accounts or current balances.",
				string(ToolGetPeriodSummary): "Summarize income, purchases, refunds, debt payments, transfers, balances, or expense categories for a stated period.",
				string(ToolGetCashPosition):  "Show liquid money remaining after known obligations through a stated horizon date or period.",
				string(ToolGetObligations):   "List debts, installments, obligations, schedules, due payments, or their status.",
				string(ToolGetReceipts):      "List recent confirmed receipts and their line items.",
				string(ToolClarify):          "The request maps to a supported capability but omits information required to execute it safely or is genuinely ambiguous between supported capabilities.",
				string(ToolUnsupported):      "The request cannot be answered by the five read-only capabilities, including any request to create, edit, delete, recommend, or analyze unavailable details.",
			},
		},
		questionPeriod: {
			Type: "choice",
			Instructions: map[string]any{
				"question":       "How is the financial time period or horizon expressed in `state.utterance`? Read only what is stated; do not calculate concrete dates.",
				"reference_date": "`state.reference_date` is context for recognizing relative language, not permission to perform calendar arithmetic.",
			},
			Criteria: map[string]string{
				string(PeriodCalendar):      "A named calendar period such as this month, September, last week, or year 2025.",
				string(PeriodRelative):      "A period relative to a financial event or moving horizon, such as after salary, before payday, through next Friday, or until the next payment.",
				string(PeriodExplicitRange): "Both explicit range boundaries are stated, for example from 1 September through 30 September.",
				string(PeriodMissing):       "A period-sensitive request is made but no usable period or horizon is stated.",
				string(PeriodNotApplicable): "The requested capability does not need a period or horizon, or the request is unsupported for reasons unrelated to a missing period.",
			},
		},
		questionArguments: {
			Type:         "choice",
			Instructions: "Does `state.utterance` state enough temporal information for the period-sensitive read-only capability it requests? Judge presence only; do not calculate dates or assume unstated values.",
			Criteria: map[string]string{
				string(ArgumentsSufficient):    "A period summary or cash-position request includes a usable calendar period, explicit range, relative financial period, or horizon.",
				string(ArgumentsMissing):       "A period summary or cash-position request lacks a usable period or horizon and needs clarification.",
				string(ArgumentsNotApplicable): "The request is for accounts, obligations, receipts, or an unsupported action where temporal arguments are not required for routing.",
			},
		},
	}
}

func requestFor(evaluationCase EvaluationCase, model string) Request {
	return Request{
		State: State{
			Utterance:     evaluationCase.Utterance,
			Language:      evaluationCase.Language,
			ReferenceDate: evaluationCase.ReferenceDate,
		},
		Model:     model,
		Questions: Questions(),
	}
}
