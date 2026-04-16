package detail

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/kkjang/sprite-gen/internal/pixel"
)

var update = flag.Bool("update", false, "update golden test files")

func ensureNormalizeFixtures(t *testing.T) {
	t.Helper()
	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, "testdata", "input", "normalize", "lantern_walk.png"),
		filepath.Join(root, "testdata", "input", "normalize", "knight_native.png"),
		filepath.Join(root, "testdata", "golden", "normalize", "lantern_walk_h48.png"),
		filepath.Join(root, "testdata", "golden", "normalize", "knight_native_h48.png"),
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
	writeNormalizeFixtures(t, root)
}

func writeNormalizeFixtures(t *testing.T, root string) {
	t.Helper()
	lantern := makeLanternWalkFixture()
	knight := makeKnightNativeFixture()

	lantern48, err := Normalize(lantern, Options{TargetHeight: 48, AlphaThreshold: 8})
	if err != nil {
		t.Fatalf("Normalize(lantern) error = %v", err)
	}
	knight48, err := Normalize(knight, Options{TargetHeight: 48, AlphaThreshold: 8})
	if err != nil {
		t.Fatalf("Normalize(knight) error = %v", err)
	}

	writePNGFile(t, filepath.Join(root, "testdata", "input", "normalize", "lantern_walk.png"), lantern)
	writePNGFile(t, filepath.Join(root, "testdata", "input", "normalize", "knight_native.png"), knight)
	writePNGFile(t, filepath.Join(root, "testdata", "golden", "normalize", "lantern_walk_h48.png"), lantern48.Image)
	writePNGFile(t, filepath.Join(root, "testdata", "golden", "normalize", "knight_native_h48.png"), knight48.Image)
}

func makeLanternWalkFixture() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	colors := []color.NRGBA{
		{R: 0x18, G: 0x12, B: 0x30, A: 0xff},
		{R: 0xe2, G: 0xa8, B: 0x38, A: 0xff},
		{R: 0xff, G: 0xdc, B: 0x7a, A: 0xff},
		{R: 0x6a, G: 0x42, B: 0x18, A: 0xff},
	}
	baseX := []int{8, 40, 72, 104}
	for i, x0 := range baseX {
		frame := image.Rect(x0, 0, x0+24, 128)
		fillRect(img, image.Rect(frame.Min.X+8, 28, frame.Min.X+16, 42), colors[1])
		fillRect(img, image.Rect(frame.Min.X+7, 42, frame.Min.X+17, 67), colors[2])
		fillRect(img, image.Rect(frame.Min.X+5, 67, frame.Min.X+19, 102), colors[1])
		fillRect(img, image.Rect(frame.Min.X+3, 48, frame.Min.X+5, 88), colors[3])
		fillRect(img, image.Rect(frame.Min.X+19, 48, frame.Min.X+21, 88), colors[3])
		fillRect(img, image.Rect(frame.Min.X+8+i%2, 102, frame.Min.X+11+i%2, 124), colors[0])
		fillRect(img, image.Rect(frame.Min.X+13-(i%2), 102, frame.Min.X+16-(i%2), 124), colors[0])
		img.SetNRGBA(frame.Min.X+11, 37, colors[2])
		img.SetNRGBA(frame.Min.X+12, 37, colors[2])
	}
	return img
}

func makeKnightNativeFixture() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	dark := color.NRGBA{R: 0x18, G: 0x22, B: 0x4c, A: 0xff}
	armor := color.NRGBA{R: 0x72, G: 0x9a, B: 0xd2, A: 0xff}
	highlight := color.NRGBA{R: 0xe6, G: 0xf2, B: 0xff, A: 0xff}
	trim := color.NRGBA{R: 0xf0, G: 0xc4, B: 0x58, A: 0xff}
	haze := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 7}

	fillRect(img, image.Rect(38, 12, 58, 28), dark)
	fillRect(img, image.Rect(34, 28, 62, 58), armor)
	fillRect(img, image.Rect(30, 58, 66, 86), dark)
	fillRect(img, image.Rect(40, 18, 56, 24), highlight)
	fillRect(img, image.Rect(36, 32, 60, 36), trim)
	fillRect(img, image.Rect(28, 64, 34, 88), trim)
	fillRect(img, image.Rect(62, 64, 68, 88), trim)
	fillRect(img, image.Rect(38, 86, 46, 96), dark)
	fillRect(img, image.Rect(50, 86, 58, 96), dark)
	img.SetNRGBA(47, 11, haze)
	img.SetNRGBA(48, 11, haze)
	img.SetNRGBA(47, 10, haze)

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
	want, err := pixel.LoadPNG(wantPath)
	if err != nil {
		t.Fatalf("LoadPNG(%q) error = %v", wantPath, err)
	}
	if want.Bounds() != got.Bounds() {
		t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
	}
	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			r1, g1, b1, a1 := want.At(x, y).RGBA()
			r2, g2, b2, a2 := got.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				t.Fatalf("pixel (%d,%d) = (%d,%d,%d,%d), want (%d,%d,%d,%d)", x, y, r2>>8, g2>>8, b2>>8, a2>>8, r1>>8, g1>>8, b1>>8, a1>>8)
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

func fillRect(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
