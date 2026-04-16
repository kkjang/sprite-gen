package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("prep", runPrep)
	specreg.Register(specreg.Command{
		Name:        "prep alpha",
		Description: "Remove low-alpha background noise from a PNG",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to clean before slicing or segmenting"}},
		Flags: []specreg.Flag{
			{Name: "alpha-threshold", Default: "128", Description: "Pixels with alpha below this value become transparent"},
			{Name: "out", Description: "Output PNG path"},
			{Name: "dry-run", Default: "false", Description: "Report the output path without writing"},
		},
	})
}

func runPrep(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing prep subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "alpha":
		return runPrepAlpha(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown prep subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runPrepAlpha(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("prep alpha", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	alphaThreshold := fs.Int("alpha-threshold", 128, "pixels with alpha below this value become transparent")
	outPath := fs.String("out", "", "output PNG path")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	inPath, err := resolveSinglePathArg(path, fs, "prep alpha")
	if err != nil {
		return err
	}
	if *alphaThreshold < 1 || *alphaThreshold > 255 {
		return fmt.Errorf("invalid --alpha-threshold %d; want 1-255", *alphaThreshold)
	}
	if *outPath == "" {
		*outPath = defaultPrepAlphaOutPath(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}
	result := pixel.ThresholdAlpha(img, 0, uint8(*alphaThreshold))

	resp := map[string]any{
		"out":                      *outPath,
		"fractional_pixels_zeroed": countNewlyTransparent(img, result),
		"changed_pixels":           countChangedPixels(img, result),
		"alpha_threshold":          *alphaThreshold,
		"dry_run":                  *dryRun,
	}

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(*outPath, result); err != nil {
		return err
	}

	text := fmt.Sprintf("%s: %s\nfractional_pixels_zeroed: %d\nchanged_pixels: %d\n",
		verb,
		*outPath,
		resp["fractional_pixels_zeroed"],
		resp["changed_pixels"],
	)
	return jsonout.Write(stdout, asJSON, text, resp)
}

func defaultPrepAlphaOutPath(inPath string) string {
	return defaultStageOutPath(inPath, "prep", "clean.png")
}
