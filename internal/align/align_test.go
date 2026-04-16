package align

import (
	"image"
	"image/color"
	"testing"
)

func TestComputePivotBBox(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(img, image.Rect(8, 2, 24, 30), color.NRGBA{R: 255, A: 255})

	got := ComputePivot(img, AnchorBBox)
	if got != (Pivot{X: 16, Y: 16}) {
		t.Fatalf("ComputePivot(..., bbox) = %+v, want {16 16}", got)
	}
}

func TestComputePivotFeet(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(img, image.Rect(8, 2, 24, 30), color.NRGBA{G: 255, A: 255})

	got := ComputePivot(img, AnchorFeet)
	if got != (Pivot{X: 16, Y: 29}) {
		t.Fatalf("ComputePivot(..., feet) = %+v, want {16 29}", got)
	}
}

func TestComputePivotCentroid(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(img, image.Rect(8, 8, 24, 24), color.NRGBA{B: 255, A: 255})

	got := ComputePivot(img, AnchorCentroid)
	if got != (Pivot{X: 16, Y: 16}) {
		t.Fatalf("ComputePivot(..., centroid) = %+v, want {16 16}", got)
	}
}

func TestAlignFramesSharedPivot(t *testing.T) {
	imgs := []image.Image{
		driftingSprite(image.Pt(0, 2)),
		driftingSprite(image.Pt(0, -2)),
	}
	pivots := []Pivot{
		ComputePivot(imgs[0], AnchorFeet),
		ComputePivot(imgs[1], AnchorFeet),
	}

	aligned, target, err := AlignFrames(imgs, pivots)
	if err != nil {
		t.Fatalf("AlignFrames() error = %v", err)
	}
	if len(aligned) != 2 {
		t.Fatalf("len(aligned) = %d, want 2", len(aligned))
	}
	if aligned[0].Bounds().Dx() != aligned[1].Bounds().Dx() || aligned[0].Bounds().Dy() != aligned[1].Bounds().Dy() {
		t.Fatalf("aligned bounds = %v and %v, want shared canvas size", aligned[0].Bounds(), aligned[1].Bounds())
	}
	for i, img := range aligned {
		got := ComputePivot(img, AnchorFeet)
		if got != target {
			t.Fatalf("aligned pivot %d = %+v, want %+v", i, got, target)
		}
	}
}

func TestAlignFramesIdempotent(t *testing.T) {
	imgA := driftingSprite(image.Pt(0, 0))
	imgB := driftingSprite(image.Pt(2, 0))
	imgs := []image.Image{imgA, imgB}
	pivots := []Pivot{{X: 16, Y: 27}, {X: 16, Y: 27}}

	aligned, target, err := AlignFrames(imgs, pivots)
	if err != nil {
		t.Fatalf("AlignFrames() error = %v", err)
	}
	if target != (Pivot{X: 16, Y: 27}) {
		t.Fatalf("target = %+v, want {16 27}", target)
	}
	if !sameImage(imgA, aligned[0]) {
		t.Fatal("first aligned frame changed, want identical image")
	}
	if !sameImage(imgB, aligned[1]) {
		t.Fatal("second aligned frame changed, want identical image")
	}
}

func driftingSprite(offset image.Point) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(img, image.Rect(12, 12, 20, 28).Add(offset), color.NRGBA{R: 255, A: 255})
	return img
}

func fillRect(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

func sameImage(a, b *image.NRGBA) bool {
	if !a.Bounds().Eq(b.Bounds()) {
		return false
	}
	return string(a.Pix) == string(b.Pix)
}
