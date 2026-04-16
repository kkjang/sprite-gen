package sheetpng

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	internalexport "github.com/kkjang/sprite-gen/internal/export"
	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

func TestSheetPNGExportWritesPNG(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "sheet.png")
	ctx := testSheetContext(outPath, false, map[string]string{"cols": "4"}, []image.Point{{32, 32}, {32, 32}, {32, 32}, {32, 32}})

	if _, err := (SheetPNG{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	img, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG() error = %v", err)
	}
	if got := img.Bounds().Dx(); got != 128 {
		t.Fatalf("sheet width = %d, want 128", got)
	}
	if got := img.Bounds().Dy(); got != 32 {
		t.Fatalf("sheet height = %d, want 32", got)
	}
}

func TestSheetPNGExportPadsMixedSizes(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "sheet.png")
	ctx := testSheetContext(outPath, false, map[string]string{"cols": "2"}, []image.Point{{16, 16}, {32, 24}})

	result, err := SheetPNG{}.Export(ctx)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	data := result.Data.(map[string]any)
	if got := data["mixed_sizes"]; got != true {
		t.Fatalf("mixed_sizes = %v, want true", got)
	}

	img, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG() error = %v", err)
	}
	if got := img.Bounds().Dx(); got != 64 {
		t.Fatalf("sheet width = %d, want 64", got)
	}
	if got := img.Bounds().Dy(); got != 24 {
		t.Fatalf("sheet height = %d, want 24", got)
	}
}

func TestSheetPNGExportPadding(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "sheet.png")
	ctx := testSheetContext(outPath, false, map[string]string{"cols": "2", "padding": "2"}, []image.Point{{10, 10}, {10, 10}, {10, 10}, {10, 10}})

	if _, err := (SheetPNG{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	img, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG() error = %v", err)
	}
	if got := img.Bounds().Dx(); got != 22 {
		t.Fatalf("sheet width = %d, want 22", got)
	}
	if got := img.Bounds().Dy(); got != 22 {
		t.Fatalf("sheet height = %d, want 22", got)
	}
}

func TestSheetPNGExportDryRunDoesNotWrite(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "sheet.png")
	ctx := testSheetContext(outPath, true, nil, []image.Point{{16, 16}})

	if _, err := (SheetPNG{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", outPath, err)
	}
}

func testSheetContext(outPath string, dryRun bool, options map[string]string, sizes []image.Point) *internalexport.Context {
	frames := make([]internalexport.Frame, len(sizes))
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}}
	for i, size := range sizes {
		img := image.NewNRGBA(image.Rect(0, 0, size.X, size.Y))
		fillSheet(img, image.Rect(0, 0, size.X, size.Y), colors[i%len(colors)])
		frames[i] = internalexport.Frame{
			Index: i,
			Path:  "frame.png",
			Rect:  manifest.Rect{W: size.X, H: size.Y},
			Image: img,
		}
	}
	return &internalexport.Context{Format: "sheet-png", OutPath: outPath, Frames: frames, Options: options, DryRun: dryRun}
}

func fillSheet(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
