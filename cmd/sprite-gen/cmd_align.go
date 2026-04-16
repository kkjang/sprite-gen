package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"sort"

	internalalign "github.com/kkjang/sprite-gen/internal/align"
	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("align", runAlign)
	specreg.Register(specreg.Command{
		Name:        "align frames",
		Description: "Align a frame-set directory to a shared pivot",
		Args:        []specreg.Arg{{Name: "dir", Required: true, Description: "Directory containing frame PNGs and optional manifest.json"}},
		Flags: []specreg.Flag{
			{Name: "anchor", Default: string(internalalign.AnchorFeet), Description: "Pivot anchor: centroid, bbox, or feet"},
			{Name: "out", Description: "Output directory for aligned frame PNGs and manifest"},
			{Name: "dry-run", Default: "false", Description: "Report output paths without writing"},
		},
	})
}

func runAlign(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing align subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "frames":
		return runAlignFrames(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown align subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runAlignFrames(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("align frames", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	anchorFlag := fs.String("anchor", string(internalalign.AnchorFeet), "pivot anchor: centroid, bbox, or feet")
	outDir := fs.String("out", "", "output directory for aligned frame PNGs and manifest")
	dryRun := fs.Bool("dry-run", false, "report output paths without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	inDir, err := resolveSinglePathArg(path, fs, "align frames")
	if err != nil {
		return err
	}
	anchor, err := internalalign.ParseAnchor(*anchorFlag)
	if err != nil {
		return err
	}
	if *outDir == "" {
		*outDir = defaultAlignOutDir(inDir)
	}

	set, err := loadFrameSet(inDir)
	if err != nil {
		return err
	}

	imgs := make([]image.Image, len(set.frames))
	pivots := make([]internalalign.Pivot, len(set.frames))
	respFrames := make([]alignFrameResponse, len(set.frames))
	for i, frame := range set.frames {
		pivot := internalalign.ComputePivot(frame.Image, anchor)
		imgs[i] = frame.Image
		pivots[i] = pivot
		respFrames[i] = alignFrameResponse{
			Index: frame.Index,
			Path:  frame.Path,
			DX:    0,
			DY:    0,
			Pivot: pivotSummary{X: pivot.X, Y: pivot.Y},
		}
	}

	aligned, target, err := internalalign.AlignFrames(imgs, pivots)
	if err != nil {
		return err
	}
	for i := range respFrames {
		respFrames[i].DX = target.X - respFrames[i].Pivot.X
		respFrames[i].DY = target.Y - respFrames[i].Pivot.Y
	}

	outManifest := buildAlignedManifest(set, target, aligned)
	if !*dryRun {
		for i, frame := range set.frames {
			if err := pixel.SavePNG(filepath.Join(*outDir, frame.Path), aligned[i]); err != nil {
				return err
			}
		}
		if err := manifest.Write(filepath.Join(*outDir, "manifest.json"), outManifest); err != nil {
			return err
		}
	}

	resp := alignResponse{
		Out:         *outDir,
		Anchor:      string(anchor),
		TargetPivot: pivotSummary{X: target.X, Y: target.Y},
		Frames:      respFrames,
		DryRun:      *dryRun,
	}
	verb := "wrote"
	if *dryRun {
		verb = "would write"
	}
	text := fmt.Sprintf("%s: %s (%d frames)\nanchor: %s\ntarget_pivot: %d,%d\nframe offsets:", verb, *outDir, len(respFrames), anchor, target.X, target.Y)
	for _, frame := range respFrames {
		text += fmt.Sprintf(" [%d,%d]", frame.DX, frame.DY)
	}
	text += "\n"
	return jsonout.Write(stdout, asJSON, text, resp)
}

type alignResponse struct {
	Out         string               `json:"out"`
	Anchor      string               `json:"anchor"`
	TargetPivot pivotSummary         `json:"target_pivot"`
	Frames      []alignFrameResponse `json:"frames"`
	DryRun      bool                 `json:"dry_run"`
}

type alignFrameResponse struct {
	Index int          `json:"index"`
	Path  string       `json:"path"`
	DX    int          `json:"dx"`
	DY    int          `json:"dy"`
	Pivot pivotSummary `json:"pivot"`
}

type pivotSummary struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type loadedFrame struct {
	Index int
	Path  string
	Rect  manifest.Rect
	Image *image.NRGBA
}

type frameSet struct {
	source   string
	manifest *manifest.Manifest
	frames   []loadedFrame
}

func loadFrameSet(dir string) (*frameSet, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("open frame directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("align frames requires a directory, got %q", dir)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return loadFrameSetFromManifest(dir, manifestPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("open manifest %q: %w", manifestPath, err)
	}
	return loadFrameSetFromGlob(dir)
}

func loadFrameSetFromManifest(dir, manifestPath string) (*frameSet, error) {
	m, err := manifest.Read(manifestPath)
	if err != nil {
		return nil, err
	}
	if len(m.Frames) == 0 {
		return nil, fmt.Errorf("manifest %q has no frames", manifestPath)
	}
	frames := make([]loadedFrame, len(m.Frames))
	for i, frame := range m.Frames {
		img, err := pixel.LoadPNG(filepath.Join(dir, frame.Path))
		if err != nil {
			return nil, err
		}
		frames[i] = loadedFrame{Index: frame.Index, Path: frame.Path, Rect: frame.Rect, Image: img}
	}
	return &frameSet{source: m.Source, manifest: m, frames: frames}, nil
}

func loadFrameSetFromGlob(dir string) (*frameSet, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "frame_*.png"))
	if err != nil {
		return nil, fmt.Errorf("list frames in %q: %w", dir, err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no frame_*.png files found in %q", dir)
	}
	frames := make([]loadedFrame, len(paths))
	for i, path := range paths {
		img, err := pixel.LoadPNG(path)
		if err != nil {
			return nil, err
		}
		bounds := img.Bounds()
		frames[i] = loadedFrame{
			Index: i,
			Path:  filepath.Base(path),
			Rect:  manifest.Rect{X: 0, Y: 0, W: bounds.Dx(), H: bounds.Dy()},
			Image: img,
		}
	}
	return &frameSet{source: dir, frames: frames}, nil
}

func buildAlignedManifest(set *frameSet, target internalalign.Pivot, aligned []*image.NRGBA) *manifest.Manifest {
	out := &manifest.Manifest{}
	if set.manifest != nil {
		*out = *set.manifest
	}
	if len(aligned) > 0 {
		out.CellW = aligned[0].Bounds().Dx()
		out.CellH = aligned[0].Bounds().Dy()
	}
	if out.Cols == 0 {
		out.Cols = len(set.frames)
	}
	if out.Rows == 0 {
		out.Rows = 1
	}
	if out.Source == "" {
		out.Source = set.source
	}
	out.Frames = make([]manifest.Frame, len(set.frames))
	for i, frame := range set.frames {
		out.Frames[i] = manifest.Frame{
			Index: frame.Index,
			Path:  frame.Path,
			Rect:  frame.Rect,
			W:     aligned[i].Bounds().Dx(),
			H:     aligned[i].Bounds().Dy(),
			Pivot: &manifest.Point{X: target.X, Y: target.Y},
		}
	}
	return out
}

func defaultAlignOutDir(inPath string) string {
	return filepath.Join("out", outputSubject(inPath), "align")
}
