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
	"github.com/kkjang/sprite-gen/internal/pixel"
)

func TestRunPrepAlphaJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sheet.png")
	img := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 127})
	img.SetNRGBA(2, 0, color.NRGBA{B: 255, A: 1})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "alpha", path, "--alpha-threshold", "128", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["alpha_threshold"].(float64) != 128 {
		t.Fatalf("alpha_threshold = %v, want 128", data["alpha_threshold"])
	}
	if data["fractional_pixels_zeroed"].(float64) != 2 {
		t.Fatalf("fractional_pixels_zeroed = %v, want 2", data["fractional_pixels_zeroed"])
	}
}

func TestRunPrepAlphaWritesCleanedPNG(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sheet.png")
	outPath := filepath.Join(root, "clean.png")
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 127})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "alpha", path, "--alpha-threshold", "128", "--out", outPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	got, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG(%q) error = %v", outPath, err)
	}
	if got.NRGBAAt(0, 0).A != 255 {
		t.Fatalf("alpha at 0,0 = %d, want 255", got.NRGBAAt(0, 0).A)
	}
	if got.NRGBAAt(1, 0).A != 0 {
		t.Fatalf("alpha at 1,0 = %d, want 0", got.NRGBAAt(1, 0).A)
	}
}

func TestRunPrepAlphaDryRunDoesNotWriteFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sheet.png")
	outPath := filepath.Join(root, "clean.png")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 127})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "alpha", path, "--out", outPath, "--dry-run"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", outPath, err)
	}
}

func TestRunPrepAlphaInvalidThreshold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sheet.png")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "alpha", path, "--alpha-threshold", "256"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "--alpha-threshold") {
		t.Fatalf("stderr = %q, want threshold validation message", stderr.String())
	}
}

func TestRunPrepBackgroundKeyJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sheet.png")
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	fillNRGBA(img, img.Bounds(), color.NRGBA{R: 255, B: 255, A: 255})
	fillNRGBA(img, image.Rect(1, 1, 3, 3), color.NRGBA{G: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "background", path, "--method", "key", "--color", "#FF00FF", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["method"] != "key" {
		t.Fatalf("method = %v, want key", data["method"])
	}
	if data["removed_pixels"].(float64) <= 0 {
		t.Fatalf("removed_pixels = %v, want > 0", data["removed_pixels"])
	}
	if data["key_color"] != "#FF00FF" {
		t.Fatalf("key_color = %v, want #FF00FF", data["key_color"])
	}
}

func TestRunPrepBackgroundEdgeWritesCleanedPNG(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sheet.png")
	outPath := filepath.Join(root, "background.png")
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	fillNRGBA(img, img.Bounds(), color.NRGBA{R: 32, G: 32, B: 32, A: 255})
	fillNRGBA(img, image.Rect(2, 2, 6, 6), color.NRGBA{R: 255, G: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "background", path, "--method", "edge", "--out", outPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	got, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG(%q) error = %v", outPath, err)
	}
	if got.NRGBAAt(0, 0).A != 0 {
		t.Fatalf("background alpha = %d, want 0", got.NRGBAAt(0, 0).A)
	}
	if got.NRGBAAt(3, 3).A != 255 {
		t.Fatalf("subject alpha = %d, want 255", got.NRGBAAt(3, 3).A)
	}
}

func TestRunPrepBackgroundDryRunDoesNotWriteFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sheet.png")
	outPath := filepath.Join(root, "background.png")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, B: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "background", path, "--method", "key", "--color", "#FF00FF", "--out", outPath, "--dry-run"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", outPath, err)
	}
}

func TestRunPrepBackgroundMissingColorForKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sheet.png")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "background", path, "--method", "key"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "missing required --color") {
		t.Fatalf("stderr = %q, want missing color message", stderr.String())
	}
}

func TestRunPrepBackgroundConnectivityJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sheet.png")
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	fillNRGBA(img, img.Bounds(), color.NRGBA{R: 255, B: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"prep", "background", path, "--method", "edge", "--connectivity", "8", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["connectivity"].(float64) != 8 {
		t.Fatalf("connectivity = %v, want 8", data["connectivity"])
	}
}
