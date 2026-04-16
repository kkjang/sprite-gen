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

func TestRunSliceGridJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "walk_4x1.png")
	img := image.NewNRGBA(image.Rect(0, 0, 128, 32))
	for i := 0; i < 4; i++ {
		fillNRGBA(img, image.Rect(i*32+8, 8, i*32+24, 24), color.NRGBA{R: uint8(20 + i), A: 255})
	}
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"slice", "grid", path, "--cols", "4", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["cell_w"].(float64) != 32 || data["cols"].(float64) != 4 {
		t.Fatalf("data = %#v, want cell_w=32 cols=4", data)
	}
	frames := data["frames"].([]any)
	if len(frames) != 4 {
		t.Fatalf("len(data.frames) = %d, want 4", len(frames))
	}
}

func TestRunSliceGridWritesFramesAndManifest(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "run_2x2.png")
	outDir := filepath.Join(root, "frames")
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	fillNRGBA(img, image.Rect(0, 0, 32, 32), color.NRGBA{R: 255, A: 255})
	fillNRGBA(img, image.Rect(32, 0, 64, 32), color.NRGBA{G: 255, A: 255})
	fillNRGBA(img, image.Rect(0, 32, 32, 64), color.NRGBA{B: 255, A: 255})
	fillNRGBA(img, image.Rect(32, 32, 64, 64), color.NRGBA{R: 255, G: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"slice", "grid", path, "--cols", "2", "--rows", "2", "--out", outDir}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	frame0, err := pixel.LoadPNG(filepath.Join(outDir, "frame_000.png"))
	if err != nil {
		t.Fatalf("pixel.LoadPNG(frame_000) error = %v", err)
	}
	if frame0.Bounds().Dx() != 32 || frame0.Bounds().Dy() != 32 {
		t.Fatalf("frame_000 size = %dx%d, want 32x32", frame0.Bounds().Dx(), frame0.Bounds().Dy())
	}
	gotManifest, err := manifest.Read(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if len(gotManifest.Frames) != 4 {
		t.Fatalf("manifest frame count = %d, want 4", len(gotManifest.Frames))
	}
}

func TestRunSliceGridInvalidCols(t *testing.T) {
	path := filepath.Join(t.TempDir(), "walk.png")
	img := image.NewNRGBA(image.Rect(0, 0, 128, 32))
	fillNRGBA(img, image.Rect(0, 0, 128, 32), color.NRGBA{R: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"slice", "grid", path, "--cols", "5"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "128 is not divisible by 5") {
		t.Fatalf("stderr = %q, want divisibility message", stderr.String())
	}
}

func TestRunSliceAutoJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gutter_strip.png")
	img := image.NewNRGBA(image.Rect(0, 0, 68, 16))
	fillNRGBA(img, image.Rect(0, 0, 16, 16), color.NRGBA{R: 255, A: 255})
	fillNRGBA(img, image.Rect(17, 0, 33, 16), color.NRGBA{G: 255, A: 255})
	fillNRGBA(img, image.Rect(34, 0, 50, 16), color.NRGBA{B: 255, A: 255})
	fillNRGBA(img, image.Rect(51, 0, 67, 16), color.NRGBA{R: 255, G: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"slice", "auto", path, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	detected := data["detected"].(map[string]any)
	if detected["confidence"].(float64) < 0.8 {
		t.Fatalf("detected.confidence = %v, want >= 0.8", detected["confidence"])
	}
}

func TestRunSliceGridDryRunDoesNotWriteFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "walk.png")
	outDir := filepath.Join(root, "frames")
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillNRGBA(img, image.Rect(0, 0, 32, 32), color.NRGBA{R: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"slice", "grid", path, "--cols", "1", "--out", outDir, "--dry-run"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", outDir, err)
	}
}
