package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/segment"
	"github.com/kkjang/sprite-gen/internal/sheet"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("segment", runSegment)
	specreg.Register(specreg.Command{
		Name:        "segment subjects",
		Description: "Segment alpha-separated subjects into normalized frame cells",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG canvas to segment"}},
		Flags: []specreg.Flag{
			{Name: "alpha-threshold", Default: "128", Description: "Pixels with alpha below this value become background before labeling"},
			{Name: "erode", Default: "0", Description: "Binary erosion iterations to remove soft-edge halos"},
			{Name: "dilate", Default: "0", Description: "Binary dilation iterations after erosion"},
			{Name: "min-area", Default: "auto", Description: "Minimum component area to keep"},
			{Name: "expected", Description: "Fail unless this many components survive filtering"},
			{Name: "cell", Default: "auto", Description: "Output cell size as WxH or auto"},
			{Name: "anchor", Default: "feet", Description: "Placement anchor: feet, center, or top"},
			{Name: "fit", Default: "error", Description: "Oversize policy: error, scale, or crop"},
			{Name: "sort", Default: "ltr", Description: "Frame ordering: ltr (row-major sheet order), area, or none"},
			{Name: "out", Description: "Output directory for frame PNGs and manifest"},
			{Name: "dry-run", Default: "false", Description: "Report output paths without writing"},
		},
	})
}

