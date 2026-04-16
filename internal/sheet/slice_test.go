package sheet

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

func TestSliceGridProducesManifestAndFrames(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 128, 32))
	for i := 0; i < 4; i++ {
		fillRect(img, image.Rect(i*32+8, 8, i*32+24, 24), color.NRGBA{R: uint8(10 + i), A: 255})
	}

	got, err := SliceGrid(img, "walk.png", 4, 1, false)
	if err != nil {
		t.Fatalf("SliceGrid() error = %v", err)
	}
	if got.CellW != 32 || got.CellH != 32 {
		t.Fatalf("cell size = %dx%d, want 32x32", got.CellW, got.CellH)
	}
	if len(got.Frames) != 4 || len(got.Manifest.Frames) != 4 {
		t.Fatalf("frame count = %d/%d, want 4", len(got.Frames), len(got.Manifest.Frames))
	}
	if got.Manifest.Version != 0 {
		t.Fatalf("manifest version before write = %d, want 0 so Write applies default", got.Manifest.Version)
	}
	if got.Manifest.Frames[1].Rect != (manifest.Rect{X: 32, Y: 0, W: 32, H: 32}) {
		t.Fatalf("frame rect = %+v, want second cell rect", got.Manifest.Frames[1].Rect)
	}
}

func TestSliceGridTrimUsesTrimmedSourceRect(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(img, image.Rect(10, 12, 20, 22), color.NRGBA{G: 255, A: 255})

	got, err := SliceGrid(img, "frame.png", 1, 1, true)
	if err != nil {
		t.Fatalf("SliceGrid() error = %v", err)
	}
	if got.Manifest.Frames[0].Rect != (manifest.Rect{X: 10, Y: 12, W: 10, H: 10}) {
		t.Fatalf("trimmed rect = %+v, want source-space trimmed rect", got.Manifest.Frames[0].Rect)
	}
	if got.Frames[0].Image.Bounds().Dx() != 10 || got.Frames[0].Image.Bounds().Dy() != 10 {
		t.Fatalf("trimmed image size = %dx%d, want 10x10", got.Frames[0].Image.Bounds().Dx(), got.Frames[0].Image.Bounds().Dy())
	}
}

func TestSliceAutoDetectsGrid(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 68, 16))
	fillRect(img, image.Rect(0, 0, 16, 16), color.NRGBA{R: 255, A: 255})
	fillRect(img, image.Rect(17, 0, 33, 16), color.NRGBA{G: 255, A: 255})
	fillRect(img, image.Rect(34, 0, 50, 16), color.NRGBA{B: 255, A: 255})
	fillRect(img, image.Rect(51, 0, 67, 16), color.NRGBA{R: 255, G: 255, A: 255})

	got, err := SliceAuto(img, "gutter.png", 1)
	if err != nil {
		t.Fatalf("SliceAuto() error = %v", err)
	}
	if got.Detected == nil || got.Detected.Cols != 4 || got.Detected.Rows != 1 {
		t.Fatalf("detected grid = %+v, want 4x1", got.Detected)
	}
	if got.Detected.Confidence < 0.8 {
		t.Fatalf("confidence = %.2f, want >= 0.8", got.Detected.Confidence)
	}
}

func TestWriteWritesFramesAndManifest(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillRect(img, image.Rect(0, 0, 16, 16), color.NRGBA{R: 255, A: 255})
	fillRect(img, image.Rect(16, 0, 32, 16), color.NRGBA{G: 255, A: 255})

	result, err := SliceGrid(img, "sheet.png", 2, 2, false)
	if err != nil {
		t.Fatalf("SliceGrid() error = %v", err)
	}

	outDir := t.TempDir()
	if err := Write(outDir, result); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if _, err := pixel.LoadPNG(filepath.Join(outDir, "frame_000.png")); err != nil {
		t.Fatalf("frame_000.png missing: %v", err)
	}
	gotManifest, err := manifest.Read(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if gotManifest.Version != manifest.CurrentVersion {
		t.Fatalf("manifest version = %d, want %d", gotManifest.Version, manifest.CurrentVersion)
	}
}

func fillRect(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
