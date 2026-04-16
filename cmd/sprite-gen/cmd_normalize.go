package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/kkjang/sprite-gen/internal/detail"
	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("normalize", runNormalize)
	specreg.Register(specreg.Command{
		Name:        "normalize detail",
		Description: "Scale a PNG toward a target visible subject height",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to normalize"}},
		Flags: []specreg.Flag{
			{Name: "target-height", Description: "Desired visible subject height in pixels"},
			{Name: "factor", Description: "Explicit integer downscale factor"},
			{Name: "alpha-threshold", Default: "8", Description: "Minimum alpha used when measuring the visible bbox"},
			{Name: "out", Description: "Output PNG path"},
			{Name: "dry-run", Default: "false", Description: "Report output path without writing"},
		},
	})
}

func runNormalize(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing normalize subcommand; try: sprite-gen spec")
	}

	switch args[0] {
	case "detail":
		return runNormalizeDetail(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown normalize subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runNormalizeDetail(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("normalize detail", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	targetHeight := fs.Int("target-height", 0, "desired visible subject height in pixels")
	factor := fs.Int("factor", 0, "explicit integer downscale factor")
	alphaThreshold := fs.Int("alpha-threshold", int(pixel.DefaultBBoxAlphaThreshold), "minimum alpha used when measuring the visible bbox")
	outPath := fs.String("out", "", "output PNG path")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	inPath, err := resolveSinglePathArg(path, fs, "normalize detail")
	if err != nil {
		return err
	}
	if err := validateNormalizeDetailFlags(*targetHeight, *factor, *alphaThreshold); err != nil {
		return err
	}
	if *outPath == "" {
		*outPath = defaultNormalizeDetailOutPath(inPath)
	}

	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}
	result, err := detail.Normalize(img, detail.Options{
		TargetHeight:   *targetHeight,
		Factor:         *factor,
		AlphaThreshold: uint8(*alphaThreshold),
	})
	if err != nil {
		return err
	}

	resp := map[string]any{
		"out":             *outPath,
		"factor":          result.Factor,
		"input_w":         result.InputW,
		"input_h":         result.InputH,
		"output_w":        result.OutputW,
		"output_h":        result.OutputH,
		"input_bbox_h":    result.InputBBoxH,
		"output_bbox_h":   result.OutputBBoxH,
		"alpha_threshold": *alphaThreshold,
		"unchanged":       result.Unchanged,
		"dry_run":         *dryRun,
	}

	verb := "wrote"
	if *dryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(*outPath, result.Image); err != nil {
		return err
	}

	text := fmt.Sprintf("%s: %s\nfactor: %d\ninput: %dx%d\noutput: %dx%d\ninput_bbox_h: %d\noutput_bbox_h: %d\nunchanged: %t\n",
		verb,
		*outPath,
		result.Factor,
		result.InputW,
		result.InputH,
		result.OutputW,
		result.OutputH,
		result.InputBBoxH,
		result.OutputBBoxH,
		result.Unchanged,
	)
	return jsonout.Write(stdout, asJSON, text, resp)
}

func validateNormalizeDetailFlags(targetHeight, factor, alphaThreshold int) error {
	if targetHeight != 0 && factor != 0 {
		return fmt.Errorf("provide exactly one of --target-height or --factor")
	}
	if targetHeight == 0 && factor == 0 {
		return fmt.Errorf("provide exactly one of --target-height or --factor")
	}
	if targetHeight < 0 {
		return fmt.Errorf("invalid --target-height %d; want greater than 0", targetHeight)
	}
	if factor < 0 {
		return fmt.Errorf("invalid --factor %d; want an integer greater than or equal to 1", factor)
	}
	if targetHeight == 0 && factor < 1 {
		return fmt.Errorf("invalid --factor %d; want an integer greater than or equal to 1", factor)
	}
	if factor == 0 && targetHeight < 1 {
		return fmt.Errorf("invalid --target-height %d; want greater than 0", targetHeight)
	}
	if alphaThreshold < 0 || alphaThreshold > 255 {
		return fmt.Errorf("invalid --alpha-threshold %d; want 0-255", alphaThreshold)
	}
	return nil
}

func defaultNormalizeDetailOutPath(inPath string) string {
	return defaultStageOutPath(inPath, "normalize", "detail.png")
}
