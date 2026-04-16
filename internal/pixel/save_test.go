package pixel

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

func TestSavePNGRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "img.png")
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(1, 1, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	if err := SavePNG(path, img); err != nil {
		t.Fatalf("SavePNG() error = %v", err)
	}
	got, err := LoadPNG(path)
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	if got.Bounds() != img.Bounds() {
		t.Fatalf("bounds = %v, want %v", got.Bounds(), img.Bounds())
	}
}
