// Package documentqualification measures public documents without importing finances.
package documentqualification

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

const Version = "document-reading.spec0056-q.v1"
const NormalizerToolchain = "go1.24.0"
const CorpusPath = "internal/documentqualification/testdata/corpus.json"
const MaximumMicros int64 = 2000000
const MaximumCallMicros int64 = 250000

// Observation bounds are not spending permission: preserve honest upstream overruns.
const MaximumObservedMicros int64 = (1 << 60) - 1
const ReportLimit = 32 << 20

func IDs() []string {
	return []string{"cards", "columns", "cards-photo", "full32-native", "full32-scan", "receipt-discount", "receipt-pdf", "receipt-unreadable"}
}

type Case struct {
	ID            string                          `json:"id"`
	SourcePath    string                          `json:"sourcePath"`
	Document      documentdata.Document           `json:"document"`
	Native        *statementdata.Document         `json:"native,omitempty"`
	Oracle        statementdata.Document          `json:"oracle"`
	Golden        *statementextract.DirectReading `json:"golden,omitempty"`
	ReceiptGolden json.RawMessage                 `json:"receiptGolden,omitempty"`
	Readable      bool                            `json:"readable"`
}
type Corpus struct {
	Normalizer string            `json:"normalizer"`
	Version    string            `json:"version"`
	Renderer   string            `json:"renderer"`
	Files      map[string]string `json:"files"`
	Cases      []Case            `json:"cases"`
}

func (c Corpus) Hash() string { return documentdata.Digest(c) }
func (c Corpus) Validate() error {
	fail := errors.New("corpus_invalid")
	if c.Normalizer != NormalizerToolchain || c.Version != Version || c.Renderer != "poppler-25.04.0" || len(c.Cases) != 8 || len(c.Files) < 8 || len(c.Files) > 64 {
		return fail
	}
	for path, hash := range c.Files {
		if !publicPath(path) || !hash64(hash) {
			return fail
		}
	}
	for i, v := range c.Cases {
		if v.ID != IDs()[i] || v.Document.Validate() != nil || len(v.Document.Original) != 0 || c.Files[v.SourcePath] != v.Document.SourceSHA256 {
			return fail
		}
		pages := []int{1, 2, 1, 3, 3, 1, 1, 1}[i]
		if len(v.Document.Pages) != pages || v.Readable != (i != 7) {
			return fail
		}
		native := i == 0 || i == 1 || i == 3
		if native != (v.Native != nil) || (!native && v.Document.Text != "") || (native && (v.Native.Validate() != nil || v.Document.Text == "")) {
			return fail
		}
		if i < 5 {
			if v.Golden == nil || len(v.ReceiptGolden) > 0 || v.Oracle.Validate() != nil || len(v.Golden.Rows) != []int{2, 2, 2, 32, 32}[i] {
				return fail
			}
		} else {
			if v.Golden != nil || !json.Valid(v.ReceiptGolden) {
				return fail
			}
		}
		if (i == 2 || i == 5 || i == 7) != (v.Document.Kind == "image") {
			return fail
		}
	}
	return nil
}
func hash64(s string) bool   { return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == "" }
func source40(s string) bool { return len(s) == 40 && strings.Trim(s, "0123456789abcdef") == "" }
func publicPath(s string) bool {
	return !filepath.IsAbs(s) && filepath.Clean(s) == s && !strings.Contains(s, "\\") && (strings.HasPrefix(s, "internal/pdfextract/testdata/") || strings.HasPrefix(s, "internal/statementextract/testdata/direct-page-reading/") || strings.HasPrefix(s, "internal/documentqualification/testdata/") || strings.HasPrefix(s, "testdata/ai-evaluation/"))
}
func ReadPublic(root, path string, limit int64) ([]byte, error) {
	if !publicPath(path) {
		return nil, errors.New("corpus_path")
	}
	current := root
	for _, part := range strings.Split(path, "/") {
		current = filepath.Join(current, part)
		s, e := os.Lstat(current)
		if e != nil || s.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("corpus_path")
		}
	}
	return readBounded(current, limit)
}
func readBounded(path string, limit int64) ([]byte, error) {
	s, e := os.Lstat(path)
	if e != nil || !s.Mode().IsRegular() || s.Size() > limit {
		return nil, errors.New("file_invalid")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, errors.New("file_invalid")
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, limit+1))
}
func Load(root string) (Corpus, error) {
	raw, e := ReadPublic(root, CorpusPath, 8<<20)
	if e != nil {
		return Corpus{}, e
	}
	c, e := DecodeCorpus(raw)
	if e != nil {
		return c, e
	}
	for path, hash := range c.Files {
		raw, e := ReadPublic(root, path, 10<<20)
		if e != nil || documentdata.Hash(raw) != hash {
			return Corpus{}, errors.New("corpus_source_changed")
		}
	}
	return c, nil
}
func DecodeCorpus(raw []byte) (Corpus, error) {
	var c Corpus
	if len(raw) > 8<<20 || artifactJSON(raw, &c) != nil || c.Validate() != nil {
		return c, errors.New("corpus_invalid")
	}
	return c, nil
}

