package pixel

import (
	"image"
	"image/color"
	"testing"
)

func TestBBoxOpaqueImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	fillRect(img, image.Rect(0, 0, 8, 6), color.NRGBA{A: 255})

	got := BBox(img, 0)
	if got != img.Bounds() {
		t.Fatalf("BBox() = %v, want %v", got, img.Bounds())
	}
}

func TestBBoxCenteredSquare(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(img, image.Rect(8, 8, 24, 24), color.NRGBA{A: 255})

	got := BBox(img, 0)
	want := image.Rect(8, 8, 24, 24)
	if got != want {
		t.Fatalf("BBox() = %v, want %v", got, want)
	}
}

func fillRect(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
