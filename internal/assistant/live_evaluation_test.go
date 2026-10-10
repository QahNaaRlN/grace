//go:build liveeval

package assistant

// This opt-in test is excluded from normal CI. It runs only the checked-in
// synthetic corpus, uses persistent PostgreSQL metering, and writes no ledger.
import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/receiptdrafts"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type liveResult struct {
	ID               string               `json:"id"`
	Total            *string              `json:"total"`
	Currency         *string              `json:"currency"`
	MandatoryUnknown bool                 `json:"mandatoryUnknown"`
	Error            string               `json:"error"`
	LatencyMS        int64                `json:"latencyMs"`
	InputTokens      int                  `json:"inputTokens"`
	OutputTokens     int                  `json:"outputTokens"`
	CostMicros       string               `json:"costMicros"`
	ItemErrors       []string             `json:"itemErrors"`
	Fields           receiptdrafts.Fields `json:"fields"`
	GenerationID     string               `json:"generationId"`
	HTTPStatus       int                  `json:"httpStatus"`
}
type capturedHTTP struct {
	inner  HTTPDoer
	status int
	body   []byte
}

func (c *capturedHTTP) Do(req *http.Request) (*http.Response, error) {
	c.status = 0
	c.body = nil
	response, err := c.inner.Do(req)
	if err != nil {
		return response, err
	}
	c.status = response.StatusCode
	data, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(data))
	c.body = data
	return response, readErr
}

type capturedMeter struct {
	*UsageRepository
	response *LLMResponse
}

