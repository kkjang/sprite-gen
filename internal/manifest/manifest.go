package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const CurrentVersion = 1

type Frame struct {
	Index      int
	Path       string
	Rect       Rect
	Row        *int
	Col        *int
	Pivot      *Point
	DurationMS *int
	Tag        string
}

type frameJSON struct {
	Index      int    `json:"index"`
	Path       string `json:"path"`
	X          *int   `json:"x,omitempty"`
	Y          *int   `json:"y,omitempty"`
	W          *int   `json:"w,omitempty"`
	H          *int   `json:"h,omitempty"`
	Rect       *Rect  `json:"rect,omitempty"`
	Row        *int   `json:"row,omitempty"`
	Col        *int   `json:"col,omitempty"`
	Pivot      *Point `json:"pivot,omitempty"`
	DurationMS *int   `json:"duration_ms,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Size struct {
	W int `json:"w"`
	H int `json:"h"`
}

type Manifest struct {
	Version   int     `json:"version"`
	Source    string  `json:"source"`
	CellW     int     `json:"cell_w"`
	CellH     int     `json:"cell_h"`
	Cols      int     `json:"cols"`
	Rows      int     `json:"rows"`
	Sheet     string  `json:"sheet,omitempty"`
	SheetSize *Size   `json:"sheet_size,omitempty"`
	Frames    []Frame `json:"frames"`
}

func (f Frame) MarshalJSON() ([]byte, error) {
	x := f.Rect.X
	y := f.Rect.Y
	w := f.Rect.W
	h := f.Rect.H
	return json.Marshal(frameJSON{
		Index:      f.Index,
		Path:       f.Path,
		X:          &x,
		Y:          &y,
		W:          &w,
		H:          &h,
		Row:        f.Row,
		Col:        f.Col,
		Pivot:      f.Pivot,
		DurationMS: f.DurationMS,
		Tag:        f.Tag,
	})
}

func (f *Frame) UnmarshalJSON(data []byte) error {
	var raw frameJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	flatPresent := raw.X != nil && raw.Y != nil && raw.W != nil && raw.H != nil
	flatPartial := raw.X != nil || raw.Y != nil || raw.W != nil || raw.H != nil

	if !flatPresent && flatPartial && raw.Rect == nil {
		return fmt.Errorf("frame is missing one or more of x/y/w/h")
	}

	frame := Frame{
		Index:      raw.Index,
		Path:       raw.Path,
		Row:        raw.Row,
		Col:        raw.Col,
		Pivot:      raw.Pivot,
		DurationMS: raw.DurationMS,
		Tag:        raw.Tag,
	}
	switch {
	case flatPresent:
		frame.Rect = Rect{X: *raw.X, Y: *raw.Y, W: *raw.W, H: *raw.H}
	case raw.Rect != nil:
		frame.Rect = *raw.Rect
	default:
		return fmt.Errorf("frame is missing x/y/w/h and rect")
	}

	*f = frame
	return nil
}

func Read(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %q: %w", path, err)
	}
	if m.Version == 0 {
		m.Version = CurrentVersion
	}
	return &m, nil
}

func Write(path string, m *Manifest) error {
	if m == nil {
		return fmt.Errorf("write manifest %q: manifest is nil", path)
	}

	copy := *m
	if copy.Version == 0 {
		copy.Version = CurrentVersion
	}

	data, err := json.MarshalIndent(copy, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest %q: %w", path, err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory for %q: %w", path, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp manifest for %q: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp manifest for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp manifest for %q: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("write manifest %q: %w", path, err)
	}
	return nil
}
