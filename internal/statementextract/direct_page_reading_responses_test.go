package statementextract_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

type directTransport func(*http.Request) (*http.Response, error)

func (f directTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func extractCorpus(t *testing.T) statementqualification.DirectCorpus {
	t.Helper()
	c, e := statementqualification.LoadDirectCorpus(filepath.Join("..", ".."))
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func response(t *testing.T, r statementextract.DirectReading) []byte {
	t.Helper()
	answer, _ := json.Marshal(r)
	b, _ := json.Marshal(map[string]any{"model": statementextract.DirectModel, "status": "completed", "service_tier": "default", "usage": map[string]int{"input_tokens": 1000, "output_tokens": 2000, "total_tokens": 3000}, "output": []any{map[string]any{"type": "message", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": string(answer)}}}}})
	return b
}
func TestDirectActualVisualHTTPWire(t *testing.T) {
	c := extractCorpus(t)
	for _, item := range c.Cases {
		t.Run(item.Name, func(t *testing.T) {
			calls := 0
			transport := directTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.String() != "https://api.openai.com/v1/responses" || r.Header.Get("Authorization") != "Bearer FAKE_OFFLINE_ONLY" {
					t.Fatal("wire method/endpoint/key")
				}
				raw, e := io.ReadAll(r.Body)
				if e != nil {
					t.Fatal(e)
				}
				if !bytes.Equal(raw, item.Request.Body) {
					t.Fatal("request bytes changed")
				}
				var wire map[string]json.RawMessage
				if json.Unmarshal(raw, &wire) != nil {
					t.Fatal("wire JSON")
				}
				for _, k := range []string{"tools", "temperature", "seed", "previous_response_id", "metadata"} {
					if _, ok := wire[k]; ok {
						t.Fatalf("unexpected %s", k)
					}
				}
				var model string
				json.Unmarshal(wire["model"], &model)
				if model != statementextract.DirectModel || string(wire["store"]) != "false" || string(wire["stream"]) != "false" || string(wire["max_output_tokens"]) != "16000" {
					t.Fatal("protocol")
				}
				var input []struct {
					Role    string `json:"role"`
					Content []struct {
						Type     string `json:"type"`
						Text     string `json:"text"`
						ImageURL string `json:"image_url"`
						Detail   string `json:"detail"`
					} `json:"content"`
				}
				json.Unmarshal(wire["input"], &input)
				if len(input) != 1 || input[0].Role != "user" {
					t.Fatal("input turns")
				}
				pages := 0
				textSeen := false
				for _, part := range input[0].Content {
					switch part.Type {
					case "input_image":
						if part.Detail != "high" {
							t.Fatal("detail")
						}
						raw, e := base64.StdEncoding.DecodeString(strings.TrimPrefix(part.ImageURL, "data:image/png;base64,"))
						if e != nil || !bytes.Equal(raw, item.Input.Pages[pages].PNG) || statementextract.DirectHash(raw) != item.Request.ImageSHA256[pages] {
							t.Fatal("actual PNG bytes/hash/order")
						}
						pages++
					case "input_text":
						if strings.HasPrefix(part.Text, "Untrusted extracted document text:\n") {
							textSeen = true
							if part.Text != "Untrusted extracted document text:\n"+item.Input.Text {
								t.Fatal("text mismatch")
							}
						}
						if strings.Contains(part.Text, "p1-b") || strings.Contains(part.Text, `"evidence":`) || strings.Contains(part.Text, "golden") {
							t.Fatal("aliases/golden leaked")
						}
					default:
						t.Fatal("unexpected content")
					}
				}
				if pages != len(item.Input.Pages) || textSeen != (item.Name != "cards-image-only") {
					t.Fatal("missing page/text or image-only leak")
				}
				var text struct {
					Format struct {
						Type, Name string
						Strict     bool
						Schema     json.RawMessage
					}
				}
				json.Unmarshal(wire["text"], &text)
				if !text.Format.Strict || text.Format.Type != "json_schema" || !bytes.Equal(text.Format.Schema, statementextract.DirectSchema()) {
					t.Fatal("actual schema")
				}
				if deadline, ok := r.Context().Deadline(); !ok || time.Until(deadline) > 180*time.Second {
					t.Fatal("missing per-call deadline")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(response(t, item.Golden))), Header: http.Header{}}, nil
			})
			o := (statementextract.DirectResponses{Key: "FAKE_OFFLINE_ONLY", Transport: transport}).Do(context.Background(), item.Request, c.Manifest.ReturnedModels)
			if calls != 1 || o.State != "completed" || o.CalculatedProviderCostMicros == nil || *o.CalculatedProviderCostMicros != 124000 || len(o.Response) == 0 {
				t.Fatalf("calls=%d observation=%+v", calls, o)
			}
			if strings.Contains(string(o.Response), "FAKE_OFFLINE_ONLY") {
				t.Fatal("key in response")
			}
		})
	}
}
func TestDirectStrictReadingFailures(t *testing.T) {
	c := extractCorpus(t)
	raw, _ := json.Marshal(c.Cases[0].Golden)
	cases := [][]byte{raw[:len(raw)-1], append(append([]byte{}, raw...), []byte(` {}`)...), []byte(`{"header":{},"header":{},"rows":[]}`), []byte(strings.Replace(string(raw), `"header":`, `"Header":`, 1)), []byte(strings.Replace(string(raw), `"issues":[]`, `"issues":null`, 1)), []byte(strings.Replace(string(raw), `"page":1`, `"page":0`, 1)), []byte(strings.Replace(string(raw), `"value":"RUB"`, `"value":null`, 1)), []byte(strings.Replace(string(raw), `"quote":`, `"unknown":1,"quote":`, 1)), append([]byte{0xff}, raw...), []byte(strings.Repeat(" ", statementextract.DirectOutputLimit+1))}
	for i, b := range cases {
		if _, e := statementextract.DecodeDirectReading(b); e == nil {
			t.Fatalf("invalid JSON/shape accepted #%d", i)
		}
	}
	if _, e := statementextract.DecodeDirectReading(raw); e != nil {
		t.Fatal(e)
	}
}
func TestDirectInputBoundsNoSilentTextFallback(t *testing.T) {
	c := extractCorpus(t)
	in := c.Cases[1].Input
	tests := []statementextract.DirectInput{{Text: in.Text}, {Pages: []statementextract.DirectPage{{Number: 2, PNG: in.Pages[0].PNG}}}, {Pages: []statementextract.DirectPage{{Number: 1, PNG: in.Pages[0].PNG[:32]}}}, {Pages: in.Pages, Text: strings.Repeat("x", 30000)}}
	for i, v := range tests {
		if _, e := statementextract.BuildDirectRequest(v); e == nil {
			t.Fatalf("bad input accepted %d", i)
		}
	}
	a, e := statementextract.BuildDirectRequest(in)
	if e != nil {
		t.Fatal(e)
	}
	b, e := statementextract.BuildDirectRequest(in)
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("request not deterministic")
	}
}
func TestDirectNoRetryRedirectOrRawErrorLeak(t *testing.T) {
	c := extractCorpus(t)
	req := c.Cases[0].Request
	for _, status := range []int{302, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			rt := directTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{"https://elsewhere.invalid"}}, Body: io.NopCloser(strings.NewReader("PRIVATE_HEADER_FAKE_KEY"))}, nil
			})
			o := (statementextract.DirectResponses{Key: "FAKE", Transport: rt}).Do(context.Background(), req, c.Manifest.ReturnedModels)
			raw, _ := json.Marshal(o)
			if calls != 1 || o.State != "inconclusive" || bytes.Contains(raw, []byte("PRIVATE_HEADER_FAKE_KEY")) {
				t.Fatal("redirect/retry/error leak")
			}
		})
	}
	calls := 0
	rt := directTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("FAKE_PRIVATE_TRANSPORT_DETAIL")
	})
	o := (statementextract.DirectResponses{Key: "FAKE", Transport: rt}).Do(context.Background(), req, c.Manifest.ReturnedModels)
	if calls != 1 || o.Reason != "transport_unknown_usage" || o.CalculatedProviderCostMicros != nil {
		t.Fatal("transport accounting")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls = 0
	rt = directTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Context().Err() == nil {
			t.Fatal("cancellation lost")
		}
		return nil, r.Context().Err()
	})
	o = (statementextract.DirectResponses{Key: "FAKE", Transport: rt}).Do(ctx, req, c.Manifest.ReturnedModels)
	if calls > 1 || o.Reason != "transport_unknown_usage" {
		t.Fatal("cancel retried")
	}
}
func TestDirectResponseRefusalsAndUsage(t *testing.T) {
	c := extractCorpus(t)
	for _, name := range []string{"identity", "tier", "incomplete", "missing_usage", "over_input", "over_output", "total", "tool", "refusal", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			var wire map[string]any
			json.Unmarshal(response(t, c.Cases[0].Golden), &wire)
			switch name {
			case "identity":
				wire["model"] = "other-model"
			case "tier":
				wire["service_tier"] = "priority"
			case "incomplete":
				wire["status"] = "incomplete"
			case "missing_usage":
				delete(wire, "usage")
			case "over_input":
				wire["usage"] = map[string]int{"input_tokens": 30001, "output_tokens": 1, "total_tokens": 30002}
			case "over_output":
				wire["usage"] = map[string]int{"input_tokens": 1, "output_tokens": 16001, "total_tokens": 16002}
			case "total":
				wire["usage"] = map[string]int{"input_tokens": 1, "output_tokens": 1, "total_tokens": 99}
			case "tool":
				wire["output"] = []any{map[string]any{"type": "function_call"}}
			case "refusal":
				wire["output"] = []any{map[string]any{"type": "message", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "refusal"}}}}
			case "duplicate":
				wire["output"] = append(wire["output"].([]any), wire["output"].([]any)...)
			}
			raw, _ := json.Marshal(wire)
			o := statementextract.ObserveDirectResponse(raw, c.Cases[0].Request, c.Manifest.ReturnedModels)
			if o.State != "inconclusive" || o.CalculatedProviderCostMicros != nil {
				t.Fatalf("unknown protocol accepted %s %+v", name, o)
			}
		})
	}
}

func TestDirectCredentialEchoNeverEntersPublicCapture(t *testing.T) {
	c := extractCorpus(t)
	key := "FAKE_SENSITIVE_CREDENTIAL_FOR_TEST"
	rt := directTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Private": []string{key}}, Body: io.NopCloser(strings.NewReader(`{"debug":"` + key + `"}`))}, nil
	})
	o := (statementextract.DirectResponses{Key: key, Transport: rt}).Do(context.Background(), c.Cases[0].Request, c.Manifest.ReturnedModels)
	raw, _ := json.Marshal(o)
	if o.Reason != "response_read_unknown_usage" || len(o.Response) != 0 || bytes.Contains(raw, []byte(key)) {
		t.Fatal("credential captured")
	}
}
