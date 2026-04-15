package pixel

import (
	"image"
	"image/color"
	"testing"
)

func TestComputeStatsSolidImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	fillRect(img, img.Bounds(), color.NRGBA{R: 20, G: 30, B: 40, A: 255})

	got := ComputeStats(img)
	if got.UniqueColors != 1 {
		t.Fatalf("UniqueColors = %d, want 1", got.UniqueColors)
	}
	if got.AAScore != 0 {
		t.Fatalf("AAScore = %v, want 0", got.AAScore)
	}
}

func TestComputeStatsAAScore(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	fillRect(img, img.Bounds(), color.NRGBA{R: 10, G: 10, B: 10, A: 255})
	for i := 0; i < 10; i++ {
		img.SetNRGBA(i, i, color.NRGBA{R: 10, G: 10, B: 10, A: 128})
	}

	got := ComputeStats(img)
	if got.AAScore <= 0.05 {
		t.Fatalf("AAScore = %v, want > 0.05", got.AAScore)
	}
}
