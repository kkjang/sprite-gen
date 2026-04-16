package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

func TestRunInspectSheetJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grid.png")
	img := image.NewNRGBA(image.Rect(0, 0, 128, 32))
	for i := 0; i < 4; i++ {
		x0 := i * 32
		fillNRGBA(img, image.Rect(x0, 0, x0+31, 31), color.NRGBA{R: 255, A: 255})
	}
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"inspect", "sheet", path, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	grid := data["grid"].(map[string]any)
	if grid["cols"].(float64) != 4 {
		t.Fatalf("data.grid.cols = %v, want 4", grid["cols"])
	}
}

func TestRunInspectFrameJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillNRGBA(img, image.Rect(8, 8, 24, 24), color.NRGBA{G: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"inspect", "frame", path, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	bbox := data["bbox"].(map[string]any)
	if bbox["w"].(float64) != 16 || bbox["h"].(float64) != 16 {
		t.Fatalf("data.bbox = %#v, want w=16 h=16", bbox)
	}
	if data["bbox_alpha_threshold"].(float64) != float64(pixel.DefaultBBoxAlphaThreshold) {
		t.Fatalf("data.bbox_alpha_threshold = %v, want %d", data["bbox_alpha_threshold"], pixel.DefaultBBoxAlphaThreshold)
	}
}

func TestRunInspectFrameIgnoresStrayLowAlphaByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillNRGBA(img, image.Rect(8, 8, 24, 24), color.NRGBA{G: 255, A: 255})
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 1})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"inspect", "frame", path, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	bbox := data["bbox"].(map[string]any)
	if bbox["x"].(float64) != 8 || bbox["y"].(float64) != 8 || bbox["w"].(float64) != 16 || bbox["h"].(float64) != 16 {
		t.Fatalf("default bbox = %#v, want centered square without low-alpha stray pixel", bbox)
	}
}

func TestRunInspectFrameAlphaThresholdOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	fillNRGBA(img, image.Rect(8, 8, 24, 24), color.NRGBA{G: 255, A: 255})
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 1})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"inspect", "frame", path, "--alpha-threshold", "1", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	bbox := data["bbox"].(map[string]any)
	if bbox["x"].(float64) != 0 || bbox["y"].(float64) != 0 || bbox["w"].(float64) != 24 || bbox["h"].(float64) != 24 {
		t.Fatalf("overridden bbox = %#v, want bbox that includes low-alpha stray pixel", bbox)
	}
}

func TestRunInspectFrameInvalidAlphaThreshold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	img.SetNRGBA(1, 1, color.NRGBA{G: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"inspect", "frame", path, "--alpha-threshold", "256"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "--alpha-threshold") {
		t.Fatalf("stderr = %q, want threshold validation message", stderr.String())
	}
}

func TestRunInspectMissingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"inspect", "sheet"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("run() exit code = %d, want non-zero", exitCode)
	}
	if !strings.Contains(stderr.String(), "missing path") {
		t.Fatalf("stderr = %q, want missing path message", stderr.String())
	}
}

func TestRunInspectNonPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-png.txt")
	if err := os.WriteFile(path, []byte("nope"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"inspect", "frame", path}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("run() exit code = %d, want non-zero", exitCode)
	}
	if !strings.Contains(stderr.String(), "decode PNG") {
		t.Fatalf("stderr = %q, want actionable decode error", stderr.String())
	}
}

func writeCommandPNG(t *testing.T, path string, img image.Image) {
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

func fillNRGBA(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
