package statementqualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type DirectManifestPage struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
type DirectManifestCase struct {
	Name   string               `json:"name"`
	Rows   int                  `json:"rows"`
	PDF    string               `json:"pdf"`
	Source string               `json:"source"`
	Text   *string              `json:"text"`
	Golden string               `json:"golden"`
	Pages  []DirectManifestPage `json:"pages"`
}
type DirectManifest struct {
	Version          string               `json:"version"`
	PublicReview     string               `json:"publicReview"`
	RendererImage    string               `json:"rendererImage"`
	RendererVersion  string               `json:"rendererVersion"`
	GeneratorVersion string               `json:"generatorVersion"`
	FontSHA256       string               `json:"fontSHA256"`
	Order            []string             `json:"order"`
	Model            string               `json:"model"`
	ReturnedModels   []string             `json:"returnedModels"`
	Files            map[string]string    `json:"files"`
	Cases            []DirectManifestCase `json:"cases"`
	PromptSHA256     string               `json:"promptSHA256"`
	SchemaSHA256     string               `json:"schemaSHA256"`
}
type DirectCase struct {
	Name     string
	Input    statementextract.DirectInput
	Document statementdata.Document
	Golden   statementextract.DirectReading
	Request  statementextract.DirectRequest
}
type DirectCorpus struct {
	Manifest       DirectManifest
	ManifestSHA256 string
	Cases          []DirectCase
}

const DirectCorpusPath = "internal/statementextract/testdata/direct-page-reading/manifest.json"

func directRead(root, path string, limit int64) ([]byte, error) {
	if filepath.IsAbs(path) || filepath.Clean(path) != path || strings.Contains(path, "\\") || !(strings.HasPrefix(path, "internal/statementextract/testdata/direct-page-reading/") || path == "internal/pdfextract/testdata/cards.pdf" || path == "internal/pdfextract/testdata/columns.pdf") {
		return nil, errors.New("corpus_path")
	}
	current := root
	for _, part := range strings.Split(path, "/") {
		current = filepath.Join(current, part)
		s, e := os.Lstat(current)
		if e != nil || s.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("corpus_symlink_or_missing")
		}
	}
	f, e := os.Open(current)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > limit {
		return nil, errors.New("corpus_size")
	}
	return io.ReadAll(io.LimitReader(f, limit+1))
}
func LoadDirectCorpus(root string) (DirectCorpus, error) {
	var out DirectCorpus
	raw, e := directRead(root, DirectCorpusPath, 2<<20)
	if e != nil {
		return out, e
	}
	if e = statementextract.DirectJSON(raw, &out.Manifest); e != nil {
		return out, e
	}
	out.ManifestSHA256 = statementextract.DirectHash(raw)
	m := out.Manifest
	names := []string{"cards", "columns", "cards-image-only", "full32"}
	order := append(append([]string{}, names...), names...)
	if m.Version != statementextract.DirectProfile || m.Model != statementextract.DirectModel || !reflect.DeepEqual(m.ReturnedModels, []string{statementextract.DirectModel}) || !reflect.DeepEqual(m.Order, order) || len(m.Cases) != 4 || m.PublicReview == "" || !strings.HasPrefix(m.RendererImage, "sha256:") || m.RendererVersion != "poppler-25.04.0" || m.PromptSHA256 != statementextract.DirectHash([]byte(statementextract.DirectPrompt)) || m.SchemaSHA256 != statementextract.DirectHash(statementextract.DirectSchema()) {
		return out, errors.New("corpus_identity_or_protocol")
	}
	files := map[string][]byte{}
	for path, hash := range m.Files {
		r, e := directRead(root, path, 10<<20)
		if e != nil || statementextract.DirectHash(r) != hash {
			return out, errors.New("corpus_hash")
		}
		files[path] = r
	}
	for i, c := range m.Cases {
		expectedRows := 2
		expectedPages := 1
		if i == 1 {
			expectedPages = 2
		}
		if i == 3 {
			expectedRows = 32
			expectedPages = 3
		}
		if c.Name != names[i] || c.Rows != expectedRows || len(c.Pages) != expectedPages || (i == 2) != (c.Text == nil) {
			return out, errors.New("corpus_case_shape")
		}
		if !bytes.HasPrefix(files[c.PDF], []byte("%PDF-")) {
			return out, errors.New("corpus_pdf")
		}
		d, e := statementdata.DecodeLayout(bytes.NewReader(files[c.Source]))
		if e != nil || len(d.Pages) != expectedPages {
			return out, errors.New("corpus_source")
		}
		golden, e := statementextract.DecodeDirectReading(files[c.Golden])
		if e != nil || len(golden.Rows) != expectedRows {
			return out, errors.New("corpus_golden")
		}
		item := DirectCase{Name: c.Name, Document: d, Golden: golden}
		if c.Text != nil {
			r, ok := files[*c.Text]
			if !ok || len(r) == 0 {
				return out, errors.New("corpus_text")
			}
			item.Input.Text = string(r)
		}
		for j, p := range c.Pages {
			r, ok := files[p.Path]
			if !ok || statementextract.DirectHash(r) != p.SHA256 || p.Width < 1 || p.Height < 1 || p.Width > 1800 || p.Height > 1800 {
				return out, errors.New("corpus_png")
			}
			cfg, e := png.DecodeConfig(bytes.NewReader(r))
			pdfPage := d.Pages[j]
			if e != nil || cfg.Width != p.Width || cfg.Height != p.Height || math.Abs(float64(cfg.Width)/float64(cfg.Height)-pdfPage.Width/pdfPage.Height) > 0.002 {
				return out, errors.New("corpus_page_dimensions")
			}
			item.Input.Pages = append(item.Input.Pages, statementextract.DirectPage{Number: j + 1, PNG: r})
		}
		item.Request, e = statementextract.BuildDirectRequest(item.Input)
		if e != nil {
			return out, e
		}
		out.Cases = append(out.Cases, item)
	}
	if !reflect.DeepEqual(out.Cases[0].Input.Pages, out.Cases[2].Input.Pages) || !reflect.DeepEqual(out.Cases[0].Golden, out.Cases[2].Golden) {
		return out, errors.New("corpus_image_control")
	}
	return out, nil
}