func runSegment(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing segment subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "subjects":
		return runSegmentSubjects(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown segment subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runSegmentSubjects(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("segment subjects", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	alphaThreshold := fs.Int("alpha-threshold", 128, "pixels with alpha below this value become background before labeling")
	erode := fs.Int("erode", 0, "binary erosion iterations to remove soft-edge halos")
	dilate := fs.Int("dilate", 0, "binary dilation iterations after erosion")
	minAreaFlag := fs.String("min-area", "auto", "minimum component area to keep")
	expected := fs.Int("expected", 0, "fail unless this many components survive filtering")
	cellFlag := fs.String("cell", "auto", "output cell size as WxH or auto")
	anchorFlag := fs.String("anchor", string(segment.AnchorFeet), "placement anchor: feet, center, or top")
	fitFlag := fs.String("fit", string(segment.FitError), "oversize policy: error, scale, or crop")
	sortFlag := fs.String("sort", "ltr", "frame ordering: ltr (row-major sheet order), area, or none")
	outDir := fs.String("out", "", "output directory for frame PNGs and manifest")
	dryRun := fs.Bool("dry-run", false, "report output paths without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	inPath, err := resolveSinglePathArg(path, fs, "segment subjects")
	if err != nil {
		return err
	}
	if *alphaThreshold < 0 || *alphaThreshold > 255 {
		return fmt.Errorf("invalid --alpha-threshold %d; want 0-255", *alphaThreshold)
	}
	if *erode < 0 {
		return fmt.Errorf("--erode must be greater than or equal to 0")
	}
	if *dilate < 0 {
		return fmt.Errorf("--dilate must be greater than or equal to 0")
	}
	if *expected < 0 {
		return fmt.Errorf("--expected must be greater than or equal to 0")
	}

	anchor, err := parseSegmentAnchor(*anchorFlag)
	if err != nil {
		return err
	}
	fit, err := parseSegmentFit(*fitFlag)
	if err != nil {
		return err
	}
	sortMode := strings.ToLower(*sortFlag)
	if sortMode != "ltr" && sortMode != "area" && sortMode != "none" {
		return fmt.Errorf("invalid --sort value %q; want ltr, area, or none", *sortFlag)
	}
	if *outDir == "" {
		*outDir = defaultSegmentOutDir(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}

	mask := pixel.AlphaMask(img, uint8(*alphaThreshold))
	if *erode > 0 {
		mask = pixel.MorphErode(mask, *erode)
	}
	if *dilate > 0 {
		mask = pixel.MorphDilate(mask, *dilate)
	}

	_, components := segment.Label(mask)
	detectedCount := len(components)
	minArea, err := parseSegmentMinArea(*minAreaFlag, img.Bounds())
	if err != nil {
		return err
	}
	components = segment.Filter(components, minArea)
	if len(components) == 0 {
		return fmt.Errorf("no subjects survived segmentation; adjust --min-area (now %d), --alpha-threshold (now %d), --erode (now %d), or --dilate (now %d)", minArea, *alphaThreshold, *erode, *dilate)
	}

	if *expected > 0 && len(components) != *expected {
		return fmt.Errorf("%s", expectedCountMessage(*expected, len(components), minArea, *alphaThreshold))
	}

	cell, err := parseSegmentCell(*cellFlag, components)
	if err != nil {
		return err
	}

	switch sortMode {
	case "ltr":
		segment.SortLTR(components)
	case "area":
		segment.SortAreaDesc(components)
	case "none":
		// Keep the label scan order for deterministic but unsorted output.
	}

	rowGroups := []segment.Row{{Components: components}}
	if sortMode == "ltr" {
		rowGroups = segment.GroupRows(components)
	}

	rowCount := 1
	colCount := len(components)
	if sortMode == "ltr" {
		rowCount = len(rowGroups)
		colCount = 0
		for _, row := range rowGroups {
			if len(row.Components) > colCount {
				colCount = len(row.Components)
			}
		}
	}

	frames := make([]sheet.Frame, 0, len(components))
	manifestFrames := make([]manifest.Frame, 0, len(components))
	responseFrames := make([]segmentFrameResponse, 0, len(components))
	frameIndex := 0
	for rowIndex, row := range rowGroups {
		for colIndex, component := range row.Components {
			normalized, err := segment.NormalizeToCell(img, component.BBox, cell, anchor, fit)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("frame_%03d.png", frameIndex)
			frames = append(frames, sheet.Frame{Index: frameIndex, Path: path, Rect: component.BBox, Image: normalized})
			manifestFrame := manifest.Frame{
				Index: frameIndex,
				Path:  path,
				Rect:  manifest.Rect{X: component.BBox.Min.X, Y: component.BBox.Min.Y, W: component.BBox.Dx(), H: component.BBox.Dy()},
			}
			if sortMode == "ltr" {
				rowValue := rowIndex
				colValue := colIndex
				manifestFrame.Row = &rowValue
				manifestFrame.Col = &colValue
			}
			manifestFrames = append(manifestFrames, manifestFrame)
			responseFrames = append(responseFrames, segmentFrameResponse{
				Index: frameIndex,
				Path:  path,
				Rect:  rectSummary{X: component.BBox.Min.X, Y: component.BBox.Min.Y, W: component.BBox.Dx(), H: component.BBox.Dy()},
				Area:  component.Area,
			})
			frameIndex++
		}
	}

	result := &sheet.Result{
		Cols:   colCount,
		Rows:   rowCount,
		CellW:  cell.X,
		CellH:  cell.Y,
		Frames: frames,
		Manifest: &manifest.Manifest{
			Source: inPath,
			CellW:  cell.X,
			CellH:  cell.Y,
			Cols:   colCount,
			Rows:   rowCount,
			Frames: manifestFrames,
		},
	}
	if !*dryRun {
		if err := sheet.Write(*outDir, result); err != nil {
			return err
		}
	}

	resp := segmentResponse{
		Out:                *outDir,
		CellW:              cell.X,
		CellH:              cell.Y,
		Anchor:             string(anchor),
		Fit:                string(fit),
		Threshold:          *alphaThreshold,
		Erode:              *erode,
		Dilate:             *dilate,
		MinArea:            minArea,
		ComponentsDetected: detectedCount,
		ComponentsKept:     len(components),
		Frames:             responseFrames,
		DryRun:             *dryRun,
	}

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	}
	text := fmt.Sprintf("%s: %s (%d frames, %dx%d each)\ndetected: %d components (%d kept; min_area=%d, threshold=%d, erode=%d, dilate=%d)\nanchor: %s\n",
		verb,
		*outDir,
		len(frames),
		cell.X,
		cell.Y,
		detectedCount,
		len(components),
		minArea,
		*alphaThreshold,
		*erode,
		*dilate,
		anchor,
	)
	return jsonout.Write(stdout, asJSON, text, resp)
}

type segmentResponse struct {
	Out                string                 `json:"out"`
	CellW              int                    `json:"cell_w"`
	CellH              int                    `json:"cell_h"`
	Anchor             string                 `json:"anchor"`
	Fit                string                 `json:"fit"`
	Threshold          int                    `json:"threshold"`
	Erode              int                    `json:"erode"`
	Dilate             int                    `json:"dilate"`
	MinArea            int                    `json:"min_area"`
	ComponentsDetected int                    `json:"components_detected"`
	ComponentsKept     int                    `json:"components_kept"`
	Frames             []segmentFrameResponse `json:"frames"`
	DryRun             bool                   `json:"dry_run"`
}

type segmentFrameResponse struct {
	Index int         `json:"index"`
	Path  string      `json:"path"`
	Rect  rectSummary `json:"src_rect"`
	Area  int         `json:"area"`
}

func parseSegmentAnchor(raw string) (segment.Anchor, error) {
	switch strings.ToLower(raw) {
	case string(segment.AnchorFeet):
		return segment.AnchorFeet, nil
	case string(segment.AnchorCenter):
		return segment.AnchorCenter, nil
	case string(segment.AnchorTop):
		return segment.AnchorTop, nil
	default:
		return "", fmt.Errorf("invalid --anchor value %q; want feet, center, or top", raw)
	}
}

func parseSegmentFit(raw string) (segment.Fit, error) {
	switch strings.ToLower(raw) {
	case string(segment.FitError):
		return segment.FitError, nil
	case string(segment.FitDownscale):
		return segment.FitDownscale, nil
	case string(segment.FitCrop):
		return segment.FitCrop, nil
	default:
		return "", fmt.Errorf("invalid --fit value %q; want error, scale, or crop", raw)
	}
}

func parseSegmentCell(raw string, components []segment.Component) (image.Point, error) {
	if strings.EqualFold(raw, "auto") {
		cell := segment.AutoCell(components, 8)
		if cell.X <= 0 || cell.Y <= 0 {
			return image.Point{}, fmt.Errorf("could not infer a non-zero --cell from detected subjects")
		}
		return cell, nil
	}
	parts := strings.Split(strings.ToLower(raw), "x")
	if len(parts) != 2 {
		return image.Point{}, fmt.Errorf("invalid --cell value %q; want auto or WxH", raw)
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil || w <= 0 {
		return image.Point{}, fmt.Errorf("invalid --cell value %q; want positive WxH", raw)
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil || h <= 0 {
		return image.Point{}, fmt.Errorf("invalid --cell value %q; want positive WxH", raw)
	}
	return image.Pt(w, h), nil
}

func defaultMinArea(bounds image.Rectangle) int {
	imageArea := bounds.Dx() * bounds.Dy()
	value := int(math.Ceil(float64(imageArea) * 0.001))
	if value < 64 {
		return 64
	}
	return value
}

func parseSegmentMinArea(raw string, bounds image.Rectangle) (int, error) {
	if strings.EqualFold(raw, "auto") || raw == "" {
		return defaultMinArea(bounds), nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid --min-area value %q; want auto or a non-negative integer", raw)
	}
	return value, nil
}

func expectedCountMessage(expected, found, minArea, threshold int) string {
	if found < expected {
		return fmt.Sprintf("expected %d subjects, found %d; lower --min-area (now %d) or lower --alpha-threshold (now %d)", expected, found, minArea, threshold)
	}
	return fmt.Sprintf("expected %d subjects, found %d; raise --min-area (now %d) or raise --alpha-threshold (now %d)", expected, found, minArea, threshold)
}

func defaultSegmentOutDir(inPath string) string {
	return filepath.Join("out", outputSubject(inPath), "segment")
}
