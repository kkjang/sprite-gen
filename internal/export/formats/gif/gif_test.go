package gif

import (
	"bytes"
	"image"
	"image/color"
	stdgif "image/gif"
	"os"
	"path/filepath"
	"testing"

	internalexport "github.com/kkjang/sprite-gen/internal/export"
	"github.com/kkjang/sprite-gen/internal/manifest"
)

func TestGIFExportWritesAnimatedGIF(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "preview.gif")
	ctx := testGIFContext(outPath, false, map[string]string{"fps": "8"}, 32, 32)

	result, err := GIF{}.Export(ctx)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if result == nil {
		t.Fatal("Export() result = nil, want summary")
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !bytes.HasPrefix(data, []byte("GIF89a")) && !bytes.HasPrefix(data, []byte("GIF87a")) {
		t.Fatalf("GIF header = %q, want GIF89a or GIF87a", data[:6])
	}

	decoded, err := stdgif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gif.DecodeAll() error = %v", err)
	}
	if len(decoded.Image) != 4 {
		t.Fatalf("len(decoded.Image) = %d, want 4", len(decoded.Image))
	}
	if decoded.Delay[0] != 13 {
		t.Fatalf("decoded.Delay[0] = %d, want 13 centiseconds for 8fps", decoded.Delay[0])
	}
}

func TestGIFExportScale2(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "preview.gif")
	ctx := testGIFContext(outPath, false, map[string]string{"scale": "2"}, 32, 32)

	if _, err := (GIF{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	decoded, err := stdgif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gif.DecodeAll() error = %v", err)
	}
	if got := decoded.Image[0].Bounds().Dx(); got != 64 {
		t.Fatalf("frame width = %d, want 64", got)
	}
	if got := decoded.Image[0].Bounds().Dy(); got != 64 {
		t.Fatalf("frame height = %d, want 64", got)
	}
}

func TestGIFExportDryRunDoesNotWrite(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "preview.gif")
	ctx := testGIFContext(outPath, true, nil, 16, 16)

	if _, err := (GIF{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", outPath, err)
	}
}

func testGIFContext(outPath string, dryRun bool, options map[string]string, w, h int) *internalexport.Context {
	frames := make([]internalexport.Frame, 4)
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}}
	for i, c := range colors {
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		fill(img, image.Rect(4, 4, w-4, h-4), c)
		frames[i] = internalexport.Frame{
			Index: i,
			Path:  "frame.png",
			Rect:  manifest.Rect{W: w, H: h},
			Image: img,
		}
	}
	return &internalexport.Context{Format: "gif", OutPath: outPath, Frames: frames, Options: options, DryRun: dryRun}
}

func fill(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