func (m *capturedMeter) Settle(ctx context.Context, id string, response *LLMResponse) error {
	m.response = response
	return m.UsageRepository.Settle(ctx, id, response)
}
func TestLiveSyntheticEvaluation(t *testing.T) {
	if os.Getenv("GRACE_LIVE_EVAL") != "synthetic-only" {
		t.Skip("explicit synthetic-only opt-in required")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if !strings.Contains(url, "@127.0.0.1:") {
		t.Fatal("local isolated database required")
	}
	key, err := os.ReadFile(os.Getenv("GRACE_EVAL_KEY_FILE"))
	if err != nil {
		t.Fatal("key file unavailable")
	}
	provider, err := NewOpenRouterProvider(strings.TrimSpace(string(key)), nil)
	if err != nil {
		t.Fatal("invalid key")
	}
	capture := &capturedHTTP{inner: provider.client}
	provider.client = capture
	bg := context.Background()
	pool, err := pgxpool.New(bg, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var allowance int64
	if err = pool.QueryRow(bg, `SELECT allowance_micros FROM ai_experiment_budget`).Scan(&allowance); err != nil || allowance > 250000 || allowance < 1 {
		t.Fatal("evaluation allowance must be 1..250000 microUSD")
	}
	raw, err := os.ReadFile(os.Getenv("GRACE_EVAL_POLICY_FILE"))
	if err != nil {
		t.Fatal(err)
	}
	var policies []ModelPolicy
	if json.Unmarshal(raw, &policies) != nil || len(policies) != 2 {
		t.Fatal("two verified policies required")
	}
	root := filepath.Join("..", "..")
	raw, err = os.ReadFile(filepath.Join(root, "testdata", "ai-evaluation", "synthetic.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Synthetic bool `json:"synthetic"`
		Receipts  []struct {
			ID       string               `json:"id"`
			Readable bool                 `json:"readable"`
			Expected receiptdrafts.Fields `json:"expected"`
		} `json:"receipts"`
		Questions []struct{ ID, Question, Month, Scenario string } `json:"questions"`
	}
	if json.Unmarshal(raw, &corpus) != nil || !corpus.Synthetic || len(corpus.Receipts) != 20 || len(corpus.Questions) != 20 {
		t.Fatal("invalid corpus")
	}
	for _, policy := range policies {
		if !policy.Valid(time.Now()) || !policy.Vision {
			t.Fatal("stale or non-vision policy")
		}
		var owner string
		if err = pool.QueryRow(bg, `INSERT INTO users(display_name,role) VALUES('Synthetic live evaluation','member') RETURNING id::text`).Scan(&owner); err != nil {
			t.Fatal(err)
		}
		ctx := access.WithPrincipal(bg, access.Principal{ID: owner})
		consents := NewConversationRepository(pool)
		if err = consents.Consent(ctx, "receipt", ConsentHash("receipt", policy), true); err != nil {
			t.Fatal(err)
		}
		meter := &capturedMeter{UsageRepository: NewUsageRepository(pool)}
		extractor := ReceiptExtractor{Policy: policy, Provider: provider, Meter: meter, Consents: consents}
		report := struct {
			Model    string           `json:"model"`
			Provider string           `json:"provider"`
			Receipts []liveResult     `json:"receipts"`
			Analysis []map[string]any `json:"analysis"`
		}{Model: policy.Model, Provider: policy.Provider, Receipts: []liveResult{}, Analysis: []map[string]any{}}
		output := filepath.Join(os.Getenv("GRACE_EVAL_OUTPUT_DIR"), strings.ReplaceAll(policy.Model, "/", "-")+".json")
		if previous, readErr := os.ReadFile(output); readErr == nil {
			if json.Unmarshal(previous, &report) != nil || report.Model != policy.Model {
				t.Fatal("invalid saved evaluation")
			}
		}
		save := func() {
			b, _ := json.MarshalIndent(report, "", "  ")
			if err = os.WriteFile(output, b, 0600); err != nil {
				t.Fatal(err)
			}
		}
		for _, fixture := range corpus.Receipts {
			if os.Getenv("GRACE_EVAL_SMOKE") == "true" && fixture.ID != "receipt-01" {
				continue
			}
			done := false
			for _, result := range report.Receipts {
				if result.ID == fixture.ID {
					done = true
					break
				}
			}
			if done {
				continue
			}
			image, readErr := os.ReadFile(filepath.Join(root, "testdata", "ai-evaluation", "images", fixture.ID+".png"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			photo, normErr := receiptdrafts.NormalizePhoto(image)
			if normErr != nil {
				t.Fatal(normErr)
			}
			meter.response = nil
			start := time.Now()
			out, callErr := extractor.Extract(ctx, photo)
			result := liveResult{ID: fixture.ID, Total: out.Fields.Total, Currency: out.Fields.Currency, LatencyMS: time.Since(start).Milliseconds(), CostMicros: "0", ItemErrors: []string{}, Fields: out.Fields}
			if meter.response != nil {
				result.InputTokens = meter.response.PromptTokens
				result.OutputTokens = meter.response.CompletionTokens
				result.CostMicros = strconv.FormatInt(meter.response.CostMicros, 10)
				result.GenerationID = meter.response.ID
			}
			result.HTTPStatus = capture.status
			if callErr != nil {
				result.Error = callErr.Error()
			}
			result.MandatoryUnknown = out.Fields.Total == nil || out.Fields.Currency == nil || out.Fields.Seller == nil || (out.Fields.IssuedDate == nil && out.Fields.IssuedAt == nil)
			if len(out.Fields.Items) != len(fixture.Expected.Items) {
				result.ItemErrors = append(result.ItemErrors, "item_count")
			}
			report.Receipts = append(report.Receipts, result)
			save()
			t.Logf("%s %s error=%s", policy.Model, fixture.ID, result.Error)
			if callErr != nil {
				if writeErr := os.WriteFile(filepath.Join(os.Getenv("GRACE_EVAL_OUTPUT_DIR"), fixture.ID+"-error.private.json"), capture.body, 0600); writeErr != nil {
					t.Fatal(writeErr)
				}
				t.Fatal("live evaluation stopped: review persisted usage before another paid call")
			}
		}
		facts := AnalysisFacts{Month: "2026-09", From: "2026-09-01T00:00:00+05:00", To: "2026-10-01T00:00:00+05:00", CapturedAt: time.Now().UTC().Format(time.RFC3339), Coverage: "confirmed_only_coverage_unknown", Facts: []AnalysisFact{{ID: "currency-0-purchases", Label: "Подтверждённые покупки", ValueMinor: "120000", Currency: "RUB"}, {ID: "currency-0-food", Label: "Продукты", ValueMinor: "80000", Currency: "RUB"}}}
		for _, question := range corpus.Questions {
			if os.Getenv("GRACE_EVAL_SMOKE") == "true" && question.ID != "question-1" {
				continue
			}
			done := false
			for _, result := range report.Analysis {
				if result["id"] == question.ID {
					done = true
					break
				}
			}
			if done {
				continue
			}
			questionFacts := facts
			if question.Scenario == "empty" {
				questionFacts.Facts = []AnalysisFact{}
			}
			if question.Scenario == "multiple_currencies" {
				questionFacts.Facts = append(append([]AnalysisFact{}, facts.Facts...), AnalysisFact{ID: "currency-1-purchases", Label: "Покупки USD", ValueMinor: "1000", Currency: "USD"})
			}
			request, requestErr := AnalysisRequest(policy, question.Question, questionFacts)
			if requestErr != nil {
				t.Fatal(requestErr)
			}
			start := time.Now()
			out, callErr := MeteredCompletion(ctx, provider, meter, request, "analysis")
			valid := false
			message := ""
			if callErr == nil {
				_, validationErr := ValidateInterpretations(out.Content, questionFacts)
				valid = validationErr == nil
				if validationErr != nil {
					message = validationErr.Error()
				}
			} else {
				message = callErr.Error()
			}
			report.Analysis = append(report.Analysis, map[string]any{"id": question.ID, "valid": valid, "error": message, "latencyMs": time.Since(start).Milliseconds(), "generationId": out.ID, "costMicros": out.CostMicros})
			save()
			t.Logf("%s %s valid=%t", policy.Model, question.ID, valid)
			if callErr != nil {
				t.Fatal("live evaluation stopped: review persisted usage")
			}
		}
	}
}