type Renderer interface {
	Render(context.Context, []byte) (pdfextract.Rendered, error)
}

// Prepare runs locally against the real networkless helper, never a provider.
func Prepare(ctx context.Context, root string, renderer Renderer) (Corpus, error) {
	out := Corpus{Normalizer: NormalizerToolchain, Version: Version, Renderer: "poppler-25.04.0", Files: map[string]string{}, Cases: []Case{}}
	if runtime.Version() != NormalizerToolchain {
		return out, errors.New("corpus_normalizer_toolchain")
	}
	direct, e := statementqualification.LoadDirectCorpus(root)
	if e != nil {
		return out, e
	}
	for path, hash := range direct.Manifest.Files {
		out.Files[path] = hash
	}
	data, e := ReadPublic(root, "testdata/ai-evaluation/synthetic.v1.json", 1<<20)
	if e != nil {
		return out, e
	}
	out.Files["testdata/ai-evaluation/synthetic.v1.json"] = documentdata.Hash(data)
	var receipts struct {
		Receipts []struct {
			ID       string          `json:"id"`
			Expected json.RawMessage `json:"expected"`
		} `json:"receipts"`
	}
	if json.Unmarshal(data, &receipts) != nil {
		return out, errors.New("corpus_receipts")
	}
	for i, id := range IDs() {
		v := Case{ID: id, Readable: i != 7}
		switch i {
		case 0:
			v.SourcePath = "internal/pdfextract/testdata/cards.pdf"
		case 1:
			v.SourcePath = "internal/pdfextract/testdata/columns.pdf"
		case 2:
			v.SourcePath = "internal/statementextract/testdata/direct-page-reading/cards/page-1.png"
		case 3:
			v.SourcePath = "internal/statementextract/testdata/direct-page-reading/full32.pdf"
		case 4:
			v.SourcePath = "internal/pdfextract/testdata/full32-scan.pdf"
		case 5:
			v.SourcePath = "testdata/ai-evaluation/images/receipt-01.png"
		case 6:
			v.SourcePath = "internal/documentqualification/testdata/receipt-02.pdf"
		case 7:
			v.SourcePath = "testdata/ai-evaluation/images/receipt-19.png"
		}
		raw, e := ReadPublic(root, v.SourcePath, 10<<20)
		if e != nil {
			return out, e
		}
		out.Files[v.SourcePath] = documentdata.Hash(raw)
		if i == 2 || i == 5 || i == 7 {
			normalized, e := documentdata.NormalizePhoto(raw)
			if e != nil {
				return out, e
			}
			page, e := documentdata.NewPage(1, normalized)
			if e != nil {
				return out, e
			}
			v.Document = documentdata.Document{Kind: "image", SourceSHA256: documentdata.Hash(raw), Pages: []documentdata.Page{page}}
		} else {
			rendered, e := renderer.Render(ctx, raw)
			if e != nil {
				return out, e
			}
			v.Document = rendered.Document
			v.Native = rendered.Layout
			v.Document.Original = nil
		}
		if i < 5 {
			j := []int{0, 1, 2, 3, 3}[i]
			v.Golden = &direct.Cases[j].Golden
			v.Oracle = direct.Cases[j].Document
		} else {
			rid := []string{"receipt-01", "receipt-02", "receipt-19"}[i-5]
			for _, item := range receipts.Receipts {
				if item.ID == rid {
					v.ReceiptGolden = item.Expected
				}
			}
		}
		out.Cases = append(out.Cases, v)
	}
	return out, out.Validate()
}

