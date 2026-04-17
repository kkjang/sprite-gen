package main

import (
	"flag"
	"fmt"
	"io"
	"strconv"

	internalexport "github.com/kkjang/sprite-gen/internal/export"
	_ "github.com/kkjang/sprite-gen/internal/export/formats/gif"
	_ "github.com/kkjang/sprite-gen/internal/export/formats/sheet"
	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("export", runExport)
	specreg.Register(specreg.Command{
		Name:        "export",
		Description: "Export a frame-set directory to a registered format",
		Args:        []specreg.Arg{{Name: "dir", Required: true, Description: "Directory containing frame PNGs and optional manifest.json"}},
		Flags: []specreg.Flag{
			{Name: "format", Description: "Registered export format name"},
			{Name: "list-formats", Default: "false", Description: "List available export formats and exit"},
			{Name: "out", Description: "Output directory for exported artifacts"},
			{Name: "dry-run", Default: "false", Description: "Report output path without writing"},
			{Name: "fps", Default: "8", Description: "GIF frame rate in frames per second"},
			{Name: "scale", Default: "1", Description: "GIF integer preview upscale factor"},
			{Name: "loop", Default: "true", Description: "GIF loop forever when true"},
			{Name: "cols", Description: "sheet columns; default is auto-packed"},
			{Name: "padding", Default: "0", Description: "sheet padding in pixels between cells"},
		},
	})
}

func runExport(args []string, stdout, _ io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	formatName := fs.String("format", "", "registered export format name")
	listFormats := fs.Bool("list-formats", false, "list available export formats and exit")
	outPath := fs.String("out", "", "output directory for exported artifacts")
	dryRun := fs.Bool("dry-run", false, "report output path without writing")
	fps := fs.Int("fps", 8, "GIF frame rate in frames per second")
	scale := fs.Int("scale", 1, "GIF integer preview upscale factor")
	loop := fs.Bool("loop", true, "GIF loop forever when true")
	cols := fs.Int("cols", 0, "sheet columns; default is auto-packed")
	padding := fs.Int("padding", 0, "sheet padding in pixels between cells")
	path, parseArgs := splitSinglePathArg(args)
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}

	if *listFormats {
		if path != "" || fs.NArg() != 0 {
			return fmt.Errorf("export --list-formats takes no directory")
		}
		if *formatName != "" {
			return fmt.Errorf("export --list-formats does not take --format")
		}
		if *outPath != "" {
			return fmt.Errorf("export --list-formats does not take --out")
		}
		return writeFormatList(stdout, asJSON)
	}

	inDir, err := resolveSinglePathArg(path, fs, "export")
	if err != nil {
		return err
	}
	if *formatName == "" {
		return fmt.Errorf("missing required --format for export")
	}
	if *fps <= 0 {
		return fmt.Errorf("invalid --fps %d; want a positive integer", *fps)
	}
	if *scale <= 0 {
		return fmt.Errorf("invalid --scale %d; want a positive integer", *scale)
	}
	if *cols < 0 {
		return fmt.Errorf("invalid --cols %d; want 0 or a positive integer", *cols)
	}
	if *padding < 0 {
		return fmt.Errorf("invalid --padding %d; want 0 or greater", *padding)
	}

	format, err := internalexport.Get(*formatName)
	if err != nil {
		return err
	}
	subject := outputSubject(inDir)
	if *outPath == "" {
		*outPath = defaultExportOut(inDir, format.Name())
	}

	ctx, err := internalexport.LoadContext(inDir, format.Name(), subject, *outPath, *dryRun, exportOptions(*fps, *scale, *loop, *cols, *padding))
	if err != nil {
		return err
	}

	result, err := format.Export(ctx)
	if err != nil {
		return err
	}
	return jsonout.Write(stdout, asJSON, result.Text, result.Data)
}

type exportFormatResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func writeFormatList(stdout io.Writer, asJSON bool) error {
	formats := internalexport.All()
	resp := make([]exportFormatResponse, len(formats))
	text := ""
	for i, format := range formats {
		resp[i] = exportFormatResponse{Name: format.Name(), Description: format.Description()}
		text += fmt.Sprintf("%-10s %s\n", format.Name(), format.Description())
	}
	return jsonout.Write(stdout, asJSON, text, map[string]any{"formats": resp})
}

func exportOptions(fps, scale int, loop bool, cols, padding int) map[string]string {
	options := map[string]string{
		"fps":     strconv.Itoa(fps),
		"scale":   strconv.Itoa(scale),
		"loop":    strconv.FormatBool(loop),
		"padding": strconv.Itoa(padding),
	}
	if cols > 0 {
		options["cols"] = strconv.Itoa(cols)
	}
	return options
}
