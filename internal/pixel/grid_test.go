package pixel

import (
	"image"
	"image/color"
	"testing"
)

func TestGuessGridDetectsFourByOne(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 128, 32))
	for i := 0; i < 4; i++ {
		x0 := i * 32
		fillRect(img, image.Rect(x0, 0, x0+31, 31), color.NRGBA{R: 255, A: 255})
	}

	got := GuessGrid(img)
	if got.Cols != 4 || got.Rows != 1 || got.CellW != 32 || got.CellH != 32 {
		t.Fatalf("GuessGrid() = %+v, want 4x1 cells of 32x32", got)
	}
	if got.Confidence != 1 {
		t.Fatalf("Confidence = %v, want 1", got.Confidence)
	}
}

func TestGuessGridSolidImageHasNoGrid(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	fillRect(img, img.Bounds(), color.NRGBA{R: 255, A: 255})

	got := GuessGrid(img)
	if got.Confidence != 0 {
		t.Fatalf("Confidence = %v, want 0", got.Confidence)
	}
}
