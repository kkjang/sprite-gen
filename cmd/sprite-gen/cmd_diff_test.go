package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"github.com/kkjang/sprite-gen/internal/jsonout"
)

func TestRunDiffFramesJSON(t *testing.T) {
	root := t.TempDir()
	aPath := filepath.Join(root, "frame_000.png")
	bPath := filepath.Join(root, "frame_001.png")
	imgA := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	imgB := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	fillNRGBA(imgA, image.Rect(4, 4, 12, 12), color.NRGBA{R: 255, A: 255})
	fillNRGBA(imgB, image.Rect(5, 4, 13, 12), color.NRGBA{R: 255, A: 255})
	writeCommandPNG(t, aPath, imgA)
	writeCommandPNG(t, bPath, imgB)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"diff", "frames", aPath, bPath, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["diff_pixels"].(float64) <= 0 {
		t.Fatalf("diff_pixels = %v, want > 0", data["diff_pixels"])
	}
}

func TestRunDiffFramesIdenticalImages(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "frame.png")
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	fillNRGBA(img, image.Rect(4, 4, 12, 12), color.NRGBA{G: 255, A: 255})
	writeCommandPNG(t, path, img)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"diff", "frames", path, path, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["diff_pixels"].(float64) != 0 || data["percent"].(float64) != 0 {
		t.Fatalf("data = %#v, want zero diff", data)
	}
}

func TestRunDiffFramesMismatchedSizesReportsMismatch(t *testing.T) {
	root := t.TempDir()
	aPath := filepath.Join(root, "small.png")
	bPath := filepath.Join(root, "large.png")
	writeCommandPNG(t, aPath, image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	writeCommandPNG(t, bPath, image.NewNRGBA(image.Rect(0, 0, 12, 10)))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"diff", "frames", aPath, bPath, "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	mismatch, ok := data["size_mismatch"].(map[string]any)
	if !ok {
		t.Fatalf("data.size_mismatch = %#v, want object", data["size_mismatch"])
	}
	a := mismatch["a"].(map[string]any)
	b := mismatch["b"].(map[string]any)
	if int(a["w"].(float64)) != 8 || int(b["w"].(float64)) != 12 {
		t.Fatalf("size_mismatch = %#v, want 8x8 vs 12x10", mismatch)
	}
}
