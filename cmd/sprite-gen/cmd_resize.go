package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"path/filepath"
	"strings"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
	internalresize "github.com/kkjang/sprite-gen/internal/resize"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("resize", runResize)
	specreg.Register(specreg.Command{
		Name:        "resize frames",
		Description: "Resize a frame-set directory with integer nearest-neighbor scaling",
		Args:        []specreg.Arg{{Name: "dir", Required: true, Description: "Directory containing frame PNGs and optional manifest.json"}},
		Flags: []specreg.Flag{
			{Name: "up", Description: "Integer nearest-neighbor upscale factor"},
			{Name: "down", Description: "Integer nearest-neighbor downscale factor"},
			{Name: "method", Default: "nearest", Description: "Resize method; only nearest is supported"},
			{Name: "out", Description: "Output directory for resized frame PNGs and manifest"},
			{Name: "dry-run", Default: "false", Description: "Report output paths without writing"},
		},
	})
	specreg.Register(specreg.Command{
		Name:        "resize image",
		Description: "Resize a single PNG with integer nearest-neighbor scaling",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to resize"}},
		Flags: []specreg.Flag{
			{Name: "up", Description: "Integer nearest-neighbor upscale factor"},
			{Name: "down", Description: "Integer nearest-neighbor downscale factor"},
			{Name: "method", Default: "nearest", Description: "Resize method; only nearest is supported"},
			{Name: "out", Description: "Output PNG path"},
			{Name: "dry-run", Default: "false", Description: "Report output path without writing"},
		},
	})
}

