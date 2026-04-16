package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/palette"
	"github.com/kkjang/sprite-gen/internal/pixel"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("palette", runPalette)
	specreg.Register(specreg.Command{
		Name:        "palette apply",
		Description: "Apply a palette to a PNG",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to recolor"}},
		Flags:       []specreg.Flag{{Name: "palette", Description: "Path to a .hex or .gpl palette file"}, {Name: "dither", Default: "false", Description: "Enable Floyd-Steinberg dithering"}, {Name: "out", Description: "Output PNG path"}, {Name: "dry-run", Default: "false", Description: "Report the output path without writing"}},
	})
	specreg.Register(specreg.Command{
		Name:        "palette extract",
		Description: "Extract a palette from a PNG",
		Args:        []specreg.Arg{{Name: "path", Required: true, Description: "PNG image to sample"}},
		Flags:       []specreg.Flag{{Name: "max", Default: "16", Description: "Maximum colors to emit"}, {Name: "format", Default: "hex", Description: "Palette format: hex or gpl"}, {Name: "out", Description: "Output palette path; use - for stdout"}, {Name: "dry-run", Default: "false", Description: "Report the output path without writing"}},
	})
}

func runPalette(args []string, stdout, _ io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing palette subcommand; try: sprite-gen spec")
	}
	switch args[0] {
	case "extract":
		return runPaletteExtract(args[1:], stdout, asJSON)
	case "apply":
		return runPaletteApply(args[1:], stdout, asJSON)
	default:
		return fmt.Errorf("unknown palette subcommand %q; try: sprite-gen spec", args[0])
	}
}

func runPaletteExtract(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("palette extract", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	maxColors := fs.Int("max", 16, "maximum colors to emit")
	format := fs.String("format", "hex", "palette format: hex or gpl")
	outPath := fs.String("out", "", "output palette path; use - for stdout")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	if path == "" && fs.NArg() == 0 {
		return fmt.Errorf("missing path for palette extract")
	}
	if path != "" && fs.NArg() != 0 {
		return fmt.Errorf("palette extract takes exactly one path")
	}
	if path == "" && fs.NArg() > 1 {
		return fmt.Errorf("palette extract takes exactly one path")
	}
	if path == "" {
		path = fs.Arg(0)
	}
	if *maxColors <= 0 {
		return fmt.Errorf("--max must be greater than 0")
	}
	formatName := strings.ToLower(*format)
	if *outPath == "" {
		*outPath = defaultPaletteExtractOutPath(path, formatName, *maxColors)
	}

	img, err := pixel.LoadPNG(path)
	if err != nil {
		return err
	}
	pal := palette.Extract(img, *maxColors)
	if len(pal) == 0 {
		return fmt.Errorf("extract palette from %q: no visible colors found", path)
	}

	var buf bytes.Buffer
	if err := writePalette(&buf, formatName, filepath.Base(path), pal); err != nil {
		return err
	}

	resp := map[string]any{"colors": hexStrings(pal), "count": len(pal), "format": formatName, "out": *outPath, "dry_run": *dryRun}
	if *outPath == "-" {
		if *dryRun {
			text := fmt.Sprintf("would write: -\ncount: %d\nformat: %s\n", len(pal), formatName)
			return jsonout.Write(stdout, asJSON, text, resp)
		}
		return jsonout.Write(stdout, asJSON, buf.String(), resp)
	}

	if *dryRun {
		text := fmt.Sprintf("would write: %s\ncount: %d\nformat: %s\n", *outPath, len(pal), formatName)
		return jsonout.Write(stdout, asJSON, text, resp)
	}
	if err := writeFile(*outPath, buf.Bytes()); err != nil {
		return err
	}
	text := fmt.Sprintf("wrote: %s\ncount: %d\nformat: %s\n", *outPath, len(pal), formatName)
	return jsonout.Write(stdout, asJSON, text, resp)
}

func runPaletteApply(args []string, stdout io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("palette apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	palettePath := fs.String("palette", "", "path to a .hex or .gpl palette file")
	dither := fs.Bool("dither", false, "enable Floyd-Steinberg dithering")
	outPath := fs.String("out", "", "output PNG path")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	if path == "" && fs.NArg() == 0 {
		return fmt.Errorf("missing path for palette apply")
	}
	if path != "" && fs.NArg() != 0 {
		return fmt.Errorf("palette apply takes exactly one path")
	}
	if path == "" && fs.NArg() > 1 {
		return fmt.Errorf("palette apply takes exactly one path")
	}
	if *palettePath == "" {
		return fmt.Errorf("missing required --palette for palette apply")
	}

	inPath := path
	if inPath == "" {
		inPath = fs.Arg(0)
	}
	if *outPath == "" {
		*outPath = defaultApplyOutPath(inPath)
	}
	img, err := pixel.LoadPNG(inPath)
	if err != nil {
		return err
	}
	pal, err := readPaletteFile(*palettePath)
	if err != nil {
		return err
	}
	result := palette.Apply(img, pal, *dither)
	resp := map[string]any{
		"out":        *outPath,
		"colors_in":  countUniqueRGB(img),
		"colors_out": countUniqueRGB(result),
		"dither":     *dither,
		"dry_run":    *dryRun,
	}
	textVerb := "wrote"
	if *dryRun {
		textVerb = "would write"
	} else if err := pixel.SavePNG(*outPath, result); err != nil {
		return err
	}
	text := fmt.Sprintf("%s: %s\ncolors in: %d\ncolors out: %d\n", textVerb, *outPath, resp["colors_in"], resp["colors_out"])
	return jsonout.Write(stdout, asJSON, text, resp)
}

func writePalette(w io.Writer, format, name string, pal []color.NRGBA) error {
	switch format {
	case "hex":
		return palette.WriteHex(w, pal)
	case "gpl":
		return palette.WriteGPL(w, name, pal)
	default:
		return fmt.Errorf("unsupported palette format %q; want hex or gpl", format)
	}
}

func readPaletteFile(path string) ([]color.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open palette %q: %w", path, err)
	}
	defer f.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".hex":
		return palette.ReadHex(f)
	case ".gpl":
		return palette.ReadGPL(f)
	default:
		return nil, fmt.Errorf("unsupported palette file %q; want .hex or .gpl", path)
	}
}

func writeFile(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory for %q: %w", path, err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return fmt.Errorf("write file %q: %w", path, err)
	}
	return nil
}

func defaultApplyOutPath(inPath string) string {
	return defaultStageOutPath(inPath, "palette", "applied.png")
}

func hexStrings(pal []color.NRGBA) []string {
	out := make([]string, len(pal))
	for i, c := range pal {
		out[i] = fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return out
}

func countUniqueRGB(img image.Image) int {
	bounds := img.Bounds()
	seen := map[[3]uint8]struct{}{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			seen[[3]uint8{c.R, c.G, c.B}] = struct{}{}
		}
	}
	return len(seen)
}

func splitSinglePathArg(args []string) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", args
	}
	return args[0], args[1:]
}
