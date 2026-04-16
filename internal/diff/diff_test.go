package diff

import (
	"image"
	"image/color"
	"testing"
)

func TestCompareIdenticalImages(t *testing.T) {
	img := solidImage(8, 8, color.NRGBA{R: 255, A: 255})
	got := Compare(img, img, 0)
	if got.DiffPixels != 0 {
		t.Fatalf("DiffPixels = %d, want 0", got.DiffPixels)
	}
	if !got.BBox.Empty() {
		t.Fatalf("BBox = %v, want empty", got.BBox)
	}
}

func TestCompareDifferentImages(t *testing.T) {
	red := solidImage(4, 4, color.NRGBA{R: 255, A: 255})
	blue := solidImage(4, 4, color.NRGBA{B: 255, A: 255})
	got := Compare(red, blue, 0)
	if got.DiffPixels != 16 || got.TotalPixels != 16 {
		t.Fatalf("Compare() = %+v, want 16/16 diff", got)
	}
	if got.BBox != image.Rect(0, 0, 4, 4) {
		t.Fatalf("BBox = %v, want full image", got.BBox)
	}
}

func TestCompareTolerance(t *testing.T) {
	a := solidImage(2, 2, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	b := solidImage(2, 2, color.NRGBA{R: 105, G: 100, B: 100, A: 255})
	if got := Compare(a, b, 4); got.DiffPixels != 4 {
		t.Fatalf("Compare(..., 4).DiffPixels = %d, want 4", got.DiffPixels)
	}
	if got := Compare(a, b, 5); got.DiffPixels != 0 {
		t.Fatalf("Compare(..., 5).DiffPixels = %d, want 0", got.DiffPixels)
	}
}

func TestDiffImageHighlightsDifferences(t *testing.T) {
	a := solidImage(3, 3, color.NRGBA{R: 255, A: 255})
	b := solidImage(3, 3, color.NRGBA{R: 255, A: 255})
	b.SetNRGBA(1, 1, color.NRGBA{B: 255, A: 255})

	overlay := DiffImage(a, b, 0)
	if got := overlay.NRGBAAt(1, 1); got != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("diff pixel = %+v, want opaque red", got)
	}
	if got := overlay.NRGBAAt(0, 0); got != (color.NRGBA{R: 128, G: 128, B: 128, A: 64}) {
		t.Fatalf("same pixel = %+v, want faint gray", got)
	}
}

func solidImage(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}
