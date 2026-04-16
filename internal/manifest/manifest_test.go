package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	want := &Manifest{
		Source: "sheet.png",
		CellW:  32,
		CellH:  32,
		Cols:   4,
		Rows:   1,
		Frames: []Frame{{
			Index: 0,
			Path:  "frame_000.png",
			Rect:  Rect{X: 1, Y: 2, W: 30, H: 28},
			W:     30,
			H:     28,
			Pivot: &Point{X: 15, Y: 27},
		}},
	}

	if err := Write(path, want); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d", got.Version, CurrentVersion)
	}
	if got.Source != want.Source || got.CellW != want.CellW || got.CellH != want.CellH || got.Cols != want.Cols || got.Rows != want.Rows {
		t.Fatalf("Read() basic fields = %+v, want %+v", got, want)
	}
	if len(got.Frames) != 1 {
		t.Fatalf("len(Frames) = %d, want 1", len(got.Frames))
	}
	frame := got.Frames[0]
	if frame.Index != want.Frames[0].Index || frame.Path != want.Frames[0].Path || frame.Rect != want.Frames[0].Rect || frame.W != want.Frames[0].W || frame.H != want.Frames[0].H {
		t.Fatalf("Frame = %+v, want %+v", frame, want.Frames[0])
	}
	if frame.Pivot == nil || want.Frames[0].Pivot == nil || *frame.Pivot != *want.Frames[0].Pivot {
		t.Fatalf("Frame.Pivot = %+v, want %+v", frame.Pivot, want.Frames[0].Pivot)
	}
}

func TestReadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	_, err := Read(path)
	if err == nil {
		t.Fatal("Read() error = nil, want error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error = %q, want path %q", err.Error(), path)
	}
}

func TestReadMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := Read(path)
	if err == nil {
		t.Fatal("Read() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "parse manifest") || !strings.Contains(err.Error(), path) {
		t.Fatalf("error = %q, want actionable parse error naming path", err.Error())
	}
}

func TestWriteDefaultsVersionToOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := Write(path, &Manifest{}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d", got.Version, CurrentVersion)
	}
}
