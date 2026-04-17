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
	outDir := filepath.Join(t.TempDir(), "export")
	ctx := testGIFContext(outDir, false, map[string]string{"fps": "8"}, "hero", 32, 32)

	result, err := GIF{}.Export(ctx)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if result == nil {
		t.Fatal("Export() result = nil, want summary")
	}
	data := result.Data.(map[string]any)
	gifPath := filepath.Join(outDir, "hero_preview.gif")
	if data["out"] != outDir || data["gif"] != gifPath {
		t.Fatalf("result paths = %+v, want out=%q gif=%q", data, outDir, gifPath)
	}

	dataBytes, err := os.ReadFile(gifPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !bytes.HasPrefix(dataBytes, []byte("GIF89a")) && !bytes.HasPrefix(dataBytes, []byte("GIF87a")) {
		t.Fatalf("GIF header = %q, want GIF89a or GIF87a", dataBytes[:6])
	}

	decoded, err := stdgif.DecodeAll(bytes.NewReader(dataBytes))
	if err != nil {
		t.Fatalf("gif.DecodeAll() error = %v", err)
	}
	if len(decoded.Image) != 4 {
		t.Fatalf("len(decoded.Image) = %d, want 4", len(decoded.Image))
	}
	if decoded.Delay[0] != 13 {
		t.Fatalf("decoded.Delay[0] = %d, want 13 centiseconds for 8fps", decoded.Delay[0])
	}
	if len(decoded.Disposal) != 4 {
		t.Fatalf("len(decoded.Disposal) = %d, want 4", len(decoded.Disposal))
	}
	for i, disposal := range decoded.Disposal {
		if disposal != stdgif.DisposalBackground {
			t.Fatalf("decoded.Disposal[%d] = %d, want %d", i, disposal, stdgif.DisposalBackground)
		}
	}
}

func TestGIFExportScale2(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "export")
	ctx := testGIFContext(outDir, false, map[string]string{"scale": "2"}, "hero", 32, 32)

	if _, err := (GIF{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "hero_preview.gif"))
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
	outDir := filepath.Join(t.TempDir(), "export")
	ctx := testGIFContext(outDir, true, nil, "hero", 16, 16)

	if _, err := (GIF{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	gifPath := filepath.Join(outDir, "hero_preview.gif")
	if _, err := os.Stat(gifPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", gifPath, err)
	}
}

func testGIFContext(outDir string, dryRun bool, options map[string]string, subject string, w, h int) *internalexport.Context {
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
	return &internalexport.Context{Format: "gif", OutPath: outDir, Frames: frames, Options: options, DryRun: dryRun, Subject: subject}
}

func fill(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
