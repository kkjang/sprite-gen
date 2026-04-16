package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const CurrentVersion = 1

type Frame struct {
	Index int    `json:"index"`
	Path  string `json:"path"`
	Rect  Rect   `json:"rect"`
	W     int    `json:"w,omitempty"`
	H     int    `json:"h,omitempty"`
	Pivot *Point `json:"pivot,omitempty"`
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

type Manifest struct {
	Version int     `json:"version"`
	Source  string  `json:"source"`
	CellW   int     `json:"cell_w"`
	CellH   int     `json:"cell_h"`
	Cols    int     `json:"cols"`
	Rows    int     `json:"rows"`
	Frames  []Frame `json:"frames"`
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest %q: %w", path, err)
	}
	return nil
}
