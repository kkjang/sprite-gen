package pixel

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

func TestDetectScaleTinyImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	if got := DetectScale(img); got != 1 {
		t.Fatalf("DetectScale() = %d, want 1", got)
	}
}

func TestDetectScaleUpscaled4x(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	src.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 255})
	src.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, A: 255})

	img := upscaleNearest(src, 4)
	if got := DetectScale(img); got != 4 {
		t.Fatalf("DetectScale() = %d, want 4", got)
	}
}

func TestDetectScaleRandomLikeImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x*13 + y), G: uint8(x*7 + y*11), B: uint8(x*5 + y*3), A: 255})
		}
	}
	if got := DetectScale(img); got != 1 {
		t.Fatalf("DetectScale() = %d, want 1", got)
	}
}

func TestDownscaleFactorOneIdentity(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(1, 1, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	got := Downscale(img, 1)
	if got.Bounds() != img.Bounds() {
		t.Fatalf("bounds = %v, want %v", got.Bounds(), img.Bounds())
	}
	if got.NRGBAAt(1, 1) != img.NRGBAAt(1, 1) {
		t.Fatalf("pixel = %#v, want %#v", got.NRGBAAt(1, 1), img.NRGBAAt(1, 1))
	}
}

func TestDownscaleFactorFour(t *testing.T) {
	ensureSnapFixtures(t)
	inputPath := filepath.Join(repoRoot(t), "testdata", "input", "snap", "upscaled_4x.png")
	goldenPath := filepath.Join(repoRoot(t), "testdata", "golden", "snap", "upscaled_4x_scaled.png")

	img, err := LoadPNG(inputPath)
	if err != nil {
		t.Fatalf("LoadPNG(%q) error = %v", inputPath, err)
	}
	got := Downscale(img, 4)
	assertPNGEqualToFile(t, got, goldenPath)
}

func TestDownscaleRoundTripDimensions(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 10), G: uint8(y * 10), A: 255})
		}
	}
	path := filepath.Join(t.TempDir(), "scaled.png")
	if err := SavePNG(path, Downscale(img, 2)); err != nil {
		t.Fatalf("SavePNG() error = %v", err)
	}
	got, err := LoadPNG(path)
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 4 {
		t.Fatalf("bounds = %v, want 4x4", got.Bounds())
	}
}

func upscaleNearest(src *image.NRGBA, factor int) *image.NRGBA {
	bounds := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx()*factor, bounds.Dy()*factor))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			c := src.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					out.SetNRGBA(x*factor+dx, y*factor+dy, c)
				}
			}
		}
	}
	return out
}
