package statementextract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

// Research-only S1.19 identity. Product Messages/Plan/consent never select it.
const ClockCandidateVersion = "statement-span-mapping.v13.clock-selection-candidate"
const ClockBasePromptSHA = "31c0496472f618aa9622918ab0013bbf87e8b5a59903bfd8fcbe40954a123819"
const ClockCandidatePromptSHA = "c417773a3ff5fd8c855b0701ca8be30606f07e910e0dff47eb2972a4cc18a187"
const ClockCardsDocumentHash = "ff6f0157ec84d773ed788efd67f4635e111425b5146ee2624e6e45aab538a4e9"
const ClockAppendix = `CLOCK SELECTION PROCEDURE: For each record, first locate the printed operation date and booking date separately. Then, before emitting JSON, inspect the adjacent lines in that same field and record for a printed clock. When both date and clock are printed, select their aliases together in the corresponding date array, in source order; a date-only selection is incomplete in that case. Preserve the printed precision. If no clock is printed, keep the field date-only. Never borrow a clock from another record or field, invent a clock or timezone, or copy operation evidence into an unprinted booking field. Perform this check silently and return only the existing JSON object.`

type ClockRequest struct {
	Variant    string
	Plan       Plan
	Messages   []map[string]string
	Schema     json.RawMessage
	Hash       string
	PromptSHA  string
	InputBytes int
}

func PromptSHA(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// BuildClockRequest accepts source data, not an arbitrary caller-manufactured plan.
// Only the research operator binds its document to the fixed shipped cards PDF.
func BuildClockRequest(d statementdata.Document, variant string) (ClockRequest, error) {
	fail := errors.New("clock_request_invalid")
	if (variant != "base" && variant != "candidate") || PromptSHA(SystemPrompt) != ClockBasePromptSHA || PromptSHA(SystemPrompt+"\n\n"+ClockAppendix) != ClockCandidatePromptSHA {
		return ClockRequest{}, fail
	}
	p, e := BuildComparisonPlan(d, ComparisonCandidates()[0].Policy)
	if e != nil || len(p.Chunks) != 1 {
		return ClockRequest{}, fail
	}
	m := Messages(p.Chunks[0])
	if variant == "candidate" {
		m[0]["content"] += "\n\n" + ClockAppendix
	}
	schema := RequestSchema(p.Chunks[0])
	raw, e := json.Marshal(m)
	size := len(raw) + len(schema)
	if e != nil || size > MaxInputBytes {
		return ClockRequest{}, errors.New("clock_request_limit")
	}
	request := struct {
		Policy   Policy
		Messages []map[string]string
		Schema   any
	}{p.Policy, m, schema}
	return ClockRequest{variant, p, m, schema, Hash(request), PromptSHA(m[0]["content"]), size}, nil
}
