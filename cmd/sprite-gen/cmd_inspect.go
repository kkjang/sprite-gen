package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"strconv"
	"strings"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("inspect", runInspect)
	specreg.Register(specreg.Command{
		Name:        "inspect frame",
		Description: "Inspect a single sprite frame PNG",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to inspect"}},
		Flags:       []specreg.Flag{{Name: "alpha-threshold", Default: "8", Description: "Minimum alpha used for bbox and pivot calculations"}},
	})
	specreg.Register(specreg.Command{
		Name:        "inspect sheet",
		Description: "Inspect a sprite sheet PNG",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to inspect"}},
		Flags:       []specreg.Flag{{Name: "grid", Default: "auto", Description: "Grid mode: auto, none, or COLSxROWS"}},
	})
}

func runInspect(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing inspect subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "sheet":
		return runInspectSheet(args[1:], stdout, asJSON)
	case "frame":
		return runInspectFrame(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown inspect subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runInspectSheet(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("inspect sheet", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	gridMode := fs.String("grid", "auto", "grid mode: auto, none, or COLSxROWS")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	if path == "" && fs.NArg() == 0 {
		return fmt.Errorf("missing path for inspect sheet")
	}
	if path != "" && fs.NArg() != 0 {
		return fmt.Errorf("inspect sheet takes exactly one path")
	}
	if path == "" && fs.NArg() > 1 {
		return fmt.Errorf("inspect sheet takes exactly one path")
	}
	if path == "" {
		path = fs.Arg(0)
	}

	img, err := pixel.LoadPNG(path)
	if err != nil {
		return err
	}

	stats := pixel.ComputeStats(img)
	declared, mode, err := resolveGrid(*gridMode, stats.W, stats.H)
	if err != nil {
		return err
	}
	detected := pixel.GuessGrid(img)

	resp := inspectSheetResponse{
		Path:         path,
		W:            stats.W,
		H:            stats.H,
		UniqueColors: stats.UniqueColors,
		Alpha:        alphaSummaryFromStats(stats),
		AAScore:      stats.AAScore,
	}

	switch mode {
	case gridModeAuto:
		if detected.Confidence > 0 {
			resp.Grid = &detected
		}
	case gridModeExplicit:
		resp.Grid = &declared
		if mismatch := gridMismatch(declared, detected); mismatch != "" {
			resp.GridWarning = mismatch
		}
	case gridModeNone:
	}

	return jsonout.Write(stdout, asJSON, renderInspectSheetText(resp), resp)
}

func runInspectFrame(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("inspect frame", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	alphaThreshold := fs.Int("alpha-threshold", int(pixel.DefaultBBoxAlphaThreshold), "minimum alpha used for bbox and pivot calculations")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	if path == "" && fs.NArg() == 0 {
		return fmt.Errorf("missing path for inspect frame")
	}
	if path != "" && fs.NArg() != 0 {
		return fmt.Errorf("inspect frame takes exactly one path")
	}
	if path == "" && fs.NArg() > 1 {
		return fmt.Errorf("inspect frame takes exactly one path")
	}
	if path == "" {
		path = fs.Arg(0)
	}
	if *alphaThreshold < 1 || *alphaThreshold > 255 {
		return fmt.Errorf("invalid --alpha-threshold %d; want 1-255", *alphaThreshold)
	}

	img, err := pixel.LoadPNG(path)
	if err != nil {
		return err
	}

	stats := pixel.ComputeStats(img)
	bbox := pixel.BBox(img, uint8(*alphaThreshold-1))
	resp := inspectFrameResponse{
		Path:               path,
		W:                  stats.W,
		H:                  stats.H,
		BBox:               rectSummaryFromRect(bbox),
		BBoxAlphaThreshold: *alphaThreshold,
		PivotHint:          pivotHintFromRect(bbox),
		UniqueColors:       stats.UniqueColors,
		Alpha:              alphaSummaryFromStats(stats),
		AAScore:            stats.AAScore,
	}

	return jsonout.Write(stdout, asJSON, renderInspectFrameText(resp), resp)
}

type inspectSheetResponse struct {
	Path         string       `json:"path"`
	W            int          `json:"w"`
	H            int          `json:"h"`
	UniqueColors int          `json:"unique_colors"`
	Alpha        alphaSummary `json:"alpha"`
	AAScore      float64      `json:"aa_score"`
	Grid         *pixel.Grid  `json:"grid,omitempty"`
	GridWarning  string       `json:"grid_warning,omitempty"`
}

type inspectFrameResponse struct {
	Path               string       `json:"path"`
	W                  int          `json:"w"`
	H                  int          `json:"h"`
	BBox               rectSummary  `json:"bbox"`
	BBoxAlphaThreshold int          `json:"bbox_alpha_threshold"`
	PivotHint          pivotHint    `json:"pivot_hint"`
	UniqueColors       int          `json:"unique_colors"`
	Alpha              alphaSummary `json:"alpha"`
	AAScore            float64      `json:"aa_score"`
}

type alphaSummary struct {
	Transparent int `json:"transparent"`
	Opaque      int `json:"opaque"`
	Fractional  int `json:"fractional"`
}

type rectSummary struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type pivotHint struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Anchor string `json:"anchor"`
}

type gridResolveMode int

const (
	gridModeAuto gridResolveMode = iota
	gridModeExplicit
	gridModeNone
)

func resolveGrid(raw string, imgW, imgH int) (pixel.Grid, gridResolveMode, error) {
	switch strings.ToLower(raw) {
	case "auto":
		return pixel.Grid{}, gridModeAuto, nil
	case "none":
		return pixel.Grid{}, gridModeNone, nil
	}

	parts := strings.Split(strings.ToLower(raw), "x")
	if len(parts) != 2 {
		return pixel.Grid{}, 0, fmt.Errorf("invalid --grid value %q; want auto, none, or COLSxROWS", raw)
	}
	cols, err := strconv.Atoi(parts[0])
	if err != nil || cols <= 0 {
		return pixel.Grid{}, 0, fmt.Errorf("invalid --grid value %q; want positive COLSxROWS", raw)
	}
	rows, err := strconv.Atoi(parts[1])
	if err != nil || rows <= 0 {
		return pixel.Grid{}, 0, fmt.Errorf("invalid --grid value %q; want positive COLSxROWS", raw)
	}
	if imgW%cols != 0 || imgH%rows != 0 {
		return pixel.Grid{}, 0, fmt.Errorf("grid %q does not evenly divide image size %dx%d", raw, imgW, imgH)
	}

	return pixel.Grid{Cols: cols, Rows: rows, CellW: imgW / cols, CellH: imgH / rows, Confidence: 1}, gridModeExplicit, nil
}

func gridMismatch(declared, detected pixel.Grid) string {
	if detected.Confidence == 0 {
		return "auto-detection found no grid structure to compare against"
	}
	if declared.Cols == detected.Cols && declared.Rows == detected.Rows && declared.CellW == detected.CellW && declared.CellH == detected.CellH {
		return ""
	}
	return fmt.Sprintf(
		"declared grid %dx%d (cell %dx%d) does not match detected %dx%d (cell %dx%d, confidence %.2f)",
		declared.Cols, declared.Rows, declared.CellW, declared.CellH,
		detected.Cols, detected.Rows, detected.CellW, detected.CellH, detected.Confidence,
	)
}

func alphaSummaryFromStats(stats pixel.Stats) alphaSummary {
	return alphaSummary{
		Transparent: stats.TransparentPx,
		Opaque:      stats.OpaquePixels,
		Fractional:  stats.FractionalPx,
	}
}

func rectSummaryFromRect(rect image.Rectangle) rectSummary {
	return rectSummary{X: rect.Min.X, Y: rect.Min.Y, W: rect.Dx(), H: rect.Dy()}
}

func pivotHintFromRect(rect image.Rectangle) pivotHint {
	if rect.Empty() {
		return pivotHint{Anchor: "feet"}
	}
	return pivotHint{X: rect.Min.X + rect.Dx()/2, Y: rect.Max.Y - 1, Anchor: "feet"}
}

func renderInspectSheetText(resp inspectSheetResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "path: %s\n", resp.Path)
	fmt.Fprintf(&b, "size: %dx%d\n", resp.W, resp.H)
	fmt.Fprintf(&b, "colors: %d (capped at %d)\n", resp.UniqueColors, pixel.MaxUniqueColors)
	fmt.Fprintf(&b, "alpha: %d transparent, %d opaque, %d fractional\n", resp.Alpha.Transparent, resp.Alpha.Opaque, resp.Alpha.Fractional)
	fmt.Fprintf(&b, "aa_score: %.2f\n", resp.AAScore)
	if resp.Grid != nil {
		fmt.Fprintf(&b, "grid: %dx%d (cell %dx%d, offset %d,%d, confidence %.2f)\n", resp.Grid.Cols, resp.Grid.Rows, resp.Grid.CellW, resp.Grid.CellH, resp.Grid.OffsetX, resp.Grid.OffsetY, resp.Grid.Confidence)
	}
	if resp.GridWarning != "" {
		fmt.Fprintf(&b, "grid_warning: %s\n", resp.GridWarning)
	}
	return b.String()
}

func renderInspectFrameText(resp inspectFrameResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "path: %s\n", resp.Path)
	fmt.Fprintf(&b, "size: %dx%d\n", resp.W, resp.H)
	fmt.Fprintf(&b, "bbox: x=%d y=%d w=%d h=%d (alpha >= %d)\n", resp.BBox.X, resp.BBox.Y, resp.BBox.W, resp.BBox.H, resp.BBoxAlphaThreshold)
	fmt.Fprintf(&b, "pivot_hint: %s x=%d y=%d\n", resp.PivotHint.Anchor, resp.PivotHint.X, resp.PivotHint.Y)
	fmt.Fprintf(&b, "colors: %d (capped at %d)\n", resp.UniqueColors, pixel.MaxUniqueColors)
	fmt.Fprintf(&b, "alpha: %d transparent, %d opaque, %d fractional\n", resp.Alpha.Transparent, resp.Alpha.Opaque, resp.Alpha.Fractional)
	fmt.Fprintf(&b, "aa_score: %.2f\n", resp.AAScore)
	return b.String()
}
