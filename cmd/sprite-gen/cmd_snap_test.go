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

func TestRunSnapScaleJSONAuto(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "upscaled_4x.png")
	outPath := filepath.Join(t.TempDir(), "out", "native.png")
	writeCommandPNG(t, inputPath, upscaleTestImage(4))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"snap", "scale", inputPath, "--out", outPath, "--json"}, &stdout, &stderr)
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
	if data["detected_factor"].(float64) != 4 {
		t.Fatalf("detected_factor = %v, want 4", data["detected_factor"])
	}
	if data["out_w"].(float64) != 2 {
		t.Fatalf("out_w = %v, want 2", data["out_w"])
	}
	if data["forced_factor"] != nil {
		t.Fatalf("forced_factor = %v, want nil", data["forced_factor"])
	}
}

func TestRunSnapScaleJSONForced(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "upscaled_4x.png")
	writeCommandPNG(t, inputPath, upscaleTestImage(4))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"snap", "scale", inputPath, "--factor", "4", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["forced_factor"].(float64) != 4 {
		t.Fatalf("forced_factor = %v, want 4", data["forced_factor"])
	}
	wantOut := filepath.Join("out", "upscaled_4x", "snap", "native.png")
	if data["out"] != wantOut {
		t.Fatalf("out = %v, want %q", data["out"], wantOut)
	}
}

func TestRunSnapPixelsJSON(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "aa.png")
	palettePath := filepath.Join(t.TempDir(), "palette.hex")
	outPath := filepath.Join(t.TempDir(), "out", "snapped.png")

	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 240, G: 20, B: 20, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 200, G: 30, B: 30, A: 96})
	writeCommandPNG(t, inputPath, img)
	writePaletteFile(t, palettePath, []color.NRGBA{{R: 255, A: 255}, {B: 255, A: 255}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"snap", "pixels", inputPath, "--palette", palettePath, "--out", outPath, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["palette_size"].(float64) != 2 {
		t.Fatalf("palette_size = %v, want 2", data["palette_size"])
	}
	if data["fractional_pixels_zeroed"].(float64) != 1 {
		t.Fatalf("fractional_pixels_zeroed = %v, want 1", data["fractional_pixels_zeroed"])
	}

	gotImg, err := pixel.LoadPNG(outPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG(%q) error = %v", outPath, err)
	}
	if gotImg.NRGBAAt(1, 0).A != 0 {
		t.Fatalf("alpha = %d, want 0", gotImg.NRGBAAt(1, 0).A)
	}
}

func TestRunSnapPixelsDefaultOutPreservesSubject(t *testing.T) {
	inputPath := filepath.Join("out", "slime3", "snap", "native.png")
	palettePath := filepath.Join(t.TempDir(), "palette.hex")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 200, B: 100, A: 255})
	writeCommandPNG(t, inputPath, img)
	writePaletteFile(t, palettePath, []color.NRGBA{{G: 255, A: 255}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"snap", "pixels", inputPath, "--palette", palettePath, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	wantOut := filepath.Join("out", "slime3", "snap", "snapped.png")
	if data["out"] != wantOut {
		t.Fatalf("out = %v, want %q", data["out"], wantOut)
	}
}

func TestRunSnapPixelsMissingPalette(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.png")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"snap", "pixels", path}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "--palette") {
		t.Fatalf("stderr = %q, want --palette message", stderr.String())
	}
}

func TestRunSnapScaleNonPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-png.txt")
	if err := os.WriteFile(path, []byte("nope"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"snap", "scale", path}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("run() exit code = %d, want non-zero", exitCode)
	}
	if !strings.Contains(stderr.String(), "decode PNG") {
		t.Fatalf("stderr = %q, want actionable decode error", stderr.String())
	}
}

func upscaleTestImage(factor int) image.Image {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	src.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 255})
	src.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, A: 255})
	out := image.NewNRGBA(image.Rect(0, 0, 2*factor, 2*factor))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			c := src.NRGBAAt(x, y)
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					out.SetNRGBA(x*factor+dx, y*factor+dy, c)
				}
			}
		}
	}
	return out
}
