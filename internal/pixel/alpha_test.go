package pixel

import (
	"flag"
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden test files")

func TestThresholdAlphaBelowThresholdBecomesTransparent(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 20, G: 30, B: 40, A: 64})

	got := ThresholdAlpha(img, 0, 128)
	if got.NRGBAAt(0, 0).A != 0 {
		t.Fatalf("alpha = %d, want 0", got.NRGBAAt(0, 0).A)
	}
}

func TestThresholdAlphaAboveThresholdPreserved(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	want := color.NRGBA{R: 20, G: 30, B: 40, A: 200}
	img.SetNRGBA(0, 0, want)

	got := ThresholdAlpha(img, 0, 128)
	if got.NRGBAAt(0, 0) != want {
		t.Fatalf("pixel = %#v, want %#v", got.NRGBAAt(0, 0), want)
	}
}

func TestCountFractional(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 5, 1))
	for x := 0; x < 5; x++ {
		img.SetNRGBA(x, 0, color.NRGBA{R: uint8(x), A: uint8(10 + x)})
	}
	if got := CountFractional(img); got != 5 {
		t.Fatalf("CountFractional() = %d, want 5", got)
	}
}

func TestThresholdAlphaGolden(t *testing.T) {
	ensureSnapFixtures(t)
	inputPath := filepath.Join(repoRoot(t), "testdata", "input", "snap", "aa_knight.png")
	goldenPath := filepath.Join(repoRoot(t), "testdata", "golden", "snap", "aa_knight_thresh.png")

	img, err := LoadPNG(inputPath)
	if err != nil {
		t.Fatalf("LoadPNG(%q) error = %v", inputPath, err)
	}
	got := ThresholdAlpha(img, 0, 128)
	assertPNGEqualToFile(t, got, goldenPath)
}
