package receiptdrafts

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }
func fixtureFields(account string) Fields {
	return Fields{Seller: ptr("Synthetic store"), IssuedDate: ptr("2026-09-12"), Currency: ptr("RUB"), Total: ptr("180.00"), AccountID: account, Action: "create", Items: []Item{{Name: ptr("Synthetic milk"), Quantity: ptr("2"), UnitPrice: ptr("100.00"), Discount: ptr("20.00"), LineTotal: ptr("180.00")}}}
}
func TestReceiptAmountsDiscountsAndUnknowns(t *testing.T) {
	f := fixtureFields("account")
	v, err := Validate(f)
	if err != nil || v.Event.AmountMinor != 18000 || !v.Event.OccurredAt.IsZero() || v.Event.OccurredDate != "2026-09-12" || v.Receipt.Items[0].Brand != nil {
		t.Fatal(v, err)
	}
	for _, change := range []func(*Fields){func(f *Fields) { f.Total = nil }, func(f *Fields) { f.Items[0].Name = nil }, func(f *Fields) { f.IssuedDate = nil }, func(f *Fields) { f.Total = ptr("160") }, func(f *Fields) { f.Items[0].Quantity = ptr("0") }, func(f *Fields) { f.Items[0].UnitPrice = ptr("100.001") }, func(f *Fields) { f.Currency = ptr("USD") }, func(f *Fields) { f.Items[0].Discount = ptr("99999999999999999999999999999999999999") }} {
		f = fixtureFields("account")
		change(&f)
		if _, err = Validate(f); err == nil {
			t.Fatal("unsafe fields accepted", f)
		}
	}
	for _, s := range []string{"NaN", "1e3", "-1", "1 000", "92233720368547758.08"} {
		if _, err := ParseScaled(s, 2); err == nil {
			t.Fatal(s)
		}
	}
	n, err := ParseScaled("0,125", 3)
	if err != nil || n != 125 {
		t.Fatal(n, err)
	}
}
func syntheticPhoto(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.White)
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestPhotoNormalizationMetadataRemovalAndStorageBoundary(t *testing.T) {
	raw := append(syntheticPhoto(t), []byte("private appended metadata")...)
	normal, err := NormalizePhoto(raw)
	if err != nil || bytes.Contains(normal, []byte("private appended")) {
		t.Fatal(err)
	}
	store, err := NewPhotoStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.Save(normal)
	if err != nil {
		t.Fatal(err)
	}
	data, err := store.Read(id)
	if err != nil || !bytes.Equal(data, normal) {
		t.Fatal(err)
	}
	for _, id := range []string{"../outside", strings.Repeat("a", 47), strings.Repeat("z", 48)} {
		if store.Delete(id) == nil {
			t.Fatal("unsafe deletion path", id)
		}
	}
	if err = store.Delete(id); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Read(id); err == nil {
		t.Fatal("photo retained")
	}
	if store.Delete(id) != nil {
		t.Fatal("deletion replay failed")
	}
	if _, err = NormalizePhoto([]byte("not an image")); err == nil {
		t.Fatal("invalid image accepted")
	}
	if _, err = NewPhotoStore("relative"); err == nil {
		t.Fatal("relative storage accepted")
	}
}
