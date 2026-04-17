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

	internaldiff "github.com/kkjang/sprite-gen/internal/diff"
	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

func TestRunAlignFramesJSON(t *testing.T) {
	root := t.TempDir()
	frameDir := filepath.Join(root, "drifting_walk")
	writeDriftingFrameSet(t, frameDir, true)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"align", "frames", frameDir, "--anchor", "feet", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	frames := data["frames"].([]any)
	if len(frames) != 4 {
		t.Fatalf("len(data.frames) = %d, want 4", len(frames))
	}
	target := data["target_pivot"].(map[string]any)
	if int(target["y"].(float64)) <= 0 {
		t.Fatalf("target_pivot.y = %v, want > 0", target["y"])
	}
}

func TestRunAlignFramesWritesManifestAndReducesDrift(t *testing.T) {
	root := t.TempDir()
	frameDir := filepath.Join(root, "drifting_walk")
	outDir := filepath.Join(root, "aligned")
	paths := writeDriftingFrameSet(t, frameDir, true)

	beforeA, err := pixel.LoadPNG(paths[0])
	if err != nil {
		t.Fatalf("pixel.LoadPNG(beforeA) error = %v", err)
	}
	beforeB, err := pixel.LoadPNG(paths[1])
	if err != nil {
		t.Fatalf("pixel.LoadPNG(beforeB) error = %v", err)
	}
	before := internaldiff.Compare(beforeA, beforeB, 0)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"align", "frames", frameDir, "--out", outDir}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	afterA, err := pixel.LoadPNG(filepath.Join(outDir, "frame_000.png"))
	if err != nil {
		t.Fatalf("pixel.LoadPNG(afterA) error = %v", err)
	}
	afterB, err := pixel.LoadPNG(filepath.Join(outDir, "frame_001.png"))
	if err != nil {
		t.Fatalf("pixel.LoadPNG(afterB) error = %v", err)
	}
	after := internaldiff.Compare(afterA, afterB, 0)
	if after.DiffPixels >= before.DiffPixels {
		t.Fatalf("aligned diff = %d, want less than original %d", after.DiffPixels, before.DiffPixels)
	}

	gotManifest, err := manifest.Read(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if len(gotManifest.Frames) != 4 {
		t.Fatalf("manifest frame count = %d, want 4", len(gotManifest.Frames))
	}
	wantPivot := *gotManifest.Frames[0].Pivot
	for i, frame := range gotManifest.Frames {
		if frame.Pivot == nil {
			t.Fatalf("frame %d pivot = nil, want populated pivot", i)
		}
		if *frame.Pivot != wantPivot {
			t.Fatalf("frame %d pivot = %+v, want shared pivot %+v", i, *frame.Pivot, wantPivot)
		}
		if frame.Row == nil || *frame.Row != 0 {
			t.Fatalf("frame %d row = %+v, want 0", i, frame.Row)
		}
		if frame.Col == nil || *frame.Col != i {
			t.Fatalf("frame %d col = %+v, want %d", i, frame.Col, i)
		}
		if frame.Tag != "walk" {
			t.Fatalf("frame %d tag = %q, want walk", i, frame.Tag)
		}
		if frame.DurationMS == nil || *frame.DurationMS != 100+i {
			t.Fatalf("frame %d duration_ms = %+v, want %d", i, frame.DurationMS, 100+i)
		}
	}
}

func TestRunAlignFramesDryRunDoesNotWriteFiles(t *testing.T) {
	root := t.TempDir()
	frameDir := filepath.Join(root, "drifting_walk")
	outDir := filepath.Join(root, "aligned")
	writeDriftingFrameSet(t, frameDir, false)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"align", "frames", frameDir, "--out", outDir, "--dry-run", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", outDir, err)
	}
}

func TestRunAlignFramesMissingDir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"align", "frames", filepath.Join(t.TempDir(), "missing")}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "open frame directory") {
		t.Fatalf("stderr = %q, want directory error", stderr.String())
	}
}

func writeDriftingFrameSet(t *testing.T, dir string, withManifest bool) []string {
	t.Helper()
	shifts := []image.Point{{0, 2}, {-1, 0}, {0, 0}, {1, -1}}
	paths := make([]string, len(shifts))
	frames := make([]manifest.Frame, len(shifts))
	for i, shift := range shifts {
		img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
		fillNRGBA(img, image.Rect(12, 12, 20, 28).Add(shift), color.NRGBA{R: 255, A: 255})
		path := filepath.Join(dir, fmt.Sprintf("frame_%03d.png", i))
		writeCommandPNG(t, path, img)
		paths[i] = path
		rowValue := 0
		colValue := i
		duration := 100 + i
		frames[i] = manifest.Frame{Index: i, Path: filepath.Base(path), Rect: manifest.Rect{X: i * 32, Y: 0, W: 32, H: 32}, Row: &rowValue, Col: &colValue, Tag: "walk", DurationMS: &duration}
	}
	if withManifest {
		if err := manifest.Write(filepath.Join(dir, "manifest.json"), &manifest.Manifest{Source: "sheet.png", CellW: 32, CellH: 32, Cols: 4, Rows: 1, Frames: frames}); err != nil {
			t.Fatalf("manifest.Write() error = %v", err)
		}
	}
	return paths
}
