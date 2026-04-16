package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/sheet"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("slice", runSlice)
	specreg.Register(specreg.Command{
		Name:        "slice auto",
		Description: "Slice a sprite sheet PNG by auto-detected grid gutters",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG sheet to slice"}},
		Flags: []specreg.Flag{
			{Name: "min-gap", Default: "1", Description: "Minimum transparent gutter width to split grid cells"},
			{Name: "out", Description: "Output directory for frame PNGs and manifest"},
			{Name: "dry-run", Default: "false", Description: "Report output paths without writing"},
		},
	})
	specreg.Register(specreg.Command{
		Name:        "slice grid",
		Description: "Slice a sprite sheet PNG by explicit columns and rows",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG sheet to slice"}},
		Flags: []specreg.Flag{
			{Name: "cols", Description: "Number of columns in the sheet"},
			{Name: "rows", Default: "1", Description: "Number of rows in the sheet"},
			{Name: "trim", Default: "false", Description: "Trim transparent borders from each written frame"},
			{Name: "out", Description: "Output directory for frame PNGs and manifest"},
			{Name: "dry-run", Default: "false", Description: "Report output paths without writing"},
		},
	})
}

func runSlice(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing slice subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "grid":
		return runSliceGrid(args[1:], stdout, asJSON)
	case "auto":
		return runSliceAuto(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown slice subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runSliceGrid(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("slice grid", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	cols := fs.Int("cols", 0, "number of columns in the sheet")
	rows := fs.Int("rows", 1, "number of rows in the sheet")
	trim := fs.Bool("trim", false, "trim transparent borders from each written frame")
	outDir := fs.String("out", "", "output directory for frame PNGs and manifest")
	dryRun := fs.Bool("dry-run", false, "report output paths without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	inPath, err := resolveSinglePathArg(path, fs, "slice grid")
	if err != nil {
		return err
	}
	if *cols <= 0 {
		return fmt.Errorf("missing required --cols for slice grid")
	}
	if *rows <= 0 {
		return fmt.Errorf("--rows must be greater than 0")
	}
	if *outDir == "" {
		*outDir = defaultSliceOutDir(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}
	result, err := sheet.SliceGrid(img, inPath, *cols, *rows, *trim)
	if err != nil {
		return err
	}
	if !*dryRun {
		if err := sheet.Write(*outDir, result); err != nil {
			return err
		}
	}

	resp := sliceResponseFromResult(*outDir, result, *dryRun)
	verb := "wrote"
	if *dryRun {
		verb = "would write"
	}
	text := fmt.Sprintf("%s: %s (%d frames, %dx%d each)\n", verb, *outDir, len(result.Frames), result.CellW, result.CellH)
	return jsonout.Write(stdout, asJSON, text, resp)
}

func runSliceAuto(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("slice auto", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	minGap := fs.Int("min-gap", 1, "minimum transparent gutter width to split grid cells")
	outDir := fs.String("out", "", "output directory for frame PNGs and manifest")
	dryRun := fs.Bool("dry-run", false, "report output paths without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	inPath, err := resolveSinglePathArg(path, fs, "slice auto")
	if err != nil {
		return err
	}
	if *minGap <= 0 {
		return fmt.Errorf("--min-gap must be greater than 0")
	}
	if *outDir == "" {
		*outDir = defaultSliceOutDir(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}
	result, err := sheet.SliceAuto(img, inPath, *minGap)
	if err != nil {
		return err
	}
	if !*dryRun {
		if err := sheet.Write(*outDir, result); err != nil {
			return err
		}
	}

	resp := sliceResponseFromResult(*outDir, result, *dryRun)
	verb := "wrote"
	if *dryRun {
		verb = "would write"
	}
	text := fmt.Sprintf("%s: %s (%d frames, %dx%d each)\n", verb, *outDir, len(result.Frames), result.CellW, result.CellH)
	return jsonout.Write(stdout, asJSON, text, resp)
}

type sliceResponse struct {
	Out      string               `json:"out"`
	Frames   []sliceFrameResponse `json:"frames"`
	Cols     int                  `json:"cols"`
	Rows     int                  `json:"rows"`
	CellW    int                  `json:"cell_w"`
	CellH    int                  `json:"cell_h"`
	DryRun   bool                 `json:"dry_run"`
	Detected *pixel.Grid          `json:"detected,omitempty"`
}

type sliceFrameResponse struct {
	Index int         `json:"index"`
	Path  string      `json:"path"`
	Rect  rectSummary `json:"rect"`
}

func sliceResponseFromResult(outDir string, result *sheet.Result, dryRun bool) sliceResponse {
	frames := make([]sliceFrameResponse, len(result.Manifest.Frames))
	for i, frame := range result.Manifest.Frames {
		frames[i] = sliceFrameResponse{
			Index: frame.Index,
			Path:  frame.Path,
			Rect:  rectSummary{X: frame.Rect.X, Y: frame.Rect.Y, W: frame.Rect.W, H: frame.Rect.H},
		}
	}
	return sliceResponse{
		Out:      outDir,
		Frames:   frames,
		Cols:     result.Cols,
		Rows:     result.Rows,
		CellW:    result.CellW,
		CellH:    result.CellH,
		DryRun:   dryRun,
		Detected: result.Detected,
	}
}

func resolveSinglePathArg(path string, fs *flag.FlagSet, commandName string) (string, error) {
	if path == "" && fs.NArg() == 0 {
		return "", fmt.Errorf("missing path for %s", commandName)
	}
	if path != "" && fs.NArg() != 0 {
		return "", fmt.Errorf("%s takes exactly one path", commandName)
	}
	if path == "" && fs.NArg() > 1 {
		return "", fmt.Errorf("%s takes exactly one path", commandName)
	}
	if path != "" {
		return path, nil
	}
	return fs.Arg(0), nil
}

func defaultSliceOutDir(inPath string) string {
	return filepath.Join("out", outputSubject(inPath), "slice")
}
