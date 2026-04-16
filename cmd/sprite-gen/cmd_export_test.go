package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/manifest"
)

func TestRunExportGIFJSON(t *testing.T) {
	frameDir := writeExportFrameSet(t, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"export", frameDir, "--format", "gif", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["format"] != "gif" {
		t.Fatalf("data.format = %v, want gif", data["format"])
	}
	if int(data["frames"].(float64)) != 4 {
		t.Fatalf("data.frames = %v, want 4", data["frames"])
	}
}

func TestRunExportSheetPNGJSON(t *testing.T) {
	frameDir := writeExportFrameSet(t, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"export", frameDir, "--format", "sheet-png", "--cols", "4", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if !strings.HasSuffix(data["out"].(string), ".png") {
		t.Fatalf("data.out = %q, want .png suffix", data["out"])
	}
	if data["format"] != "sheet-png" {
		t.Fatalf("data.format = %v, want sheet-png", data["format"])
	}
}

func TestRunExportUnknownFormat(t *testing.T) {
	frameDir := writeExportFrameSet(t, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"export", frameDir, "--format", "bogus"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if got := stderr.String(); !strings.Contains(got, `unknown export format "bogus"; available formats: gif, sheet-png`) {
		t.Fatalf("stderr = %q, want available formats", got)
	}
}

func TestRunExportListFormatsJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"export", "--list-formats", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	formats := data["formats"].([]any)
	if len(formats) != 2 {
		t.Fatalf("len(data.formats) = %d, want 2", len(formats))
	}
	first := formats[0].(map[string]any)
	second := formats[1].(map[string]any)
	if first["name"] != "gif" || second["name"] != "sheet-png" {
		t.Fatalf("format names = [%v, %v], want [gif, sheet-png]", first["name"], second["name"])
	}
}

func writeExportFrameSet(t *testing.T, withManifest bool) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "frames")
	frames := make([]manifest.Frame, 4)
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}}
	for i, c := range colors {
		img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
		fillNRGBA(img, image.Rect(2, 2, 14, 14), c)
		path := filepath.Join(dir, frameName(i))
		writeCommandPNG(t, path, img)
		frames[i] = manifest.Frame{Index: i, Path: frameName(i), Rect: manifest.Rect{X: i * 16, Y: 0, W: 16, H: 16}, W: 16, H: 16}
	}
	if withManifest {
		if err := manifest.Write(filepath.Join(dir, "manifest.json"), &manifest.Manifest{Source: "sheet.png", CellW: 16, CellH: 16, Cols: 4, Rows: 1, Frames: frames}); err != nil {
			t.Fatalf("manifest.Write() error = %v", err)
		}
	}
	return dir
}

func frameName(index int) string {
	return fmt.Sprintf("frame_%03d.png", index)
}
