package resize

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestImageUpscaleFactorTwoDuplicatesPixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	img.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 255})
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, A: 255})

	got, err := Image(img, Options{Direction: Up, Factor: 2})
	if err != nil {
		t.Fatalf("Image() error = %v", err)
	}
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 4 {
		t.Fatalf("bounds = %v, want 4x4", got.Bounds())
	}
	if got.NRGBAAt(0, 0) != img.NRGBAAt(0, 0) || got.NRGBAAt(1, 1) != img.NRGBAAt(0, 0) {
		t.Fatalf("top-left block = %#v %#v, want duplicated %#v", got.NRGBAAt(0, 0), got.NRGBAAt(1, 1), img.NRGBAAt(0, 0))
	}
	if got.NRGBAAt(2, 0) != img.NRGBAAt(1, 0) || got.NRGBAAt(3, 1) != img.NRGBAAt(1, 0) {
		t.Fatalf("top-right block not duplicated correctly")
	}
	if got.NRGBAAt(0, 2) != img.NRGBAAt(0, 1) || got.NRGBAAt(1, 3) != img.NRGBAAt(0, 1) {
		t.Fatalf("bottom-left block not duplicated correctly")
	}
	if got.NRGBAAt(2, 2) != img.NRGBAAt(1, 1) || got.NRGBAAt(3, 3) != img.NRGBAAt(1, 1) {
		t.Fatalf("bottom-right block not duplicated correctly")
	}
}

func TestImageDownscaleFactorTwoUsesNearestNeighbor(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(10*x + y), G: uint8(20*x + y), B: uint8(30*x + y), A: 255})
		}
	}

	got, err := Image(img, Options{Direction: Down, Factor: 2})
	if err != nil {
		t.Fatalf("Image() error = %v", err)
	}
	if got.Bounds().Dx() != 2 || got.Bounds().Dy() != 2 {
		t.Fatalf("bounds = %v, want 2x2", got.Bounds())
	}
	if got.NRGBAAt(0, 0) != img.NRGBAAt(0, 0) {
		t.Fatalf("pixel (0,0) = %#v, want %#v", got.NRGBAAt(0, 0), img.NRGBAAt(0, 0))
	}
	if got.NRGBAAt(1, 0) != img.NRGBAAt(2, 0) {
		t.Fatalf("pixel (1,0) = %#v, want %#v", got.NRGBAAt(1, 0), img.NRGBAAt(2, 0))
	}
	if got.NRGBAAt(0, 1) != img.NRGBAAt(0, 2) {
		t.Fatalf("pixel (0,1) = %#v, want %#v", got.NRGBAAt(0, 1), img.NRGBAAt(0, 2))
	}
	if got.NRGBAAt(1, 1) != img.NRGBAAt(2, 2) {
		t.Fatalf("pixel (1,1) = %#v, want %#v", got.NRGBAAt(1, 1), img.NRGBAAt(2, 2))
	}
}

func TestImageRejectsInvalidOptions(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))

	if _, err := Image(img, Options{Direction: "sideways", Factor: 2}); err == nil || !strings.Contains(err.Error(), "want up or down") {
		t.Fatalf("Image() error = %v, want direction validation", err)
	}
	if _, err := Image(img, Options{Direction: Up, Factor: 0}); err == nil || !strings.Contains(err.Error(), "greater than or equal to 1") {
		t.Fatalf("Image() error = %v, want factor validation", err)
	}
	if _, err := Image(img, Options{Direction: Down, Factor: 3}); err == nil || !strings.Contains(err.Error(), "does not evenly divide image size 4x4") {
		t.Fatalf("Image() error = %v, want divisibility validation", err)
	}
}

func TestFramesRejectsInvalidFrameSizes(t *testing.T) {
	imgs := []*image.NRGBA{
		image.NewNRGBA(image.Rect(0, 0, 4, 4)),
		image.NewNRGBA(image.Rect(0, 0, 5, 4)),
	}

	_, err := Frames(imgs, Options{Direction: Down, Factor: 2})
	if err == nil {
		t.Fatal("Frames() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "frame 1") || !strings.Contains(err.Error(), "5x4") {
		t.Fatalf("Frames() error = %q, want frame-specific divisibility message", err.Error())
	}
}
