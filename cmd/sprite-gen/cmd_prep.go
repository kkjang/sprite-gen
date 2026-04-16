package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/kkjang/sprite-gen/internal/background"
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
			{Name: "dry-run", Default: "false", Description: "Report output path without writing"},
		},
	})
	specreg.Register(specreg.Command{
		Name:        "prep background",
		Description: "Remove fake or opaque backgrounds from a PNG",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to clean before slicing or segmenting"}},
		Flags: []specreg.Flag{
			{Name: "method", Default: string(background.MethodAuto), Description: "Background cleanup method: auto, key, or edge"},
			{Name: "color", Description: "Key color as #RRGGBB when using --method key"},
			{Name: "tolerance", Default: "12", Description: "Per-channel match threshold for background removal"},
			{Name: "connectivity", Default: "4", Description: "Flood-fill connectivity for edge removal: 4 or 8"},
			{Name: "out", Description: "Output PNG path"},
			{Name: "dry-run", Default: "false", Description: "Report output path without writing"},
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
	case "background":
		return runPrepBackground(args[1:], stdout, asJSON)
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

func runPrepBackground(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("prep background", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	methodFlag := fs.String("method", string(background.MethodAuto), "background cleanup method: auto, key, or edge")
	colorFlag := fs.String("color", "", "key color as #RRGGBB when using --method key")
	tolerance := fs.Int("tolerance", 12, "per-channel match threshold for background removal")
	connectivity := fs.Int("connectivity", 4, "flood-fill connectivity for edge removal: 4 or 8")
	outPath := fs.String("out", "", "output PNG path")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	inPath, err := resolveSinglePathArg(path, fs, "prep background")
	if err != nil {
		return err
	}
	if *tolerance < 0 || *tolerance > 255 {
		return fmt.Errorf("invalid --tolerance %d; want 0-255", *tolerance)
	}
	if *connectivity != 4 && *connectivity != 8 {
		return fmt.Errorf("invalid --connectivity %d; want 4 or 8", *connectivity)
	}
	method, err := background.ParseMethod(*methodFlag)
	if err != nil {
		return err
	}
	options := background.Options{Method: method, Tolerance: uint8(*tolerance), Connectivity: *connectivity}
	resp := map[string]any{
		"tolerance":    *tolerance,
		"connectivity": *connectivity,
		"dry_run":      *dryRun,
	}
	if *colorFlag != "" {
		parsed, err := background.ParseHexColor(*colorFlag)
		if err != nil {
			return err
		}
		options.KeyColor = parsed
		options.HasKeyColor = true
		resp["key_color"] = fmt.Sprintf("#%02X%02X%02X", parsed.R, parsed.G, parsed.B)
	}
	if method == background.MethodKey && !options.HasKeyColor {
		return fmt.Errorf("missing required --color for prep background --method key")
	}
	if *outPath == "" {
		*outPath = defaultPrepBackgroundOutPath(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}
	result, err := background.Remove(img, options)
	if err != nil {
		return err
	}

	resp["out"] = *outPath
	resp["method"] = string(result.Method)
	resp["removed_pixels"] = result.RemovedPixels
	resp["changed_pixels"] = result.ChangedPixels

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(*outPath, result.Image); err != nil {
		return err
	}

	text := fmt.Sprintf("%s: %s\nmethod: %s\nremoved_pixels: %d\nchanged_pixels: %d\n",
		verb,
		*outPath,
		result.Method,
		result.RemovedPixels,
		result.ChangedPixels,
	)
	return jsonout.Write(stdout, asJSON, text, resp)
}

func defaultPrepAlphaOutPath(inPath string) string {
	return defaultStageOutPath(inPath, "prep", "clean.png")
}

func defaultPrepBackgroundOutPath(inPath string) string {
	return defaultStageOutPath(inPath, "prep", "background.png")
}