type DirectCheck struct {
	Field    string                       `json:"field"`
	Row      int                          `json:"row"`
	Passed   bool                         `json:"passed"`
	Expected *string                      `json:"expected"`
	Observed *string                      `json:"observed"`
	Evidence []statementextract.PageQuote `json:"evidence"`
}
type DirectMeasurement struct {
	RecognitionPassed   bool                           `json:"recognitionPassed"`
	SourceBindingPassed bool                           `json:"sourceBindingPassed"`
	ImportReady         bool                           `json:"importReady"`
	Checks              []DirectCheck                  `json:"checks"`
	Binding             statementextract.DirectBinding `json:"binding"`
	LegacyProfile       string                         `json:"legacyProfile"`
	LegacyResult        *Result                        `json:"legacyResult"`
	LegacyChecks        []ScoringCheck                 `json:"legacyChecks"`
}

func MeasureDirect(c DirectCase, answer statementextract.DirectReading) (DirectMeasurement, error) {
	out := DirectMeasurement{RecognitionPassed: true, Checks: []DirectCheck{}, LegacyChecks: []ScoringCheck{}}
	add := func(row int, key string, expected, observed *string, ev []statementextract.PageQuote) {
		passed := reflect.DeepEqual(expected, observed)
		out.Checks = append(out.Checks, DirectCheck{key, row, passed, expected, observed, ev})
		out.RecognitionPassed = out.RecognitionPassed && passed
	}
	scalar := func(s string) *string { return &s }
	compare := func(row int, keys []string, want, got map[string]*statementextract.DirectField) {
		for _, key := range keys {
			var w, g *string
			var ev []statementextract.PageQuote
			if f := want[key]; f != nil {
				w = scalar(statementextract.DirectNormalize(f.Value))
			}
			if f := got[key]; f != nil {
				g = scalar(statementextract.DirectNormalize(f.Value))
				ev = f.Evidence
			}
			add(row, key, w, g, ev)
		}
	}
	compare(0, statementextract.DirectHeaderFields, c.Golden.Header, answer.Header)
	add(0, "rowCount", scalar(fmt.Sprint(len(c.Golden.Rows))), scalar(fmt.Sprint(len(answer.Rows))), nil)
	// Include every expected and every extra row. No averaged or truncated score.
	n := len(c.Golden.Rows)
	if len(answer.Rows) > n {
		n = len(answer.Rows)
	}
	for i := 0; i < n; i++ {
		var want, got map[string]*statementextract.DirectField
		if i < len(c.Golden.Rows) {
			want = c.Golden.Rows[i].Fields
		}
		if i < len(answer.Rows) {
			got = answer.Rows[i].Fields
		}
		compare(i+1, statementextract.DirectRowFields, want, got)
		issues := "missing_row"
		if i < len(answer.Rows) {
			raw, _ := json.Marshal(answer.Rows[i].Issues)
			issues = string(raw)
		}
		add(i+1, "issues", scalar("[]"), scalar(issues), nil)
	}
	binding, e := statementextract.BindDirectReading(c.Document, answer)
	if e != nil {
		return out, e
	}
	out.Binding = binding
	out.SourceBindingPassed = binding.Passed
	if c.Name == "cards" || c.Name == "cards-image-only" || c.Name == "columns" {
		name := c.Name
		if name == "cards-image-only" {
			name = "cards"
		}
		result, checks := ScreeningChecks(name, binding.Review)
		out.LegacyResult = &result
		out.LegacyChecks = checks
		out.LegacyProfile = "unchanged fixed-corpus screening/" + name
	} else {
		out.LegacyProfile = "legacy fixed two-row profile not applicable; all 32 rows checked above; existing core guards below"
	}
	return out, nil
}

