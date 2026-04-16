package pixel

import (
	"image"
	"image/color"
	"testing"
)

func TestAlphaMaskThreshold(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{A: 127})
	img.SetNRGBA(1, 0, color.NRGBA{A: 128})

	mask := AlphaMask(img, 128)
	if got := mask.AlphaAt(0, 0).A; got != 0 {
		t.Fatalf("AlphaMask() alpha at 0,0 = %d, want 0", got)
	}
	if got := mask.AlphaAt(1, 0).A; got != 255 {
		t.Fatalf("AlphaMask() alpha at 1,0 = %d, want 255", got)
	}
}

func TestMorphErodeShrinksSquare(t *testing.T) {
	mask := image.NewAlpha(image.Rect(0, 0, 10, 10))
	fillAlpha(mask, image.Rect(1, 1, 9, 9), 255)

	got := MorphErode(mask, 1)
	if bbox := alphaBBox(got); bbox != image.Rect(2, 2, 8, 8) {
		t.Fatalf("MorphErode() bbox = %v, want %v", bbox, image.Rect(2, 2, 8, 8))
	}
}

func TestMorphDilateGrowsSinglePixel(t *testing.T) {
	mask := image.NewAlpha(image.Rect(0, 0, 5, 5))
	mask.SetAlpha(2, 2, color.Alpha{A: 255})

	got := MorphDilate(mask, 1)
	if bbox := alphaBBox(got); bbox != image.Rect(1, 1, 4, 4) {
		t.Fatalf("MorphDilate() bbox = %v, want %v", bbox, image.Rect(1, 1, 4, 4))
	}
	if count := countAlpha(got); count != 9 {
		t.Fatalf("MorphDilate() foreground count = %d, want 9", count)
	}
}

func TestMorphDilateThenErodeIsLossyAtEdges(t *testing.T) {
	mask := image.NewAlpha(image.Rect(0, 0, 5, 5))
	fillAlpha(mask, image.Rect(0, 1, 3, 4), 255)

	got := MorphErode(MorphDilate(mask, 1), 1)
	if countAlpha(got) == 0 {
		t.Fatal("MorphErode(MorphDilate()) removed the subject entirely, want non-zero foreground")
	}
}

func fillAlpha(img *image.Alpha, rect image.Rectangle, a uint8) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetAlpha(x, y, color.Alpha{A: a})
		}
	}
}

func alphaBBox(img *image.Alpha) image.Rectangle {
	return BBox(img, 0)
}

func countAlpha(img *image.Alpha) int {
	bounds := img.Bounds()
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.AlphaAt(x, y).A != 0 {
				count++
			}
		}
	}
	return count
}
