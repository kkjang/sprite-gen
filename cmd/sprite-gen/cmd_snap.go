package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"strconv"
	"strings"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/palette"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("snap", runSnap)
	specreg.Register(specreg.Command{
		Name:        "snap pixels",
		Description: "Remove soft alpha edges and snap colors to a palette",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to clean up"}},
		Flags: []specreg.Flag{
			{Name: "palette", Description: "Path to a .hex or .gpl palette file"},
			{Name: "alpha-threshold", Default: "128", Description: "Pixels with alpha below this value become transparent"},
			{Name: "out", Description: "Output PNG path"},
			{Name: "dry-run", Default: "false", Description: "Report the output path without writing"},
		},
	})
	specreg.Register(specreg.Command{
		Name:        "snap scale",
		Description: "Detect and undo integer nearest-neighbor upscaling",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to downscale"}},
		Flags: []specreg.Flag{
			{Name: "factor", Default: "auto", Description: "Scale factor: auto, 1, 2, 3, 4, or 8"},
			{Name: "out", Description: "Output PNG path"},
			{Name: "dry-run", Default: "false", Description: "Report the output path without writing"},
		},
	})
}

func runSnap(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing snap subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "pixels":
		return runSnapPixels(args[1:], stdout, asJSON)
	case "scale":
		return runSnapScale(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown snap subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runSnapPixels(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("snap pixels", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	palettePath := fs.String("palette", "", "path to a .hex or .gpl palette file")
	alphaThreshold := fs.Int("alpha-threshold", 128, "pixels with alpha below this value become transparent")
	outPath := fs.String("out", "", "output PNG path")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	if path == "" && fs.NArg() == 0 {
		return fmt.Errorf("missing path for snap pixels")
	}
	if path != "" && fs.NArg() != 0 {
		return fmt.Errorf("snap pixels takes exactly one path")
	}
	if path == "" && fs.NArg() > 1 {
		return fmt.Errorf("snap pixels takes exactly one path")
	}
	if *palettePath == "" {
		return fmt.Errorf("missing required --palette for snap pixels")
	}
	if *alphaThreshold < 1 || *alphaThreshold > 255 {
		return fmt.Errorf("invalid --alpha-threshold %d; want 1-255", *alphaThreshold)
	}

	inPath := path
	if inPath == "" {
		inPath = fs.Arg(0)
	}
	if *outPath == "" {
		*outPath = defaultSnapPixelsOutPath(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}
	pal, err := readPaletteFile(*palettePath)
	if err != nil {
		return err
	}

	thresholded := pixel.ThresholdAlpha(img, 0, uint8(*alphaThreshold))
	result := palette.Apply(thresholded, pal, false)

	resp := map[string]any{
		"out":                      *outPath,
		"fractional_pixels_zeroed": countNewlyTransparent(img, thresholded),
		"changed_pixels":           countChangedPixels(img, result),
		"palette_size":             len(pal),
		"alpha_threshold":          *alphaThreshold,
		"dry_run":                  *dryRun,
	}

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(*outPath, result); err != nil {
		return err
	}

	text := fmt.Sprintf("%s: %s\nfractional_pixels_zeroed: %d\nchanged_pixels: %d\npalette_size: %d\n",
		verb,
		*outPath,
		resp["fractional_pixels_zeroed"],
		resp["changed_pixels"],
		resp["palette_size"],
	)
	return jsonout.Write(stdout, asJSON, text, resp)
}

func runSnapScale(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("snap scale", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	factorArg := fs.String("factor", "auto", "scale factor: auto, 1, 2, 3, 4, or 8")
	outPath := fs.String("out", "", "output PNG path")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	if path == "" && fs.NArg() == 0 {
		return fmt.Errorf("missing path for snap scale")
	}
	if path != "" && fs.NArg() != 0 {
		return fmt.Errorf("snap scale takes exactly one path")
	}
	if path == "" && fs.NArg() > 1 {
		return fmt.Errorf("snap scale takes exactly one path")
	}

	inPath := path
	if inPath == "" {
		inPath = fs.Arg(0)
	}
	if *outPath == "" {
		*outPath = defaultSnapScaleOutPath(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}

	factor, forcedFactor, err := resolveSnapScaleFactor(*factorArg, img)
	if err != nil {
		return err
	}
	result := pixel.Downscale(img, factor)
	bounds := img.Bounds()
	outBounds := result.Bounds()

	resp := map[string]any{
		"out":             *outPath,
		"detected_factor": factor,
		"forced_factor":   forcedFactor,
		"in_w":            bounds.Dx(),
		"in_h":            bounds.Dy(),
		"out_w":           outBounds.Dx(),
		"out_h":           outBounds.Dy(),
		"dry_run":         *dryRun,
	}

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(*outPath, result); err != nil {
		return err
	}

	text := fmt.Sprintf("%s: %s\ndetected_factor: %d\nin: %dx%d\nout: %dx%d\n",
		verb,
		*outPath,
		factor,
		bounds.Dx(),
		bounds.Dy(),
		outBounds.Dx(),
		outBounds.Dy(),
	)
	if factor == 1 {
		text += "note: no integer upscale detected; output matches input size\n"
	}
	return jsonout.Write(stdout, asJSON, text, resp)
}

func resolveSnapScaleFactor(raw string, img image.Image) (int, any, error) {
	if strings.EqualFold(raw, "auto") {
		return pixel.DetectScale(img), nil, nil
	}
	factor, err := strconv.Atoi(raw)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid --factor %q; want auto, 1, 2, 3, 4, or 8", raw)
	}
	if factor != 1 && factor != 2 && factor != 3 && factor != 4 && factor != 8 {
		return 0, nil, fmt.Errorf("invalid --factor %d; want auto, 1, 2, 3, 4, or 8", factor)
	}
	bounds := img.Bounds()
	if bounds.Dx()%factor != 0 || bounds.Dy()%factor != 0 {
		return 0, nil, fmt.Errorf("scale factor %d does not evenly divide image size %dx%d", factor, bounds.Dx(), bounds.Dy())
	}
	return factor, factor, nil
}

func defaultSnapPixelsOutPath(inPath string) string {
	return defaultStageOutPath(inPath, "snap", "snapped.png")
}

func defaultSnapScaleOutPath(inPath string) string {
	return defaultStageOutPath(inPath, "snap", "native.png")
}

func countNewlyTransparent(before, after image.Image) int {
	bounds := before.Bounds()
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, beforeA := before.At(x, y).RGBA()
			_, _, _, afterA := after.At(x, y).RGBA()
			if beforeA > 0 && beforeA < 0xffff && afterA == 0 {
				count++
			}
		}
	}
	return count
}

func countChangedPixels(before, after image.Image) int {
	bounds := before.Bounds()
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r1, g1, b1, a1 := before.At(x, y).RGBA()
			r2, g2, b2, a2 := after.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				count++
			}
		}
	}
	return count
}
