package documentdata

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	_ "image/png"
)

const MaxPages = 4
const MaxPageBytes = 8 << 20
const MaxBundleBytes = 26 << 20

type Page struct {
	Number      int    `json:"number"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	SHA256      string `json:"sha256"`
	ContentType string `json:"contentType"`
	Data        []byte `json:"data,omitempty"`
}
type Document struct {
	Kind         string `json:"kind"`
	SourceSHA256 string `json:"sourceSHA256"`
	Pages        []Page `json:"pages"`
	Text         string `json:"text"`
	Original     []byte `json:"original,omitempty"`
}

func Hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func Digest(v any) string    { b, _ := json.Marshal(v); return Hash(b) }
func NewPage(number int, raw []byte) (Page, error) {
	p := Page{Number: number, SHA256: Hash(raw), Data: raw}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(raw))
	if e != nil || len(raw) > MaxPageBytes || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 2048 || cfg.Height > 2048 {
		return p, errors.New("invalid_image")
	}
	if format != "png" && format != "jpeg" {
		return p, errors.New("invalid_image")
	}
	p.Width, p.Height = cfg.Width, cfg.Height
	p.ContentType = "image/" + format
	if _, _, e = image.Decode(bytes.NewReader(raw)); e != nil {
		return p, errors.New("invalid_image")
	}
	return p, nil
}
func Photo(raw []byte) (Document, error) {
	normalized, e := NormalizePhoto(raw)
	if e != nil {
		return Document{}, e
	}
	page, e := NewPage(1, normalized)
	if e != nil {
		return Document{}, e
	}
	d := Document{Kind: "image", SourceSHA256: Hash(raw), Pages: []Page{page}, Original: raw}
	return d, d.Validate()
}
func (d Document) Validate() error {
	if (d.Kind != "pdf" && d.Kind != "image") || len(d.SourceSHA256) != 64 || len(d.Pages) < 1 || len(d.Pages) > MaxPages || len(d.Text) > 1<<20 || len(d.Original) > 10<<20 {
		return errors.New("document_limit")
	}
	if len(d.Original) > 0 && Hash(d.Original) != d.SourceSHA256 {
		return errors.New("invalid_image")
	}
	total := 0
	for i, p := range d.Pages {
		total += len(p.Data)
		if p.Number != i+1 || len(p.Data) == 0 || total > MaxPageBytes || p.Width < 1 || p.Height < 1 || p.Width > 2048 || p.Height > 2048 || p.SHA256 != Hash(p.Data) {
			return errors.New("document_limit")
		}
		cfg, format, e := image.DecodeConfig(bytes.NewReader(p.Data))
		if e != nil || cfg.Width != p.Width || cfg.Height != p.Height || (format != "png" && format != "jpeg") || p.ContentType != "image/"+format {
			return errors.New("invalid_image")
		}
		if d.Kind == "pdf" && (p.Width > 1800 || p.Height > 1800 || format != "png") {
			return errors.New("document_limit")
		}
		if _, _, e = image.Decode(bytes.NewReader(p.Data)); e != nil {
			return errors.New("invalid_image")
		}
	}
	return nil
}
func (d Document) Manifest() Document {
	out := d
	out.Original = nil
	out.Text = ""
	out.Pages = append([]Page{}, d.Pages...)
	for i := range out.Pages {
		out.Pages[i].Data = nil
	}
	return out
}
