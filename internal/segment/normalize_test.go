package segment

import (
	"image"
	"image/color"
	"testing"
)

func TestAutoCellRoundsUp(t *testing.T) {
	components := []Component{
		{BBox: image.Rect(0, 0, 27, 30)},
		{BBox: image.Rect(0, 0, 26, 29)},
		{BBox: image.Rect(0, 0, 28, 31)},
	}
	if got := AutoCell(components, 8); got != image.Pt(32, 32) {
		t.Fatalf("AutoCell() = %v, want %v", got, image.Pt(32, 32))
	}
	if got := AutoCell(components, 1); got != image.Pt(28, 31) {
		t.Fatalf("AutoCell(round=1) = %v, want %v", got, image.Pt(28, 31))
	}
}

func TestNormalizeToCellFeetAnchor(t *testing.T) {
	src := subjectImage(16, 16, color.NRGBA{R: 255, A: 255})
	got, err := NormalizeToCell(src, src.Bounds(), image.Pt(32, 32), AnchorFeet, FitError)
	if err != nil {
		t.Fatalf("NormalizeToCell() error = %v", err)
	}
	bbox := opaqueBBox(got)
	if bbox.Min != image.Pt(8, 16) || bbox.Max != image.Pt(24, 32) {
		t.Fatalf("opaque bbox = %v, want %v", bbox, image.Rect(8, 16, 24, 32))
	}
}

func TestNormalizeToCellCenterAnchor(t *testing.T) {
	src := subjectImage(16, 16, color.NRGBA{G: 255, A: 255})
	got, err := NormalizeToCell(src, src.Bounds(), image.Pt(32, 32), AnchorCenter, FitError)
	if err != nil {
		t.Fatalf("NormalizeToCell() error = %v", err)
	}
	bbox := opaqueBBox(got)
	if bbox.Min != image.Pt(8, 8) || bbox.Max != image.Pt(24, 24) {
		t.Fatalf("opaque bbox = %v, want %v", bbox, image.Rect(8, 8, 24, 24))
	}
}

func TestNormalizeToCellOversizeError(t *testing.T) {
	src := subjectImage(20, 20, color.NRGBA{B: 255, A: 255})
	if _, err := NormalizeToCell(src, src.Bounds(), image.Pt(16, 16), AnchorFeet, FitError); err == nil {
		t.Fatal("NormalizeToCell() error = nil, want oversize error")
	}
}

func TestNormalizeToCellDownscalesToFit(t *testing.T) {
	src := subjectImage(32, 32, color.NRGBA{R: 255, G: 255, A: 255})
	got, err := NormalizeToCell(src, src.Bounds(), image.Pt(16, 16), AnchorFeet, FitDownscale)
	if err != nil {
		t.Fatalf("NormalizeToCell() error = %v", err)
	}
	if countOpaque(got) != 16*16 {
		t.Fatalf("opaque pixel count = %d, want %d", countOpaque(got), 16*16)
	}
}

func subjectImage(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func opaqueBBox(img *image.NRGBA) image.Rectangle {
	minX, minY := img.Bounds().Max.X, img.Bounds().Max.Y
	maxX, maxY := img.Bounds().Min.X, img.Bounds().Min.Y
	found := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.NRGBAAt(x, y).A == 0 {
				continue
			}
			if !found {
				minX, minY, maxX, maxY = x, y, x+1, y+1
				found = true
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+1 > maxX {
				maxX = x + 1
			}
			if y+1 > maxY {
				maxY = y + 1
			}
		}
	}
	if !found {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX, maxY)
}

func countOpaque(img *image.NRGBA) int {
	count := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.NRGBAAt(x, y).A != 0 {
				count++
			}
		}
	}
	return count
}
