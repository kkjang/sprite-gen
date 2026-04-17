package export

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"

	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

type Frame struct {
	Index int
	Path  string
	Rect  manifest.Rect
	Tag   string
	Row   *int
	Col   *int
	Image *image.NRGBA
}

type FrameRow struct {
	Tag   string
	Start int
	Count int
}

type Context struct {
	FrameDir     string
	Manifest     *manifest.Manifest
	Frames       []Frame
	FrameRows    []FrameRow
	Options      map[string]string
	OutPath      string
	DryRun       bool
	Format       string
	Subject      string
	ManifestPath string
}

func LoadContext(dir, formatName, subject, outPath string, dryRun bool, options map[string]string) (*Context, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("open frame directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("export requires a directory, got %q", dir)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return loadContextFromManifest(dir, manifestPath, formatName, subject, outPath, dryRun, options)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("open manifest %q: %w", manifestPath, err)
	}
	if formatName == "sheet" {
		ctx, err := loadContextFromRows(dir, formatName, subject, outPath, dryRun, options)
		if err != nil {
			return nil, err
		}
		if ctx != nil {
			return ctx, nil
		}
	}
	return loadContextFromGlob(dir, formatName, subject, outPath, dryRun, options)
}

func loadContextFromManifest(dir, manifestPath, formatName, subject, outPath string, dryRun bool, options map[string]string) (*Context, error) {
	m, err := manifest.Read(manifestPath)
	if err != nil {
		return nil, err
	}
	if len(m.Frames) == 0 {
		return nil, fmt.Errorf("manifest %q has no frames", manifestPath)
	}

	frames := make([]Frame, len(m.Frames))
	for i, frame := range m.Frames {
		img, err := pixel.LoadPNG(filepath.Join(dir, frame.Path))
		if err != nil {
			return nil, err
		}
		frames[i] = Frame{Index: frame.Index, Path: frame.Path, Rect: frame.Rect, Tag: frame.Tag, Row: frame.Row, Col: frame.Col, Image: img}
	}

	return &Context{
		FrameDir:     dir,
		Manifest:     m,
		Frames:       frames,
		Options:      copyOptions(options),
		OutPath:      outPath,
		DryRun:       dryRun,
		Format:       formatName,
		Subject:      subject,
		ManifestPath: manifestPath,
	}, nil
}

func loadContextFromRows(dir, formatName, subject, outPath string, dryRun bool, options map[string]string) (*Context, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list rows in %q: %w", dir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	frames := []Frame{}
	rows := []FrameRow{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		rowDir := filepath.Join(dir, entry.Name())
		paths, err := filepath.Glob(filepath.Join(rowDir, "frame_*.png"))
		if err != nil {
			return nil, fmt.Errorf("list frames in %q: %w", rowDir, err)
		}
		sort.Strings(paths)
		if len(paths) == 0 {
			continue
		}

		start := len(frames)
		for _, path := range paths {
			img, err := pixel.LoadPNG(path)
			if err != nil {
				return nil, err
			}
			bounds := img.Bounds()
			frames = append(frames, Frame{
				Index: len(frames),
				Path:  filepath.ToSlash(filepath.Join(entry.Name(), filepath.Base(path))),
				Rect:  manifest.Rect{X: 0, Y: 0, W: bounds.Dx(), H: bounds.Dy()},
				Tag:   entry.Name(),
				Row:   intPtr(len(rows)),
				Col:   intPtr(len(frames) - start),
				Image: img,
			})
		}
		rows = append(rows, FrameRow{Tag: entry.Name(), Start: start, Count: len(paths)})
	}

	if len(rows) == 0 {
		return nil, nil
	}

	return &Context{
		FrameDir:  dir,
		Frames:    frames,
		FrameRows: rows,
		Options:   copyOptions(options),
		OutPath:   outPath,
		DryRun:    dryRun,
		Format:    formatName,
		Subject:   subject,
	}, nil
}

func intPtr(value int) *int {
	return &value
}

func loadContextFromGlob(dir, formatName, subject, outPath string, dryRun bool, options map[string]string) (*Context, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "frame_*.png"))
	if err != nil {
		return nil, fmt.Errorf("list frames in %q: %w", dir, err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no frame_*.png files found in %q", dir)
	}

	frames := make([]Frame, len(paths))
	for i, path := range paths {
		img, err := pixel.LoadPNG(path)
		if err != nil {
			return nil, err
		}
		bounds := img.Bounds()
		frames[i] = Frame{
			Index: i,
			Path:  filepath.Base(path),
			Rect:  manifest.Rect{X: 0, Y: 0, W: bounds.Dx(), H: bounds.Dy()},
			Image: img,
		}
	}

	return &Context{
		FrameDir: dir,
		Frames:   frames,
		Options:  copyOptions(options),
		OutPath:  outPath,
		DryRun:   dryRun,
		Format:   formatName,
		Subject:  subject,
	}, nil
}

func copyOptions(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