type CaseManifest struct {
	ID           string                `json:"id"`
	SourcePath   string                `json:"sourcePath"`
	Document     documentdata.Document `json:"document"`
	Mode         string                `json:"mode"`
	PageCount    int                   `json:"pageCount"`
	Rows         int                   `json:"rows"`
	Items        int                   `json:"items"`
	Readable     bool                  `json:"readable"`
	TextSHA256   string                `json:"textSHA256"`
	GoldenSHA256 string                `json:"goldenSHA256"`
	NativeSHA256 string                `json:"nativeSHA256"`
}
type Manifest struct {
	Cases                []CaseManifest             `json:"cases"`
	Normalizer           string                     `json:"normalizer"`
	Renderer             string                     `json:"renderer"`
	Version              string                     `json:"version"`
	Source               string                     `json:"source"`
	CorpusSHA256         string                     `json:"corpusSHA256"`
	ProtocolSHA256       string                     `json:"protocolSHA256"`
	Order                []string                   `json:"order"`
	MaximumMicros        int64                      `json:"maximumMicros"`
	MaximumCallMicros    int64                      `json:"maximumCallMicros"`
	MaximumPOST          int                        `json:"maximumPOST"`
	MaximumGET           int                        `json:"maximumGET"`
	Candidate            *assistant.GroupMembership `json:"candidate"`
	ExecutionAuthorized  bool                       `json:"executionAuthorized"`
	ActivationAuthorized bool                       `json:"activationAuthorized"`
}

func NewManifest(c Corpus, source string, candidate *assistant.GroupMembership) Manifest {
	prompt, schema := assistant.ReceiptDocumentProtocol()
	protocol := documentdata.Digest([]string{statementextract.DocumentPrompt, documentdata.Hash(statementdata.VisualSchema()), prompt, documentdata.Hash(schema)})
	cases := []CaseManifest{}
	for _, v := range c.Cases {
		item := CaseManifest{ID: v.ID, SourcePath: v.SourcePath, Document: v.Document.Manifest(), Mode: "photo", PageCount: len(v.Document.Pages), Readable: v.Readable, TextSHA256: documentdata.Hash([]byte(v.Document.Text))}
		if v.Document.Kind == "pdf" {
			item.Mode = "image_only_pdf"
		}
		if v.Native != nil {
			item.Mode = "native_pdf"
			item.NativeSHA256 = documentdata.Digest(v.Native)
		}
		if v.Golden != nil {
			item.Rows = len(v.Golden.Rows)
			item.GoldenSHA256 = documentdata.Digest(v.Golden)
		} else {
			item.GoldenSHA256 = documentdata.Digest(json.RawMessage(v.ReceiptGolden))
			var fields struct {
				Items []any `json:"items"`
			}
			_ = json.Unmarshal(v.ReceiptGolden, &fields)
			item.Items = len(fields.Items)
		}
		cases = append(cases, item)
	}
	return Manifest{Normalizer: c.Normalizer, Cases: cases, Renderer: c.Renderer, Version: Version, Source: source, CorpusSHA256: c.Hash(), ProtocolSHA256: protocol, Order: IDs(), MaximumMicros: MaximumMicros, MaximumCallMicros: MaximumCallMicros, MaximumPOST: 8, MaximumGET: 6, Candidate: candidate}
}
func (m Manifest) Hash() string { return documentdata.Digest(m) }
func (m Manifest) Validate(c Corpus, source string, requireCandidate bool) error {
	expected := NewManifest(c, source, m.Candidate)
	if !reflect.DeepEqual(m, expected) || c.Validate() != nil || (requireCandidate && (!source40(source) || m.Candidate == nil)) {
		return errors.New("manifest_invalid")
	}
	if m.Candidate != nil {
		v := m.Candidate
		if (v.RequestFeeEvidence != "" && (v.RequestFeeEvidence != assistant.RequestFeeRouterCeilingOnly || v.Policy.RequestFeeMicros <= 0)) || v.Qualification != "" || !v.QualifiedUntil.IsZero() || v.Priority != 0 || v.MaxImages != 4 || v.MaxImageSide < 1800 || v.MaxImageSide > 2048 || v.Region != "eu" || !strings.HasSuffix(v.Policy.Provider, "/eu") || v.Policy.Selection != nil || !v.Policy.Vision || !v.Policy.VerifiedAt.IsZero() {
			return errors.New("candidate_invalid")
		}
	}
	return nil
}