type DirectCapture struct {
	Response   json.RawMessage `json:"response"`
	Failure    string          `json:"failure"`
	DurationMS int64           `json:"durationMs"`
}
type DirectAttempt struct {
	Capture     *DirectCapture                      `json:"capture"`
	Ordinal     int                                 `json:"ordinal"`
	Case        string                              `json:"case"`
	State       string                              `json:"state"`
	Observation *statementextract.DirectObservation `json:"observation"`
	Response    json.RawMessage                     `json:"response"`
	Measurement *DirectMeasurement                  `json:"measurement"`
}
type DirectReport struct {
	CostBasis                    string          `json:"costBasis"`
	Profile                      string          `json:"profile"`
	Source                       string          `json:"source"`
	Approval                     string          `json:"approval"`
	ManifestSHA256               string          `json:"manifestSHA256"`
	Decision                     string          `json:"decision"`
	ImportReady                  bool            `json:"importReady"`
	AllowanceState               string          `json:"allowanceState"`
	ReservedMicros               int64           `json:"reservedMicros"`
	CalculatedProviderCostMicros *int64          `json:"calculatedProviderCostMicros"`
	Attempts                     []DirectAttempt `json:"attempts"`
}

// ReplayDirect derives all results from raw public responses and frozen requests,
// rather than trusting a saved decision, observation, score or boolean.
func ReplayDirect(corpus DirectCorpus, source, approval string, responses []DirectCapture) (DirectReport, error) {
	out := DirectReport{CostBasis: "usage at price ceilings: 14 input / 55 output USD per million; calculated upper bound, not invoice or production settlement", Profile: statementextract.DirectProfile, Source: source, Approval: approval, ManifestSHA256: corpus.ManifestSHA256, Decision: "INCONCLUSIVE", ReservedMicros: statementextract.DirectCampaignCeilingMicros, AllowanceState: "uncertain", Attempts: []DirectAttempt{}}
	if len(source) != 40 || strings.Trim(source, "0123456789abcdef") != "" || approval == "" || len(approval) > 80 || len(responses) > 8 {
		return out, errors.New("direct_report_identity")
	}
	stopped := false
	total := int64(0)
	known := true
	complete := 0
	for i, name := range corpus.Manifest.Order {
		a := DirectAttempt{Ordinal: i + 1, Case: name, State: "not_sent"}
		if i < len(responses) {
			if stopped {
				return out, errors.New("direct_response_after_stop")
			}
			c := corpus.Cases[i%4]
			capture := responses[i]
			// Elapsed capture includes decoding/cleanup; HTTP has its own 180s deadline.
			if capture.DurationMS < 0 || capture.DurationMS > 1500000 {
				return out, errors.New("direct_duration")
			}
			a.Capture = &capture
			a.Response = capture.Response
			o := statementextract.ObserveDirectResponse(capture.Response, c.Request, corpus.Manifest.ReturnedModels)
			if capture.Failure != "" {
				switch capture.Failure {
				case "transport_unknown_usage", "http_refusal_unknown_usage", "response_read_unknown_usage":
				default:
					return out, errors.New("direct_transport_code")
				}
				if len(capture.Response) > 0 {
					return out, errors.New("direct_transport_payload")
				}
				o = statementextract.DirectObservation{RequestSHA256: c.Request.SHA256, State: "inconclusive", Reason: capture.Failure}
			}
			o.DurationMS = capture.DurationMS
			a.Observation = &o
			a.State = o.State
			if o.CalculatedProviderCostMicros == nil {
				known = false
			} else {
				total += *o.CalculatedProviderCostMicros
			}
			if len(o.Answer) > 0 {
				reading, e := statementextract.DecodeDirectReading(o.Answer)
				if e != nil {
					return out, e
				}
				m, e := MeasureDirect(c, reading)
				if e != nil {
					return out, e
				}
				a.Measurement = &m
				if !m.RecognitionPassed || !m.SourceBindingPassed {
					out.Decision = "REJECT"
					stopped = true
				}
			}
			if o.State != "completed" {
				stopped = true
			} else {
				complete++
			}
		} else {
			stopped = true
		}
		out.Attempts = append(out.Attempts, a)
	}
	if known && len(responses) > 0 {
		out.CalculatedProviderCostMicros = &total
		out.AllowanceState = "calculated_usage_known"
	}
	if complete == 8 && !stopped {
		out.Decision = "KEEP_FOR_FURTHER_REVIEW"
	}
	return out, nil
}
func VerifyDirectReport(corpus DirectCorpus, raw []byte, source, approval string) (DirectReport, error) {
	var given DirectReport
	if e := statementextract.DirectJSON(raw, &given); e != nil {
		return given, e
	}
	if given.Source != source || given.Approval != approval {
		return given, errors.New("direct_report_binding")
	}
	responses := []DirectCapture{}
	for i := range given.Attempts {
		a := &given.Attempts[i]
		if string(a.Response) == "null" {
			a.Response = nil
		}
		if a.Capture != nil && string(a.Capture.Response) == "null" {
			a.Capture.Response = nil
		}
	}
	for _, a := range given.Attempts {
		if a.State != "not_sent" {
			if a.Capture == nil {
				return given, errors.New("direct_response_missing")
			}
			responses = append(responses, *a.Capture)
		} else if a.Response != nil || a.Capture != nil {
			return given, errors.New("direct_not_sent_payload")
		}
	}
	replay, e := ReplayDirect(corpus, source, approval, responses)
	if e != nil {
		return given, e
	}
	expected, _ := json.Marshal(replay)
	actual, _ := json.Marshal(given)
	if !bytes.Equal(expected, actual) {
		return given, errors.New("direct_report_replay_mismatch")
	}
	return replay, nil
}
