package pixel

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPNGMissingFile(t *testing.T) {
	_, err := LoadPNG(filepath.Join(t.TempDir(), "missing.png"))
	if err == nil {
		t.Fatal("LoadPNG() error = nil, want non-nil")
	}
}

func TestLoadPNGValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "one.png")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: 12, G: 34, B: 56, A: 255})
	writePNG(t, path, img)

	got, err := LoadPNG(path)
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	if got.Bounds().Dx() != 1 || got.Bounds().Dy() != 1 {
		t.Fatalf("bounds = %v, want 1x1", got.Bounds())
	}
	r, g, b, a := got.At(0, 0).RGBA()
	if r>>8 != 12 || g>>8 != 34 || b>>8 != 56 || a>>8 != 255 {
		t.Fatalf("pixel = (%d,%d,%d,%d), want (12,34,56,255)", r>>8, g>>8, b>>8, a>>8)
	}
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("png.Encode(%q) error = %v", path, err)
	}
}