func runResize(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing resize subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "image":
		return runResizeImage(args[1:], stdout, asJSON)
	case "frames":
		return runResizeFrames(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown resize subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runResizeImage(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("resize image", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	up := fs.Int("up", 0, "integer nearest-neighbor upscale factor")
	down := fs.Int("down", 0, "integer nearest-neighbor downscale factor")
	method := fs.String("method", "nearest", "resize method; only nearest is supported")
	outPath := fs.String("out", "", "output PNG path")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}

	inPath, err := resolveSinglePathArg(path, fs, "resize image")
	if err != nil {
		return err
	}
	opts, err := parseResizeOptions(*up, *down, *method)
	if err != nil {
		return err
	}
	if *outPath == "" {
		*outPath = defaultResizeImageOutPath(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}
	resized, err := internalresize.Image(img, opts)
	if err != nil {
		return err
	}

	resp := resizeImageResponse{
		Out:       *outPath,
		Direction: string(opts.Direction),
		Factor:    opts.Factor,
		Method:    strings.ToLower(*method),
		InputW:    img.Bounds().Dx(),
		InputH:    img.Bounds().Dy(),
		OutputW:   resized.Bounds().Dx(),
		OutputH:   resized.Bounds().Dy(),
		Unchanged: opts.Factor == 1,
		DryRun:    *dryRun,
	}

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(*outPath, resized); err != nil {
		return err
	}

	text := fmt.Sprintf("%s: %s\ndirection: %s\nfactor: %d\ninput: %dx%d\noutput: %dx%d\nunchanged: %t\n",
		verb,
		*outPath,
		opts.Direction,
		opts.Factor,
		img.Bounds().Dx(),
		img.Bounds().Dy(),
		resized.Bounds().Dx(),
		resized.Bounds().Dy(),
		resp.Unchanged,
	)
	return jsonout.Write(stdout, asJSON, text, resp)
}

func runResizeFrames(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("resize frames", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	up := fs.Int("up", 0, "integer nearest-neighbor upscale factor")
	down := fs.Int("down", 0, "integer nearest-neighbor downscale factor")
	method := fs.String("method", "nearest", "resize method; only nearest is supported")
	outDir := fs.String("out", "", "output directory for resized frame PNGs and manifest")
	dryRun := fs.Bool("dry-run", false, "report output paths without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}

	inDir, err := resolveSinglePathArg(path, fs, "resize frames")
	if err != nil {
		return err
	}
	opts, err := parseResizeOptions(*up, *down, *method)
	if err != nil {
		return err
	}
	if *outDir == "" {
		*outDir = defaultResizeFramesOutDir(inDir)
	}

	set, err := loadFrameSet(inDir)
	if err != nil {
		return err
	}

	imgs := make([]*image.NRGBA, len(set.frames))
	for i, frame := range set.frames {
		imgs[i] = frame.Image
	}
	resized, err := internalresize.Frames(imgs, opts)
	if err != nil {
		return err
	}

	outManifest, warnings := buildResizedManifest(set, resized, opts)
	if !*dryRun {
		for i, frame := range set.frames {
			if err := pixel.SavePNG(filepath.Join(*outDir, frame.Path), resized[i]); err != nil {
				return err
			}
		}
		if err := manifest.Write(filepath.Join(*outDir, "manifest.json"), outManifest); err != nil {
			return err
		}
	}

	resp := resizeFramesResponse{
		Out:       *outDir,
		Direction: string(opts.Direction),
		Factor:    opts.Factor,
		Method:    strings.ToLower(*method),
		Frames:    len(resized),
		CellW:     outManifest.CellW,
		CellH:     outManifest.CellH,
		Unchanged: opts.Factor == 1,
		Warnings:  warnings,
		DryRun:    *dryRun,
	}

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	}
	text := fmt.Sprintf("%s: %s (%d frames, %dx%d each)\ndirection: %s\nfactor: %d\nunchanged: %t\n",
		verb,
		*outDir,
		len(resized),
		outManifest.CellW,
		outManifest.CellH,
		opts.Direction,
		opts.Factor,
		resp.Unchanged,
	)
	for _, warning := range warnings {
		text += fmt.Sprintf("warning: %s\n", warning)
	}
	return jsonout.Write(stdout, asJSON, text, resp)
}

type resizeImageResponse struct {
	Out       string `json:"out"`
	Direction string `json:"direction"`
	Factor    int    `json:"factor"`
	Method    string `json:"method"`
	InputW    int    `json:"input_w"`
	InputH    int    `json:"input_h"`
	OutputW   int    `json:"output_w"`
	OutputH   int    `json:"output_h"`
	Unchanged bool   `json:"unchanged"`
	DryRun    bool   `json:"dry_run"`
}

type resizeFramesResponse struct {
	Out       string   `json:"out"`
	Direction string   `json:"direction"`
	Factor    int      `json:"factor"`
	Method    string   `json:"method"`
	Frames    int      `json:"frames"`
	CellW     int      `json:"cell_w"`
	CellH     int      `json:"cell_h"`
	Unchanged bool     `json:"unchanged"`
	Warnings  []string `json:"warnings,omitempty"`
	DryRun    bool     `json:"dry_run"`
}

func parseResizeOptions(up, down int, method string) (internalresize.Options, error) {
	if !strings.EqualFold(method, "nearest") {
		return internalresize.Options{}, fmt.Errorf("invalid --method value %q; want nearest", method)
	}
	if up != 0 && down != 0 {
		return internalresize.Options{}, fmt.Errorf("provide exactly one of --up or --down")
	}
	if up == 0 && down == 0 {
		return internalresize.Options{}, fmt.Errorf("provide exactly one of --up or --down")
	}
	if up < 0 {
		return internalresize.Options{}, fmt.Errorf("invalid --up %d; want an integer greater than or equal to 1", up)
	}
	if down < 0 {
		return internalresize.Options{}, fmt.Errorf("invalid --down %d; want an integer greater than or equal to 1", down)
	}
	if up > 0 {
		if up < 1 {
			return internalresize.Options{}, fmt.Errorf("invalid --up %d; want an integer greater than or equal to 1", up)
		}
		return internalresize.Options{Direction: internalresize.Up, Factor: up}, nil
	}
	if down < 1 {
		return internalresize.Options{}, fmt.Errorf("invalid --down %d; want an integer greater than or equal to 1", down)
	}
	return internalresize.Options{Direction: internalresize.Down, Factor: down}, nil
}

func buildResizedManifest(set *frameSet, resized []*image.NRGBA, opts internalresize.Options) (*manifest.Manifest, []string) {
	out := &manifest.Manifest{}
	if set.manifest != nil {
		*out = *set.manifest
		out.CellW = scaleOutputValue(out.CellW, opts)
		out.CellH = scaleOutputValue(out.CellH, opts)
	}
	if out.CellW == 0 && len(resized) > 0 {
		out.CellW = resized[0].Bounds().Dx()
	}
	if out.CellH == 0 && len(resized) > 0 {
		out.CellH = resized[0].Bounds().Dy()
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

	warnings := []string{}
	out.Frames = make([]manifest.Frame, len(set.frames))
	for i, frame := range set.frames {
		out.Frames[i] = manifest.Frame{
			Index: frame.Index,
			Path:  frame.Path,
			Rect:  frame.Rect,
			W:     resized[i].Bounds().Dx(),
			H:     resized[i].Bounds().Dy(),
		}
		if set.manifest == nil {
			continue
		}
		pivot := set.manifest.Frames[i].Pivot
		if pivot == nil {
			warnings = append(warnings, fmt.Sprintf("frame %q has no pivot; leaving manifest pivot absent", frame.Path))
			continue
		}
		out.Frames[i].Pivot = &manifest.Point{X: scaleOutputValue(pivot.X, opts), Y: scaleOutputValue(pivot.Y, opts)}
	}
	return out, warnings
}

func scaleOutputValue(value int, opts internalresize.Options) int {
	if opts.Direction == internalresize.Up {
		return value * opts.Factor
	}
	return value / opts.Factor
}

func defaultResizeImageOutPath(inPath string) string {
	return defaultStageOutPath(inPath, "resize", "image.png")
}

func defaultResizeFramesOutDir(inPath string) string {
	return filepath.Join("out", outputSubject(inPath), "resize")
}
