// ai-eval validates and renders synthetic evaluation inputs. It cannot connect
// to a provider or publish financial data. Live results are scored separately.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type receiptCase struct {
	ID       string `json:"id"`
	Readable bool   `json:"readable"`
	Text     string `json:"text"`
	Expected struct {
		Total    *string `json:"total"`
		Currency *string `json:"currency"`
	} `json:"expected"`
}
type dataset struct {
	Version      string            `json:"version"`
	Synthetic    bool              `json:"synthetic"`
	Receipts     []receiptCase     `json:"receipts"`
	Questions    []json.RawMessage `json:"questions"`
	ProductLines []json.RawMessage `json:"productLines"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	input := flag.String("dataset", "testdata/ai-evaluation/synthetic.v1.json", "synthetic corpus")
	output := flag.String("render", "", "render synthetic PNGs into this directory")
	results := flag.String("score", "", "score a model's recorded result JSON; never performs inference")
	flag.Parse()
	raw, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	var data dataset
	if json.Unmarshal(raw, &data) != nil || !data.Synthetic || len(data.Receipts) < 20 || len(data.Questions) < 20 || len(data.ProductLines) < 30 {
		return fmt.Errorf("invalid synthetic evaluation corpus")
	}
	ids := map[string]bool{}
	for _, r := range data.Receipts {
		if r.ID == "" || strings.ContainsAny(r.ID, "/\\.") || ids[r.ID] || (r.Readable && (r.Expected.Total == nil || r.Expected.Currency == nil)) {
			return fmt.Errorf("invalid receipt fixture")
		}
		ids[r.ID] = true
	}
	if *output != "" {
		if err = os.MkdirAll(*output, 0755); err != nil {
			return err
		}
		for _, r := range data.Receipts {
			img := image.NewRGBA(image.Rect(0, 0, 600, 400))
			for y := 0; y < 400; y++ {
				for x := 0; x < 600; x++ {
					img.Set(x, y, color.White)
				}
			}
			d := font.Drawer{Dst: img, Src: image.NewUniform(color.Black), Face: basicfont.Face7x13}
			for n, line := range strings.Split(r.Text, "\n") {
				d.Dot = fixed.P(30, 35+25*n)
				d.DrawString(line)
			}
			f, err := os.Create(filepath.Join(*output, r.ID+".png"))
			if err != nil {
				return err
			}
			err = png.Encode(f, img)
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	if *results != "" {
		return score(data, *results)
	}
	fmt.Printf("validated %s: receipts=%d questions=%d product-lines=%d; live inference not performed\n", data.Version, len(data.Receipts), len(data.Questions), len(data.ProductLines))
	return nil
}

type result struct {
	ID               string   `json:"id"`
	Total            *string  `json:"total"`
	Currency         *string  `json:"currency"`
	MandatoryUnknown bool     `json:"mandatoryUnknown"`
	Error            string   `json:"error"`
	LatencyMS        int64    `json:"latencyMs"`
	InputTokens      int      `json:"inputTokens"`
	OutputTokens     int      `json:"outputTokens"`
	CostMicros       string   `json:"costMicros"`
	ItemErrors       []string `json:"itemErrors"`
}

func score(data dataset, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var recorded struct {
		Model    string   `json:"model"`
		Provider string   `json:"provider"`
		Receipts []result `json:"receipts"`
	}
	if json.Unmarshal(raw, &recorded) != nil || recorded.Model == "" || recorded.Provider == "" || len(recorded.Receipts) != len(data.Receipts) {
		return fmt.Errorf("incomplete recorded model results")
	}
	byID := map[string]result{}
	for _, r := range recorded.Receipts {
		if _, ok := byID[r.ID]; ok {
			return fmt.Errorf("duplicate result")
		}
		if r.LatencyMS < 0 || r.InputTokens < 0 || r.OutputTokens < 0 || r.CostMicros == "" {
			return fmt.Errorf("missing evaluation metering")
		}
		byID[r.ID] = r
	}
	readable, exact, blocked, unknown := 0, 0, 0, 0
	for _, c := range data.Receipts {
		r, ok := byID[c.ID]
		if !ok {
			return fmt.Errorf("missing case %s", c.ID)
		}
		if c.Readable {
			readable++
			if r.Error == "" && r.Total != nil && r.Currency != nil && *r.Total == *c.Expected.Total && *r.Currency == *c.Expected.Currency {
				exact++
			}
		} else {
			unknown++
			if r.MandatoryUnknown || r.Error != "" {
				blocked++
			}
		}
	}
	report := map[string]any{"model": recorded.Model, "provider": recorded.Provider, "readableCases": readable, "exactTotalCurrency": exact, "readableAccuracy": float64(exact) / float64(readable), "unknownCases": unknown, "blockedUnknownCases": blocked, "receipts": recorded.Receipts, "acceptancePassed": exact*100 >= readable*95 && blocked == unknown, "scope": "Recorded synthetic evaluation, not a guarantee for new receipts"}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
