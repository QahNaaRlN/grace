package documentdata

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func TestPhotoFormatsOrientationAndPrivateNormalization(t *testing.T) {
	bitmap := image.NewRGBA(image.Rect(0, 0, 32, 48))
	var jpg, pngBytes bytes.Buffer
	if e := jpeg.Encode(&jpg, bitmap, nil); e != nil {
		t.Fatal(e)
	}
	if e := png.Encode(&pngBytes, bitmap); e != nil {
		t.Fatal(e)
	}
	webp, e := os.ReadFile("testdata/colors.webp")
	if e != nil {
		t.Fatal(e)
	}
	// Actual EXIF orientation=6, followed by the original JPEG sections.
	exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	segment := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(segment[2:], uint16(len(exif)+2))
	oriented := append([]byte{0xff, 0xd8}, segment...)
	oriented = append(oriented, exif...)
	oriented = append(oriented, jpg.Bytes()[2:]...)
	for _, tc := range []struct {
		name          string
		raw           []byte
		width, height int
	}{{"png", pngBytes.Bytes(), 32, 48}, {"jpeg", jpg.Bytes(), 32, 48}, {"webp", webp, 150, 100}, {"exif", oriented, 48, 32}} {
		t.Run(tc.name, func(t *testing.T) {
			d, e := Photo(tc.raw)
			if e != nil {
				t.Fatal(e)
			}
			p := d.Pages[0]
			if p.Width != tc.width || p.Height != tc.height || p.ContentType != "image/jpeg" || bytes.Contains(p.Data, []byte("Exif")) {
				t.Fatal("bad normalization", p.Width, p.Height)
			}
			if d.SourceSHA256 != Hash(tc.raw) {
				t.Fatal("source hash changed")
			}
		})
	}
	for _, raw := range [][]byte{[]byte("invalid"), make([]byte, MaxPhotoBytes+1), jpg.Bytes()[:12]} {
		if _, e = Photo(raw); e == nil {
			t.Fatal("malformed/oversize accepted")
		}
	}
}
func TestDocumentRejectsPageTruncationAndTampering(t *testing.T) {
	raw, e := os.ReadFile("testdata/colors.webp")
	if e != nil {
		t.Fatal(e)
	}
	doc, e := Photo(raw)
	if e != nil {
		t.Fatal(e)
	}
	doc.Pages[0].SHA256 = "forged"
	if doc.Validate() == nil {
		t.Fatal("forged hash accepted")
	}
	doc, e = Photo(raw)
	if e != nil {
		t.Fatal(e)
	}
	for len(doc.Pages) < 5 {
		p := doc.Pages[0]
		p.Number = len(doc.Pages) + 1
		doc.Pages = append(doc.Pages, p)
	}
	if doc.Validate() == nil {
		t.Fatal("5-page document accepted")
	}
}
