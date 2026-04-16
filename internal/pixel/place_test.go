package pixel

import (
	"image"
	"image/color"
	"testing"
)

func TestPlaceInCellTopLeftOffset(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	fillPlaceRect(src, image.Rect(1, 1, 3, 3), color.NRGBA{R: 255, A: 255})

	got := PlaceInCell(src, image.Rect(1, 1, 3, 3), image.Pt(4, 4), image.Pt(0, 0))
	if c := got.NRGBAAt(0, 0); c.A != 255 || c.R != 255 {
		t.Fatalf("PlaceInCell() top-left pixel = %#v, want opaque red", c)
	}
	if c := got.NRGBAAt(2, 2); c.A != 0 {
		t.Fatalf("PlaceInCell() pixel at 2,2 = %#v, want transparent", c)
	}
}

func TestPlaceInCellClipsOutsideCell(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	fillPlaceRect(src, src.Bounds(), color.NRGBA{G: 255, A: 255})

	got := PlaceInCell(src, src.Bounds(), image.Pt(2, 2), image.Pt(1, 1))
	if c := got.NRGBAAt(0, 0); c.A != 0 {
		t.Fatalf("PlaceInCell() pixel at 0,0 = %#v, want transparent", c)
	}
	if c := got.NRGBAAt(1, 1); c.A != 255 || c.G != 255 {
		t.Fatalf("PlaceInCell() pixel at 1,1 = %#v, want opaque green", c)
	}
}

func fillPlaceRect(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
