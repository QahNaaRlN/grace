// S1.9 operator-only replay: no provider, meter, SQL or product API changes.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

type projection struct {
	Version                 string                                `json:"version"`
	ExpectedSource          bool                                  `json:"expectedSource"`
	ReportIdentityVerified  bool                                  `json:"reportIdentityVerified"`
	ServiceReasonsValidated bool                                  `json:"serviceReasonsValidated"`
	ArtifactsUnchanged      bool                                  `json:"artifactsUnchanged"`
	Corpus                  []statementqualification.CaseEvidence `json:"corpus"`
	Result                  statementqualification.Result         `json:"result"`
	Parts                   []statementdata.Proposal              `json:"parts"`
}
type description struct {
	Ordinal    int                     `json:"ordinal"`
	Expected   string                  `json:"expected"`
	Actual     string                  `json:"actual"`
	Matched    bool                    `json:"matched"`
	Selections statementdata.Selection `json:"selections"`
}
type headerSelection struct {
	Role       string                  `json:"role"`
	Label      string                  `json:"label"`
	Selections statementdata.Selection `json:"selections"`
}
type output struct {
	Verified           bool                          `json:"verified"`
	ReplayMatchesSaved bool                          `json:"replayMatchesSaved"`
	ProviderCalls      int                           `json:"providerCalls"`
	Result             statementqualification.Result `json:"result"`
	Descriptions       []description                 `json:"descriptions"`
	Labels             []headerSelection             `json:"labels"`
}

func sameResult(a, b statementqualification.Result) bool {
	a.LatencyMS, b.LatencyMS = 0, 0
	return reflect.DeepEqual(a, b)
}
func replay(p projection, d statementdata.Document, fixtureHash string) (output, error) {
	fail := errors.New("s19_replay_unverified")
	out := output{}
	if p.Version != "statement.s19.projection.v1" || !p.ExpectedSource || !p.ReportIdentityVerified || !p.ServiceReasonsValidated || !p.ArtifactsUnchanged || len(p.Corpus) != 3 || len(p.Parts) != 1 || fixtureHash != statementqualification.FixtureHash("columns") || d.Extractor != statementdata.ExtractorVersion {
		return out, fail
	}
	for i, n := range statementqualification.ScreeningNames() {
		c := p.Corpus[i]
		if c.Name != n || c.FixtureHash != statementqualification.FixtureHash(n) {
			return out, fail
		}
	}
	plan, e := statementextract.BuildScreeningPlan(d, statementextract.ScreeningPolicy())
	if e != nil || d.Hash() != p.Corpus[0].DocumentHash || statementextract.Hash(plan) != p.Corpus[0].PlanHash || statementextract.InputBytes(plan.Chunks[0]) != p.Corpus[0].InputBytes {
		return out, fail
	}
	// Validate every locator before any public fixture text can be emitted.
	if d.Validate() != nil {
		return out, fail
	}
	prop, e := statementextract.MergeProposals(plan, p.Parts)
	if e != nil {
		return out, fail
	}
	pages := []int{}
	for _, page := range d.Pages {
		pages = append(pages, page.Number)
	}
	review, e := statementdata.ValidateProposal(d, prop, pages, true, "")
	if e != nil {
		return out, fail
	}
	result := statementqualification.ScoreScreening("columns", review)
	if !sameResult(result, p.Result) {
		return out, fail
	}
	out = output{Verified: true, ReplayMatchesSaved: true, Result: result, Descriptions: []description{}, Labels: []headerSelection{}}
	expected := []string{"SYNTHETIC SHOP", "OWN TRANSFER - REVIEW SEMANTICS"}
	if len(review.Rows) != 2 || len(prop.Rows) != 2 {
		return output{}, fail
	}
	for i, row := range review.Rows {
		out.Descriptions = append(out.Descriptions, description{i + 1, expected[i], row.Description, row.Description == expected[i], prop.Rows[i].Description})
	}
	// Text here comes only from the hash-verified public synthetic PDF.
	spans := map[string]string{}
	for _, span := range d.Spans {
		spans[span.ID] = span.Text
	}
	for _, role := range []string{"openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover"} {
		ids := prop.Header.Labels[role]
		text := []string{}
		for _, id := range ids {
			if spans[id] == "" {
				return output{}, fail
			}
			text = append(text, spans[id])
		}
		out.Labels = append(out.Labels, headerSelection{role, strings.Join(text, " "), ids})
	}
	return out, nil
}
func run() error {
	if len(os.Args) != 4 {
		return errors.New("args")
	}
	raw, e := os.ReadFile(os.Args[1])
	if e != nil || len(raw) > 524288 {
		return errors.New("projection")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var p projection
	if dec.Decode(&p) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return errors.New("projection")
	}
	pdf, e := os.ReadFile(os.Args[2])
	if e != nil {
		return e
	}
	hash := sha256.Sum256(pdf)
	// Reject arbitrary PDFs before handing bytes to the extractor.
	if hex.EncodeToString(hash[:]) != statementqualification.FixtureHash("columns") {
		return errors.New("fixture")
	}
	helper, e := pdfextract.NewClient(os.Args[3])
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	d, e := helper.Extract(ctx, pdf)
	if e != nil {
		return e
	}
	out, e := replay(p, d, hex.EncodeToString(hash[:]))
	if e != nil {
		return e
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
func main() {
	if run() != nil {
		fmt.Println(`{"error":"s19_replay_unverified","verified":false}`)
		os.Exit(1)
	}
}
