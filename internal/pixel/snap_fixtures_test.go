package pixel

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	internalpalette "github.com/kkjang/sprite-gen/internal/palette"
)

func ensureSnapFixtures(t *testing.T) {
	t.Helper()
	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "testdata", "input", "snap", "aa_knight.png"),
		filepath.Join(root, "testdata", "input", "snap", "upscaled_4x.png"),
		filepath.Join(root, "testdata", "input", "snap", "upscaled_2x.png"),
		filepath.Join(root, "testdata", "golden", "snap", "aa_knight_thresh.png"),
		filepath.Join(root, "testdata", "golden", "snap", "aa_knight_snapped.png"),
		filepath.Join(root, "testdata", "golden", "snap", "upscaled_4x_scaled.png"),
		filepath.Join(root, "testdata", "golden", "snap", "upscaled_2x_scaled.png"),
		filepath.Join(root, "testdata", "golden", "palette", "knight_4.hex"),
	}
	missing := false
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			missing = true
			break
		}
	}
	if !*update && !missing {
		return
	}
	writeSnapFixtures(t, root)
}

func writeSnapFixtures(t *testing.T, root string) {
	t.Helper()
	aaInput := makeAAKnightFixture()
	aaThresholded := ThresholdAlpha(aaInput, 0, 128)
	paletteColors := []color.NRGBA{
		{R: 0x12, G: 0x18, B: 0x38, A: 0xff},
		{R: 0x4a, G: 0x7a, B: 0xd8, A: 0xff},
		{R: 0xd8, G: 0xf0, B: 0xff, A: 0xff},
		{R: 0xf2, G: 0xc1, B: 0x4e, A: 0xff},
	}
	aaSnapped := internalpalette.Apply(aaThresholded, paletteColors, false)

	native := makeNativeFixture()
	upscaled4x := upscaleNearest(native, 4)
	upscaled2x := upscaleNearest(native, 2)

	writePNGFile(t, filepath.Join(root, "testdata", "input", "snap", "aa_knight.png"), aaInput)
	writePNGFile(t, filepath.Join(root, "testdata", "input", "snap", "upscaled_4x.png"), upscaled4x)
	writePNGFile(t, filepath.Join(root, "testdata", "input", "snap", "upscaled_2x.png"), upscaled2x)
	writePNGFile(t, filepath.Join(root, "testdata", "golden", "snap", "aa_knight_thresh.png"), aaThresholded)
	writePNGFile(t, filepath.Join(root, "testdata", "golden", "snap", "aa_knight_snapped.png"), aaSnapped)
	writePNGFile(t, filepath.Join(root, "testdata", "golden", "snap", "upscaled_4x_scaled.png"), native)
	writePNGFile(t, filepath.Join(root, "testdata", "golden", "snap", "upscaled_2x_scaled.png"), native)

	var buf bytes.Buffer
	if err := internalpalette.WriteHex(&buf, paletteColors); err != nil {
		t.Fatalf("palette.WriteHex() error = %v", err)
	}
	writeBytesFile(t, filepath.Join(root, "testdata", "golden", "palette", "knight_4.hex"), buf.Bytes())
}

func makeAAKnightFixture() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	outline := color.NRGBA{R: 0x15, G: 0x1a, B: 0x3c, A: 255}
	fill := color.NRGBA{R: 0x48, G: 0x78, B: 0xd0, A: 255}
	highlight := color.NRGBA{R: 0xdc, G: 0xee, B: 0xff, A: 255}
	metalAA := color.NRGBA{R: 0xf2, G: 0xc8, B: 0x62, A: 96}
	softAA := color.NRGBA{R: 0x70, G: 0xa0, B: 0xee, A: 80}

	for y := 2; y <= 5; y++ {
		for x := 2; x <= 5; x++ {
			img.SetNRGBA(x, y, fill)
		}
	}
	for x := 2; x <= 5; x++ {
		img.SetNRGBA(x, 2, outline)
		img.SetNRGBA(x, 5, outline)
	}
	for y := 2; y <= 5; y++ {
		img.SetNRGBA(2, y, outline)
		img.SetNRGBA(5, y, outline)
	}
	img.SetNRGBA(3, 3, highlight)
	img.SetNRGBA(4, 3, highlight)

	img.SetNRGBA(1, 2, softAA)
	img.SetNRGBA(6, 2, softAA)
	img.SetNRGBA(1, 5, softAA)
	img.SetNRGBA(6, 5, softAA)
	img.SetNRGBA(3, 1, metalAA)
	img.SetNRGBA(4, 1, metalAA)
	img.SetNRGBA(3, 6, metalAA)
	img.SetNRGBA(4, 6, metalAA)

	return img
}

func makeNativeFixture() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	colors := []color.NRGBA{
		{R: 0x10, G: 0x20, B: 0x40, A: 0xff},
		{R: 0x44, G: 0x88, B: 0xcc, A: 0xff},
		{R: 0xdd, G: 0xf2, B: 0xff, A: 0xff},
		{R: 0xf0, G: 0xc0, B: 0x48, A: 0xff},
	}
	idx := 0
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetNRGBA(x, y, colors[idx%len(colors)])
			idx++
		}
	}
	return img
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	return root
}

func assertPNGEqualToFile(t *testing.T, got image.Image, wantPath string) {
	t.Helper()
	want, err := LoadPNG(wantPath)
	if err != nil {
		t.Fatalf("LoadPNG(%q) error = %v", wantPath, err)
	}
	if want.Bounds() != got.Bounds() {
		t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
	}
	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			if want.At(x, y) != got.At(x, y) {
				r1, g1, b1, a1 := want.At(x, y).RGBA()
				r2, g2, b2, a2 := got.At(x, y).RGBA()
				if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
					t.Fatalf("pixel (%d,%d) = (%d,%d,%d,%d), want (%d,%d,%d,%d)", x, y, r2>>8, g2>>8, b2>>8, a2>>8, r1>>8, g1>>8, b1>>8, a1>>8)
				}
			}
		}
	}
}

func writePNGFile(t *testing.T, path string, img image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("png.Encode(%q) error = %v", path, err)
	}
}

func writeBytesFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
