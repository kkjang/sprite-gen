package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

func TestRunResizeImageJSONUp(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "sprite.png")
	outPath := filepath.Join(t.TempDir(), "out", "image.png")
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 1, color.NRGBA{G: 255, A: 255})
	writeCommandPNG(t, inputPath, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"resize", "image", inputPath, "--up", "2", "--out", outPath, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["direction"] != "up" || data["factor"].(float64) != 2 {
		t.Fatalf("direction/factor = %v/%v, want up/2", data["direction"], data["factor"])
	}
	if data["output_w"].(float64) != 4 || data["output_h"].(float64) != 4 {
		t.Fatalf("output size = %vx%v, want 4x4", data["output_w"], data["output_h"])
	}

	gotImg, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG(%q) error = %v", outPath, err)
	}
	if gotImg.Bounds().Dx() != 4 || gotImg.Bounds().Dy() != 4 {
		t.Fatalf("output bounds = %v, want 4x4", gotImg.Bounds())
	}
	if gotImg.NRGBAAt(0, 0) != img.NRGBAAt(0, 0) || gotImg.NRGBAAt(1, 1) != img.NRGBAAt(0, 0) {
		t.Fatalf("upscaled top-left block not duplicated correctly")
	}
}

func TestRunResizeImageJSONDown(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "sprite.png")
	outPath := filepath.Join(t.TempDir(), "out", "image.png")
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(10*x + y), G: uint8(20*x + y), A: 255})
		}
	}
	writeCommandPNG(t, inputPath, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"resize", "image", inputPath, "--down", "2", "--out", outPath, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	gotImg, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG(%q) error = %v", outPath, err)
	}
	if gotImg.Bounds().Dx() != 2 || gotImg.Bounds().Dy() != 2 {
		t.Fatalf("output bounds = %v, want 2x2", gotImg.Bounds())
	}
	if gotImg.NRGBAAt(1, 1) != img.NRGBAAt(2, 2) {
		t.Fatalf("downscaled pixel = %#v, want %#v", gotImg.NRGBAAt(1, 1), img.NRGBAAt(2, 2))
	}
}

func TestRunResizeFramesScalesManifestAndWarnsOnMissingPivot(t *testing.T) {
	root := t.TempDir()
	frameDir := filepath.Join(root, "walk")
	outDir := filepath.Join(root, "resized")
	writeResizeFrameSet(t, frameDir, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"resize", "frames", frameDir, "--up", "2", "--out", outDir, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["cell_w"].(float64) != 32 || data["cell_h"].(float64) != 32 {
		t.Fatalf("cell size = %vx%v, want 32x32", data["cell_w"], data["cell_h"])
	}
	warnings := data["warnings"].([]any)
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), "no pivot") {
		t.Fatalf("warnings = %#v, want single missing-pivot warning", warnings)
	}

	gotManifest, err := manifest.Read(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if gotManifest.CellW != 32 || gotManifest.CellH != 32 {
		t.Fatalf("manifest cell size = %dx%d, want 32x32", gotManifest.CellW, gotManifest.CellH)
	}
	if gotManifest.Frames[0].Rect != (manifest.Rect{X: 3, Y: 4, W: 16, H: 16}) {
		t.Fatalf("frame rect = %+v, want preserved rect", gotManifest.Frames[0].Rect)
	}
	if gotManifest.Frames[0].Pivot == nil || *gotManifest.Frames[0].Pivot != (manifest.Point{X: 10, Y: 22}) {
		t.Fatalf("frame 0 pivot = %+v, want scaled pivot 10,22", gotManifest.Frames[0].Pivot)
	}
	if gotManifest.Frames[1].Pivot != nil {
		t.Fatalf("frame 1 pivot = %+v, want nil preserved", gotManifest.Frames[1].Pivot)
	}
	if gotManifest.Frames[0].W != 32 || gotManifest.Frames[0].H != 32 {
		t.Fatalf("frame 0 output size = %dx%d, want 32x32", gotManifest.Frames[0].W, gotManifest.Frames[0].H)
	}

	frameImg, err := pixel.LoadPNG(filepath.Join(outDir, "frame_000.png"))
	if err != nil {
		t.Fatalf("pixel.LoadPNG() error = %v", err)
	}
	if frameImg.Bounds().Dx() != 32 || frameImg.Bounds().Dy() != 32 {
		t.Fatalf("resized frame bounds = %v, want 32x32", frameImg.Bounds())
	}
}

func TestRunResizeFramesWritesManifestWhenInputHasOnlyPNGs(t *testing.T) {
	root := t.TempDir()
	frameDir := filepath.Join(root, "walk")
	writeResizeFrameSet(t, frameDir, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"resize", "frames", frameDir, "--down", "2", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	outDir := filepath.Join("out", "walk", "resize")
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join("out", "walk")) })
	gotManifest, err := manifest.Read(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if gotManifest.Source != frameDir {
		t.Fatalf("manifest source = %q, want %q", gotManifest.Source, frameDir)
	}
	if gotManifest.CellW != 8 || gotManifest.CellH != 8 {
		t.Fatalf("manifest cell size = %dx%d, want 8x8", gotManifest.CellW, gotManifest.CellH)
	}
	if len(gotManifest.Frames) != 2 {
		t.Fatalf("len(frames) = %d, want 2", len(gotManifest.Frames))
	}
	if gotManifest.Frames[0].Pivot != nil {
		t.Fatalf("pivot = %+v, want nil when no manifest was provided", gotManifest.Frames[0].Pivot)
	}
}

func TestRunResizeRejectsBothUpAndDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sprite.png")
	writeCommandPNG(t, path, image.NewNRGBA(image.Rect(0, 0, 4, 4)))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"resize", "image", path, "--up", "2", "--down", "2"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "exactly one of --up or --down") {
		t.Fatalf("stderr = %q, want exact-option validation", stderr.String())
	}
}

func writeResizeFrameSet(t *testing.T, dir string, withManifest bool) {
	t.Helper()
	frames := []manifest.Frame{}
	for i := 0; i < 2; i++ {
		img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
		fillNRGBA(img, image.Rect(4+i, 4, 12+i, 14), color.NRGBA{R: uint8(100 + i), A: 255})
		path := filepath.Join(dir, "frame_000.png")
		if i == 1 {
			path = filepath.Join(dir, "frame_001.png")
		}
		writeCommandPNG(t, path, img)
		frame := manifest.Frame{
			Index: i,
			Path:  filepath.Base(path),
			Rect:  manifest.Rect{X: 3 + i*16, Y: 4, W: 16, H: 16},
			W:     16,
			H:     16,
		}
		if i == 0 {
			frame.Pivot = &manifest.Point{X: 5, Y: 11}
		}
		frames = append(frames, frame)
	}
	if withManifest {
		if err := manifest.Write(filepath.Join(dir, "manifest.json"), &manifest.Manifest{Source: "sheet.png", CellW: 16, CellH: 16, Cols: 2, Rows: 1, Frames: frames}); err != nil {
			t.Fatalf("manifest.Write() error = %v", err)
		}
	}
}
