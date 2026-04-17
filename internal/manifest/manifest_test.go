package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	want := &Manifest{
		Source:    "sheet.png",
		CellW:     32,
		CellH:     32,
		Cols:      4,
		Rows:      1,
		Sheet:     "hero_sheet.png",
		SheetSize: &Size{W: 128, H: 32},
		Frames: []Frame{{
			Index:      0,
			Path:       "frame_000.png",
			Rect:       Rect{X: 1, Y: 2, W: 30, H: 28},
			Row:        intPtr(1),
			Col:        intPtr(2),
			Pivot:      &Point{X: 15, Y: 27},
			DurationMS: intPtr(100),
			Tag:        "idle",
		}},
	}

	if err := Write(path, want); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	frames := decoded["frames"].([]any)
	frameJSON := frames[0].(map[string]any)
	if frameJSON["x"] != float64(1) || frameJSON["y"] != float64(2) || frameJSON["w"] != float64(30) || frameJSON["h"] != float64(28) {
		t.Fatalf("frame JSON coords = %+v, want flat x/y/w/h", frameJSON)
	}
	if _, exists := frameJSON["rect"]; exists {
		t.Fatalf("frame JSON = %+v, want rect omitted", frameJSON)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d", got.Version, CurrentVersion)
	}
	if got.Source != want.Source || got.CellW != want.CellW || got.CellH != want.CellH || got.Cols != want.Cols || got.Rows != want.Rows || got.Sheet != want.Sheet {
		t.Fatalf("Read() basic fields = %+v, want %+v", got, want)
	}
	if got.SheetSize == nil || want.SheetSize == nil || *got.SheetSize != *want.SheetSize {
		t.Fatalf("SheetSize = %+v, want %+v", got.SheetSize, want.SheetSize)
	}
	if len(got.Frames) != 1 {
		t.Fatalf("len(Frames) = %d, want 1", len(got.Frames))
	}
	frame := got.Frames[0]
	if frame.Index != want.Frames[0].Index || frame.Path != want.Frames[0].Path || frame.Rect != want.Frames[0].Rect || frame.Tag != want.Frames[0].Tag {
		t.Fatalf("Frame = %+v, want %+v", frame, want.Frames[0])
	}
	if frame.Row == nil || want.Frames[0].Row == nil || *frame.Row != *want.Frames[0].Row {
		t.Fatalf("Frame.Row = %+v, want %+v", frame.Row, want.Frames[0].Row)
	}
	if frame.Col == nil || want.Frames[0].Col == nil || *frame.Col != *want.Frames[0].Col {
		t.Fatalf("Frame.Col = %+v, want %+v", frame.Col, want.Frames[0].Col)
	}
	if frame.Pivot == nil || want.Frames[0].Pivot == nil || *frame.Pivot != *want.Frames[0].Pivot {
		t.Fatalf("Frame.Pivot = %+v, want %+v", frame.Pivot, want.Frames[0].Pivot)
	}
	if frame.DurationMS == nil || want.Frames[0].DurationMS == nil || *frame.DurationMS != *want.Frames[0].DurationMS {
		t.Fatalf("Frame.DurationMS = %+v, want %+v", frame.DurationMS, want.Frames[0].DurationMS)
	}
}

func TestReadOldNestedRectSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(`{
  "version": 1,
  "source": "sheet.png",
  "cell_w": 32,
  "cell_h": 32,
  "cols": 1,
  "rows": 1,
  "frames": [
    {
      "index": 0,
      "path": "frame_000.png",
      "rect": {"x": 3, "y": 4, "w": 30, "h": 28}
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got.Frames[0].Rect != (Rect{X: 3, Y: 4, W: 30, H: 28}) {
		t.Fatalf("Frame.Rect = %+v, want {X:3 Y:4 W:30 H:28}", got.Frames[0].Rect)
	}
}

func TestReadNewFlatRectSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(`{
  "version": 1,
  "source": "sheet.png",
  "cell_w": 32,
  "cell_h": 32,
  "cols": 1,
  "rows": 1,
  "frames": [
    {
      "index": 0,
      "path": "frame_000.png",
      "x": 5,
      "y": 6,
      "w": 30,
      "h": 28
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got.Frames[0].Rect != (Rect{X: 5, Y: 6, W: 30, H: 28}) {
		t.Fatalf("Frame.Rect = %+v, want {X:5 Y:6 W:30 H:28}", got.Frames[0].Rect)
	}
}

func TestReadPrefersFlatRectWhenBothShapesExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(`{
  "version": 1,
  "source": "sheet.png",
  "cell_w": 32,
  "cell_h": 32,
  "cols": 1,
  "rows": 1,
  "frames": [
    {
      "index": 0,
      "path": "frame_000.png",
      "x": 7,
      "y": 8,
      "w": 30,
      "h": 28,
      "rect": {"x": 1, "y": 2, "w": 9, "h": 10}
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got.Frames[0].Rect != (Rect{X: 7, Y: 8, W: 30, H: 28}) {
		t.Fatalf("Frame.Rect = %+v, want flat coords to win", got.Frames[0].Rect)
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

func intPtr(value int) *int {
	return &value
}
