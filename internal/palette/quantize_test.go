package palette

import (
	"image"
	"image/color"
	"testing"
)

func TestExtractExactColors(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}}
	for x, c := range colors {
		img.SetNRGBA(x, 0, c)
	}

	got := Extract(img, 4)
	if len(got) != 4 {
		t.Fatalf("len(Extract()) = %d, want 4", len(got))
	}
}

func TestExtractDeterministic(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 1))
	for x := 0; x < 8; x++ {
		img.SetNRGBA(x, 0, color.NRGBA{R: uint8(x * 20), G: uint8(255 - x*20), B: uint8(x * 10), A: 255})
	}
	first := Extract(img, 4)
	second := Extract(img, 4)
	if len(first) != len(second) {
		t.Fatalf("len(first) = %d, len(second) = %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("palette mismatch at %d: %#v != %#v", i, first[i], second[i])
		}
	}
}

func TestExtractIgnoresTransparentPixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 10, B: 10, A: 0})
	img.SetNRGBA(1, 0, color.NRGBA{R: 10, G: 200, B: 10, A: 255})
	got := Extract(img, 2)
	if len(got) != 1 {
		t.Fatalf("len(Extract()) = %d, want 1", len(got))
	}
	if got[0].R != 10 || got[0].G != 200 || got[0].B != 10 {
		t.Fatalf("Extract()[0] = %#v, want visible pixel color", got[0])
	}
}
