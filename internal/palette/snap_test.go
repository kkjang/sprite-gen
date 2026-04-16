package palette

import (
	"image"
	"image/color"
	"testing"
)

func TestSnapNearestColor(t *testing.T) {
	pal := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}}
	got := Snap(color.NRGBA{R: 5, G: 10, B: 240, A: 255}, pal)
	if got.B != 255 || got.R != 0 || got.G != 0 {
		t.Fatalf("Snap() = %#v, want blue", got)
	}
}

func TestApplyIdempotentOnPaletteColors(t *testing.T) {
	pal := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}}
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, pal[0])
	img.SetNRGBA(1, 0, pal[1])
	got := Apply(img, pal, false)
	for x := 0; x < 2; x++ {
		if got.NRGBAAt(x, 0) != img.NRGBAAt(x, 0) {
			t.Fatalf("pixel %d = %#v, want %#v", x, got.NRGBAAt(x, 0), img.NRGBAAt(x, 0))
		}
	}
}

func TestApplyPreservesAlphaAndPaletteMembership(t *testing.T) {
	pal := []color.NRGBA{{R: 255, A: 255}, {B: 255, A: 255}}
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 240, G: 10, B: 10, A: 128})
	img.SetNRGBA(1, 0, color.NRGBA{R: 10, G: 10, B: 240, A: 255})
	got := Apply(img, pal, false)
	if got.NRGBAAt(0, 0).A != 128 {
		t.Fatalf("alpha = %d, want 128", got.NRGBAAt(0, 0).A)
	}
	for x := 0; x < 2; x++ {
		c := got.NRGBAAt(x, 0)
		if !(c.R == 255 && c.G == 0 && c.B == 0) && !(c.R == 0 && c.G == 0 && c.B == 255) {
			t.Fatalf("pixel %d = %#v, want palette color", x, c)
		}
	}
}
