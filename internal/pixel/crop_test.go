package pixel

import (
	"image"
	"image/color"
	"testing"
)

func TestCropReturnsExpectedSize(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))

	got, err := Crop(img, image.Rect(0, 0, 16, 16))
	if err != nil {
		t.Fatalf("Crop() error = %v", err)
	}
	if got.Bounds().Dx() != 16 || got.Bounds().Dy() != 16 {
		t.Fatalf("Crop() size = %dx%d, want 16x16", got.Bounds().Dx(), got.Bounds().Dy())
	}
}

func TestCropRejectsOutOfBounds(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))

	_, err := Crop(img, image.Rect(16, 16, 40, 40))
	if err == nil {
		t.Fatal("Crop() error = nil, want error")
	}
}

func TestCropPreservesPixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}

	got, err := Crop(img, image.Rect(2, 2, 6, 6))
	if err != nil {
		t.Fatalf("Crop() error = %v", err)
	}
	for y := 0; y < got.Bounds().Dy(); y++ {
		for x := 0; x < got.Bounds().Dx(); x++ {
			if got.NRGBAAt(x, y) != (color.NRGBA{R: 10, G: 20, B: 30, A: 255}) {
				t.Fatalf("pixel at %d,%d = %#v, want solid color", x, y, got.NRGBAAt(x, y))
			}
		}
	}
}
