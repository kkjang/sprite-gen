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

func TestRunNormalizeDetailJSONFactor(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "lantern.png")
	outPath := filepath.Join(t.TempDir(), "out", "detail.png")
	writeCommandPNG(t, inputPath, normalizeTestImage(12, 12, image.Rect(2, 1, 10, 11), color.NRGBA{R: 255, A: 255}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"normalize", "detail", inputPath, "--factor", "2", "--out", outPath, "--json"}, &stdout, &stderr)
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
	if data["factor"].(float64) != 2 {
		t.Fatalf("factor = %v, want 2", data["factor"])
	}
	if data["input_bbox_h"].(float64) != 10 {
		t.Fatalf("input_bbox_h = %v, want 10", data["input_bbox_h"])
	}
	if data["output_bbox_h"].(float64) != 5 {
		t.Fatalf("output_bbox_h = %v, want 5", data["output_bbox_h"])
	}

	gotImg, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG(%q) error = %v", outPath, err)
	}
	if gotImg.Bounds().Dx() != 6 || gotImg.Bounds().Dy() != 6 {
		t.Fatalf("output bounds = %v, want 6x6", gotImg.Bounds())
	}
}

func TestRunNormalizeDetailJSONTargetHeightPreservesSubject(t *testing.T) {
	inputPath := filepath.Join("out", "slime3", "prep", "clean.png")
	writeCommandPNG(t, inputPath, normalizeTestImage(12, 12, image.Rect(0, 0, 12, 12), color.NRGBA{G: 255, A: 255}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"normalize", "detail", inputPath, "--target-height", "5", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["factor"].(float64) != 2 {
		t.Fatalf("factor = %v, want 2", data["factor"])
	}
	wantOut := filepath.Join("out", "slime3", "normalize", "detail.png")
	if data["out"] != wantOut {
		t.Fatalf("out = %v, want %q", data["out"], wantOut)
	}
}

func TestRunNormalizeDetailMissingTargetAndFactor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sprite.png")
	writeCommandPNG(t, path, normalizeTestImage(4, 4, image.Rect(0, 0, 4, 4), color.NRGBA{A: 255}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"normalize", "detail", path}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "--target-height") || !strings.Contains(stderr.String(), "--factor") {
		t.Fatalf("stderr = %q, want target/factor message", stderr.String())
	}
}

func TestRunNormalizeDetailRejectsBothTargetAndFactor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sprite.png")
	writeCommandPNG(t, path, normalizeTestImage(4, 4, image.Rect(0, 0, 4, 4), color.NRGBA{A: 255}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"normalize", "detail", path, "--target-height", "4", "--factor", "2"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "exactly one") {
		t.Fatalf("stderr = %q, want exact option message", stderr.String())
	}
}

func TestRunNormalizeDetailRejectsNonDivisibleFactor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sprite.png")
	writeCommandPNG(t, path, normalizeTestImage(10, 6, image.Rect(0, 0, 10, 6), color.NRGBA{A: 255}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"normalize", "detail", path, "--factor", "4"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "does not evenly divide image size 10x6") {
		t.Fatalf("stderr = %q, want divisibility message", stderr.String())
	}
}

func normalizeTestImage(w, h int, rect image.Rectangle, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}
