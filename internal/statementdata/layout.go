package statementdata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"
)

func finite(n float64) bool { return !math.IsInf(n, 0) && !math.IsNaN(n) }
func validBox(b Box, p Page) bool {
	return finite(b.XMin) && finite(b.YMin) && finite(b.XMax) && finite(b.YMax) && b.XMin >= 0 && b.YMin >= 0 && b.XMax > b.XMin && b.YMax > b.YMin && b.XMax <= p.Width && b.YMax <= p.Height
}

// DecodeLayout consumes Poppler bbox-layout XML with a hard byte cap. XML
// directives are data; encoding/xml never fetches a DTD or expands entities.
func DecodeLayout(reader io.Reader) (Document, error)       { return decodeLayout(reader, false) }
func DecodeVisualLayout(reader io.Reader) (Document, error) { return decodeLayout(reader, true) }
func decodeLayout(reader io.Reader, allowEmpty bool) (Document, error) {
	d := Document{Extractor: ExtractorVersion, Pages: []Page{}, Spans: []Span{}}
	limited := &io.LimitedReader{R: reader, N: MaxXMLBytes + 1}
	dec := xml.NewDecoder(limited)
	var page *Page
	block, line, word, totalText := 0, 0, 0, 0
	for {
		token, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Document{}, errors.New("invalid_pdf_layout")
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "page":
				if page != nil || len(d.Pages) >= MaxPages {
					return Document{}, errors.New("pdf_page_limit")
				}
				var attrs struct {
					Width  float64 `xml:"width,attr"`
					Height float64 `xml:"height,attr"`
				}
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "width":
						if _, e := fmt.Sscan(a.Value, &attrs.Width); e != nil {
							return Document{}, errors.New("invalid_pdf_layout")
						}
					case "height":
						if _, e := fmt.Sscan(a.Value, &attrs.Height); e != nil {
							return Document{}, errors.New("invalid_pdf_layout")
						}
					}
				}
				p := Page{len(d.Pages) + 1, attrs.Width, attrs.Height}
				if !finite(p.Width) || !finite(p.Height) || p.Width <= 0 || p.Height <= 0 || p.Width > 20000 || p.Height > 20000 {
					return Document{}, errors.New("invalid_pdf_layout")
				}
				d.Pages = append(d.Pages, p)
				page = &d.Pages[len(d.Pages)-1]
				block = 0
				line = 0
				word = 0
			case "block":
				block++
				line = 0
			case "line":
				line++
				word = 0
			case "word":
				if page == nil || block == 0 || line == 0 || len(d.Spans) >= MaxSpans {
					return Document{}, errors.New("pdf_span_limit")
				}
				var v struct {
					XMin float64 `xml:"xMin,attr"`
					YMin float64 `xml:"yMin,attr"`
					XMax float64 `xml:"xMax,attr"`
					YMax float64 `xml:"yMax,attr"`
					Text string  `xml:",chardata"`
				}
				if e := dec.DecodeElement(&v, &t); e != nil {
					return Document{}, errors.New("invalid_pdf_layout")
				}
				b := Box{v.XMin, v.YMin, v.XMax, v.YMax}
				text := strings.TrimSpace(v.Text)
				if !validBox(b, *page) || text == "" || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
					return Document{}, errors.New("invalid_pdf_layout")
				}
				totalText += len(text) + 1
				if totalText > MaxTextBytes {
					return Document{}, errors.New("pdf_text_limit")
				}
				word++
				d.Spans = append(d.Spans, Span{fmt.Sprintf("p%d-b%d-l%d-w%d", page.Number, block, line, word), page.Number, block, line, word, b, text})
			}
		case xml.EndElement:
			if t.Name.Local == "page" {
				page = nil
			}
		}
	}
	if limited.N <= 0 {
		return Document{}, errors.New("pdf_text_limit")
	}
	if page != nil || len(d.Pages) == 0 || (!allowEmpty && len(d.Spans) == 0) {
		return Document{}, errors.New("pdf_text_required")
	}
	if err := d.validate(!allowEmpty); err != nil {
		return Document{}, err
	}
	return d, nil
}
func (d Document) Validate() error             { return d.validate(true) }
func (d Document) ValidateVisualLayout() error { return d.validate(false) }
func (d Document) validate(requireText bool) error {
	if d.Extractor != ExtractorVersion || len(d.Pages) < 1 || len(d.Pages) > MaxPages || (requireText && len(d.Spans) < 1) || len(d.Spans) > MaxSpans {
		return errors.New("invalid_pdf_layout")
	}
	for i, p := range d.Pages {
		if p.Number != i+1 || !finite(p.Width) || !finite(p.Height) || p.Width <= 0 || p.Height <= 0 || p.Width > 20000 || p.Height > 20000 {
			return errors.New("invalid_pdf_layout")
		}
	}
	pageWords := map[int]int{}
	seen := map[string]bool{}
	total := 0
	previousPage := 0
	for _, s := range d.Spans {
		if s.ID == "" || len(s.ID) > 100 || seen[s.ID] || s.Page < previousPage || s.Page < 1 || s.Page > len(d.Pages) || s.Block < 1 || s.Line < 1 || s.Word < 1 || !validBox(s.Box, d.Pages[s.Page-1]) || strings.TrimSpace(s.Text) == "" || !utf8.ValidString(s.Text) || strings.ContainsRune(s.Text, 0) {
			return errors.New("invalid_pdf_layout")
		}
		pageWords[s.Page]++
		seen[s.ID] = true
		previousPage = s.Page
		total += len(s.Text) + 1
	}
	for _, page := range d.Pages {
		if requireText && pageWords[page.Number] == 0 {
			return errors.New("pdf_text_required")
		}
	}
	if total > MaxTextBytes {
		return errors.New("pdf_text_limit")
	}
	return nil
}
func (d Document) Hash() string {
	raw, _ := json.Marshal(d)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

// Ground copies literal source text in source order. Numeric/date callers also
// require one line, so crossing columns or pages cannot manufacture a number.
func (d Document) Ground(role string, ids Selection, oneLine bool) (Evidence, error) {
	positions := make(map[string]int, len(d.Spans))
	for i, s := range d.Spans {
		positions[s.ID] = i
	}
	return ground(d.Spans, positions, role, ids, oneLine)
}
func ground(spans []Span, positions map[string]int, role string, ids Selection, oneLine bool) (Evidence, error) {
	out := Evidence{Role: role, Spans: []Span{}}
	if len(ids) == 0 || len(ids) > 2048 {
		return out, errors.New("missing_evidence")
	}
	previous := -1
	var parts []string
	for _, id := range ids {
		pos, ok := positions[id]
		if !ok || pos <= previous {
			return out, errors.New("invalid_span_reference")
		}
		s := spans[pos]
		if len(out.Spans) > 0 {
			first := out.Spans[0]
			last := out.Spans[len(out.Spans)-1]
			if oneLine && (s.Page != first.Page || s.Block != first.Block || s.Line != first.Line || s.Word != last.Word+1) {
				return out, errors.New("noncontiguous_evidence")
			}
		}
		out.Spans = append(out.Spans, s)
		parts = append(parts, s.Text)
		previous = pos
	}
	out.Literal = strings.Join(parts, " ")
	return out, nil
}
