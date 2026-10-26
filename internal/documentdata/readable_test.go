package documentdata

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestReadableDocumentRejectsWhiteInputAndTampering(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			im.Set(x, y, color.White)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, im)
	d, e := Photo(b.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	if d.CheckReadable() == nil {
		t.Fatal("white image accepted")
	}
	for y := 10; y < 30; y++ {
		for x := 10; x < 30; x++ {
			im.Set(x, y, color.Black)
		}
	}
	b.Reset()
	_ = png.Encode(&b, im)
	d, e = Photo(b.Bytes())
	if e != nil || d.CheckReadable() != nil {
		t.Fatal("readable control rejected", e)
	}
	d.Pages[0].SHA256 = "bad"
	if d.CheckReadable() == nil {
		t.Fatal("tampered image accepted")
	}
}
