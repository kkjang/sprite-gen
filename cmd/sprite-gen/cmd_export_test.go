package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
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
	if !strings.HasSuffix(data["out"].(string), filepath.Join("out", filepath.Base(frameDir), "export")) {
		t.Fatalf("data.out = %q, want default export directory suffix", data["out"])
	}
	if !strings.HasSuffix(data["gif"].(string), ".gif") {
		t.Fatalf("data.gif = %q, want .gif suffix", data["gif"])
	}
	if int(data["frames"].(float64)) != 4 {
		t.Fatalf("data.frames = %v, want 4", data["frames"])
	}
}

func TestRunExportSheetJSON(t *testing.T) {
	frameDir := writeExportFrameSet(t, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"export", frameDir, "--format", "sheet", "--cols", "4", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if !strings.HasSuffix(data["out"].(string), filepath.Join("out", filepath.Base(frameDir), "export")) {
		t.Fatalf("data.out = %q, want default export directory suffix", data["out"])
	}
	if !strings.HasSuffix(data["png"].(string), ".png") {
		t.Fatalf("data.png = %q, want .png suffix", data["png"])
	}
	if !strings.HasSuffix(data["manifest"].(string), ".json") {
		t.Fatalf("data.manifest = %q, want .json suffix", data["manifest"])
	}
	if data["format"] != "sheet" {
		t.Fatalf("data.format = %v, want sheet", data["format"])
	}
	raw, err := os.ReadFile(data["manifest"].(string))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(manifest) error = %v", err)
	}
	first := decoded["frames"].([]any)[0].(map[string]any)
	if first["x"] != float64(0) || first["y"] != float64(0) || first["w"] != float64(16) || first["h"] != float64(16) {
		t.Fatalf("sheet manifest frame = %+v, want flat x/y/w/h", first)
	}
	if _, exists := first["rect"]; exists {
		t.Fatalf("sheet manifest frame = %+v, want rect omitted", first)
	}
}

func TestRunExportSheetFromRowsDirectoryPreservesRows(t *testing.T) {
	rowDir := writeExportRows(t)
	outDir := filepath.Join(t.TempDir(), "export")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"export", rowDir, "--format", "sheet", "--out", outDir, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	gotManifest, err := manifest.Read(filepath.Join(outDir, filepath.Base(rowDir)+"_sheet.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if gotManifest.Rows != 2 || gotManifest.Cols != 2 {
		t.Fatalf("manifest rows/cols = %d/%d, want 2/2", gotManifest.Rows, gotManifest.Cols)
	}
	if len(gotManifest.Frames) != 4 {
		t.Fatalf("len(frames) = %d, want 4", len(gotManifest.Frames))
	}
	if gotManifest.Frames[0].Tag != "attack" || gotManifest.Frames[1].Tag != "attack" || gotManifest.Frames[2].Tag != "idle" || gotManifest.Frames[3].Tag != "idle" {
		t.Fatalf("frame tags = [%q, %q, %q, %q], want [attack attack idle idle]", gotManifest.Frames[0].Tag, gotManifest.Frames[1].Tag, gotManifest.Frames[2].Tag, gotManifest.Frames[3].Tag)
	}
	if gotManifest.Frames[0].Rect != (manifest.Rect{X: 0, Y: 0, W: 16, H: 16}) || gotManifest.Frames[1].Rect != (manifest.Rect{X: 16, Y: 0, W: 16, H: 16}) {
		t.Fatalf("first row rects = %+v / %+v, want row-major top row", gotManifest.Frames[0].Rect, gotManifest.Frames[1].Rect)
	}
	if gotManifest.Frames[2].Rect != (manifest.Rect{X: 0, Y: 16, W: 16, H: 16}) || gotManifest.Frames[3].Rect != (manifest.Rect{X: 16, Y: 16, W: 16, H: 16}) {
		t.Fatalf("second row rects = %+v / %+v, want row-major second row", gotManifest.Frames[2].Rect, gotManifest.Frames[3].Rect)
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
	if got := stderr.String(); !strings.Contains(got, `unknown export format "bogus"; available formats: gif, sheet`) {
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
	if first["name"] != "gif" || second["name"] != "sheet" {
		t.Fatalf("format names = [%v, %v], want [gif, sheet]", first["name"], second["name"])
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
		frames[i] = manifest.Frame{Index: i, Path: frameName(i), Rect: manifest.Rect{X: i * 16, Y: 0, W: 16, H: 16}}
	}
	if withManifest {
		if err := manifest.Write(filepath.Join(dir, "manifest.json"), &manifest.Manifest{Source: "sheet.png", CellW: 16, CellH: 16, Cols: 4, Rows: 1, Frames: frames}); err != nil {
			t.Fatalf("manifest.Write() error = %v", err)
		}
	}
	return dir
}

func writeExportRows(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "rows")
	rows := []string{"attack", "idle"}
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}}
	colorIndex := 0
	for _, row := range rows {
		rowDir := filepath.Join(dir, row)
		for i := 0; i < 2; i++ {
			img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
			fillNRGBA(img, image.Rect(2, 2, 14, 14), colors[colorIndex%len(colors)])
			colorIndex++
			writeCommandPNG(t, filepath.Join(rowDir, frameName(i)), img)
		}
	}
	return dir
}

func frameName(index int) string {
	return fmt.Sprintf("frame_%03d.png", index)
}
