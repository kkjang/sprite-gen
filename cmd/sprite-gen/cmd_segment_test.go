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

func TestRunSegmentSubjectsJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "synthetic_4_blobs.png")
	img, wantRects := syntheticSegmentCanvas()
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects", path, "--cell", "32x32", "--expected", "4", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["cell_w"].(float64) != 32 || data["cell_h"].(float64) != 32 {
		t.Fatalf("data cell = %#v, want 32x32", data)
	}
	frames := data["frames"].([]any)
	if len(frames) != 4 {
		t.Fatalf("len(data.frames) = %d, want 4", len(frames))
	}
	for i, frame := range frames {
		rect := frame.(map[string]any)["src_rect"].(map[string]any)
		gotRect := image.Rect(int(rect["x"].(float64)), int(rect["y"].(float64)), int(rect["x"].(float64))+int(rect["w"].(float64)), int(rect["y"].(float64))+int(rect["h"].(float64)))
		if gotRect != wantRects[i] {
			t.Fatalf("frame %d src_rect = %v, want %v", i, gotRect, wantRects[i])
		}
	}
}

func TestRunSegmentSubjectsWritesFramesAndManifest(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "synthetic_4_blobs.png")
	outDir := filepath.Join(root, "frames")
	img, _ := syntheticSegmentCanvas()
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects", path, "--cell", "32x32", "--expected", "4", "--out", outDir}, &stdout, &stderr)
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
	if gotManifest.Frames[0].Rect.X == 0 && gotManifest.Frames[0].Rect.Y == 0 {
		t.Fatal("manifest frame rect = zero, want source-space rectangle from canvas")
	}
}

func TestRunSegmentSubjectsInfersRowMajorGridMetadata(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "synthetic_2x2_blobs.png")
	outDir := filepath.Join(root, "frames")
	img := syntheticSegmentGridCanvas()
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects", path, "--cell", "32x32", "--expected", "4", "--out", outDir}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	gotManifest, err := manifest.Read(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if gotManifest.Cols != 2 || gotManifest.Rows != 2 {
		t.Fatalf("manifest cols/rows = %d/%d, want 2/2", gotManifest.Cols, gotManifest.Rows)
	}
	for i, want := range []struct{ row, col int }{{0, 0}, {0, 1}, {1, 0}, {1, 1}} {
		frame := gotManifest.Frames[i]
		if frame.Row == nil || *frame.Row != want.row {
			t.Fatalf("frame %d row = %+v, want %d", i, frame.Row, want.row)
		}
		if frame.Col == nil || *frame.Col != want.col {
			t.Fatalf("frame %d col = %+v, want %d", i, frame.Col, want.col)
		}
	}
}

func TestRunSegmentSubjectsExpectedCountMismatch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "synthetic_4_blobs.png")
	img, _ := syntheticSegmentCanvas()
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects", path, "--expected", "5"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "expected 5 subjects, found 4") {
		t.Fatalf("stderr = %q, want expected count message", stderr.String())
	}
	if !strings.Contains(stderr.String(), "lower --min-area") {
		t.Fatalf("stderr = %q, want actionable tuning guidance", stderr.String())
	}
}

func TestRunSegmentSubjectsDryRunDoesNotWriteFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "synthetic_4_blobs.png")
	outDir := filepath.Join(root, "frames")
	img, _ := syntheticSegmentCanvas()
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects", path, "--out", outDir, "--dry-run", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", outDir, err)
	}
	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if dryRun, ok := data["dry_run"].(bool); !ok || !dryRun {
		t.Fatalf("data.dry_run = %v, want true", data["dry_run"])
	}
}

func TestRunSegmentSubjectsMissingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "missing path for segment subjects") {
		t.Fatalf("stderr = %q, want missing path message", stderr.String())
	}
}

func TestRunSegmentSubjectsNonPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-png.txt")
	if err := os.WriteFile(path, []byte("nope"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects", path}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "decode PNG") {
		t.Fatalf("stderr = %q, want actionable decode error", stderr.String())
	}
}

func TestRunSegmentSubjectsOversizeCellError(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "synthetic_4_blobs.png")
	img, _ := syntheticSegmentCanvas()
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects", path, "--cell", "16x16", "--fit", "error"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "exceeds cell 16x16") {
		t.Fatalf("stderr = %q, want oversize error", stderr.String())
	}
}

func TestRunSegmentSubjectsOversizeScaleSucceeds(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "synthetic_4_blobs.png")
	outDir := filepath.Join(root, "frames")
	img, _ := syntheticSegmentCanvas()
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"segment", "subjects", path, "--cell", "16x16", "--fit", "scale", "--out", outDir}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	frame0, err := pixel.LoadPNG(filepath.Join(outDir, "frame_000.png"))
	if err != nil {
		t.Fatalf("pixel.LoadPNG(frame_000) error = %v", err)
	}
	if frame0.Bounds().Dx() != 16 || frame0.Bounds().Dy() != 16 {
		t.Fatalf("frame_000 size = %dx%d, want 16x16", frame0.Bounds().Dx(), frame0.Bounds().Dy())
	}
}

func syntheticSegmentCanvas() (*image.NRGBA, []image.Rectangle) {
	img := image.NewNRGBA(image.Rect(0, 0, 512, 256))
	rects := []image.Rectangle{
		image.Rect(64, 200, 88, 228),
		image.Rect(198, 201, 222, 229),
		image.Rect(334, 198, 358, 226),
		image.Rect(470, 202, 494, 230),
	}
	colors := []color.NRGBA{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
		{R: 255, G: 255, A: 255},
	}
	for i, rect := range rects {
		fillNRGBA(img, rect, colors[i])
	}
	return img, rects
}

func syntheticSegmentGridCanvas() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 160, 96))
	rects := []image.Rectangle{
		image.Rect(10, 8, 28, 28),
		image.Rect(74, 10, 92, 30),
		image.Rect(12, 58, 30, 78),
		image.Rect(72, 56, 90, 76),
	}
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}}
	for i, rect := range rects {
		fillNRGBA(img, rect, colors[i])
	}
	return img
}
