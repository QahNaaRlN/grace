package documentdata

import (
	"bytes"
	"errors"
	"image"
)

// CheckReadable catches blank/corrupt inputs, not legibility or OCR correctness.
// Individual blank pages remain visible in the complete document for owner review.
func (d Document) CheckReadable() error {
	if e := d.Validate(); e != nil {
		return e
	}
	total, ink := 0, 0
	for _, p := range d.Pages {
		im, _, e := image.Decode(bytes.NewReader(p.Data))
		if e != nil {
			return errors.New("invalid_image")
		}
		b := im.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				rr, g, bb, _ := im.At(x, y).RGBA()
				total++
				if rr < 0xeeee || g < 0xeeee || bb < 0xeeee {
					ink++
				}
			}
		}
	}
	if total == 0 || ink == 0 || ink < total/10000 {
		return errors.New("blank_document")
	}
	return nil
}
