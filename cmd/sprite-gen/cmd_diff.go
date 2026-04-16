package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"path/filepath"

	internaldiff "github.com/kkjang/sprite-gen/internal/diff"
	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("diff", runDiff)
	specreg.Register(specreg.Command{
		Name:        "diff frames",
		Description: "Compare two frame PNGs and write a diff overlay",
		Args: []specreg.Arg{
			{Name: "a", Required: true, Description: "First frame PNG"},
			{Name: "b", Required: true, Description: "Second frame PNG"},
		},
		Flags: []specreg.Flag{
			{Name: "tolerance", Default: "0", Description: "Per-channel difference threshold"},
			{Name: "out", Description: "Output PNG path for the diff overlay"},
			{Name: "dry-run", Default: "false", Description: "Report the output path without writing"},
		},
	})
}

func runDiff(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing diff subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "frames":
		return runDiffFrames(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown diff subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runDiffFrames(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("diff frames", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	tolerance := fs.Int("tolerance", 0, "per-channel difference threshold")
	outPath := fs.String("out", "", "output PNG path for the diff overlay")
	dryRun := fs.Bool("dry-run", false, "report the output path without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("diff frames takes exactly two paths")
	}
	if *tolerance < 0 || *tolerance > 255 {
		return fmt.Errorf("invalid --tolerance %d; want 0-255", *tolerance)
	}
	aPath, bPath := fs.Arg(0), fs.Arg(1)
	if *outPath == "" {
		*outPath = defaultDiffOutPath(aPath, bPath)
	}

	aImg, err := pixel.LoadPNG(aPath)
	if err != nil {
		return err
	}
	bImg, err := pixel.LoadPNG(bPath)
	if err != nil {
		return err
	}

	result := internaldiff.Compare(aImg, bImg, uint8(*tolerance))
	overlay := internaldiff.DiffImage(aImg, bImg, uint8(*tolerance))
	sizeMismatch := diffSizeMismatch(aImg.Bounds(), bImg.Bounds())

	resp := diffResponse{
		DiffPixels:  result.DiffPixels,
		TotalPixels: result.TotalPixels,
		Percent:     result.Percent,
		BBox:        rectSummaryFromRect(result.BBox),
		Tolerance:   *tolerance,
		Out:         *outPath,
		DryRun:      *dryRun,
	}
	if sizeMismatch != nil {
		resp.SizeMismatch = sizeMismatch
	}

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(*outPath, overlay); err != nil {
		return err
	}

	text := fmt.Sprintf("%s: %s\ndiff_pixels: %d\ntotal_pixels: %d\npercent: %.2f\n", verb, *outPath, resp.DiffPixels, resp.TotalPixels, resp.Percent)
	if sizeMismatch != nil {
		text += fmt.Sprintf("size_mismatch: %dx%d vs %dx%d\n", sizeMismatch.A.W, sizeMismatch.A.H, sizeMismatch.B.W, sizeMismatch.B.H)
	}
	return jsonout.Write(stdout, asJSON, text, resp)
}

type diffResponse struct {
	DiffPixels   int               `json:"diff_pixels"`
	TotalPixels  int               `json:"total_pixels"`
	Percent      float64           `json:"percent"`
	BBox         rectSummary       `json:"bbox"`
	Tolerance    int               `json:"tolerance"`
	Out          string            `json:"out"`
	DryRun       bool              `json:"dry_run"`
	SizeMismatch *sizeMismatchInfo `json:"size_mismatch,omitempty"`
}

type sizeMismatchInfo struct {
	A diffImageSize `json:"a"`
	B diffImageSize `json:"b"`
}

type diffImageSize struct {
	W int `json:"w"`
	H int `json:"h"`
}

func diffSizeMismatch(a, b image.Rectangle) *sizeMismatchInfo {
	if a.Dx() == b.Dx() && a.Dy() == b.Dy() {
		return nil
	}
	return &sizeMismatchInfo{
		A: diffImageSize{W: a.Dx(), H: a.Dy()},
		B: diffImageSize{W: b.Dx(), H: b.Dy()},
	}
}

func defaultDiffOutPath(aPath, bPath string) string {
	subject := outputSubject(aPath) + "_vs_" + outputSubject(bPath)
	return filepath.Join("out", subject, "diff", "diff.png")
}
