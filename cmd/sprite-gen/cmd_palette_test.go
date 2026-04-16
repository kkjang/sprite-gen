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
	"github.com/kkjang/sprite-gen/internal/palette"
)

func TestRunPaletteExtractJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "simple.png")
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	img.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 255})
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"palette", "extract", path, "--max", "4", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["count"].(float64) != 4 {
		t.Fatalf("data.count = %v, want 4", data["count"])
	}
}

func TestRunPaletteApplyJSON(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "source.png")
	outPath := filepath.Join(t.TempDir(), "out", "applied.png")
	palettePath := filepath.Join(t.TempDir(), "palette.hex")
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 240, G: 20, B: 20, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 20, G: 20, B: 240, A: 255})
	writeCommandPNG(t, inputPath, img)
	writePaletteFile(t, palettePath, []color.NRGBA{{R: 255, A: 255}, {B: 255, A: 255}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"palette", "apply", inputPath, "--palette", palettePath, "--out", outPath, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", outPath, err)
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["out"] != outPath {
		t.Fatalf("data.out = %v, want %q", data["out"], outPath)
	}
	if data["colors_out"].(float64) != 2 {
		t.Fatalf("data.colors_out = %v, want 2", data["colors_out"])
	}
}

func TestRunPaletteApplyMissingPalette(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.png")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"palette", "apply", path}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "--palette") {
		t.Fatalf("stderr = %q, want --palette message", stderr.String())
	}
}

func writePaletteFile(t *testing.T, path string, pal []color.NRGBA) {
	t.Helper()
	var buf bytes.Buffer
	if err := palette.WriteHex(&buf, pal); err != nil {
		t.Fatalf("palette.WriteHex() error = %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
