package sheetpng

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"strconv"

	internalexport "github.com/kkjang/sprite-gen/internal/export"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

type SheetPNG struct{}

func (SheetPNG) Name() string {
	return "sheet-png"
}

func (SheetPNG) Description() string {
	return "Pack frames into a sprite sheet PNG"
}

func (SheetPNG) Export(ctx *internalexport.Context) (*internalexport.Result, error) {
	if len(ctx.Frames) == 0 {
		return nil, fmt.Errorf("export sheet-png: no frames to export")
	}

	cols, err := intOption(ctx.Options, "cols", 0)
	if err != nil {
		return nil, err
	}
	padding, err := intOption(ctx.Options, "padding", 0)
	if err != nil {
		return nil, err
	}
	if cols < 0 {
		return nil, fmt.Errorf("invalid --cols value %d; want a positive integer", cols)
	}
	if padding < 0 {
		return nil, fmt.Errorf("invalid --padding value %d; want 0 or greater", padding)
	}
	if cols == 0 {
		cols = int(math.Ceil(math.Sqrt(float64(len(ctx.Frames)))))
		if cols < 1 {
			cols = 1
		}
	}

	cellW, cellH, mixedSizes := maxCell(ctx.Frames)
	rows := (len(ctx.Frames) + cols - 1) / cols
	sheetW := cols*cellW + max(0, cols-1)*padding
	sheetH := rows*cellH + max(0, rows-1)*padding
	sheet := image.NewNRGBA(image.Rect(0, 0, sheetW, sheetH))

	for i, frame := range ctx.Frames {
		col := i % cols
		row := i / cols
		x := col * (cellW + padding)
		y := row * (cellH + padding)
		bounds := frame.Image.Bounds()
		dst := image.Rect(x, y, x+bounds.Dx(), y+bounds.Dy())
		draw.Draw(sheet, dst, frame.Image, bounds.Min, draw.Src)
	}

	verb := "wrote"
	if ctx.DryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(ctx.OutPath, sheet); err != nil {
		return nil, err
	}

	data := map[string]any{
		"format":      ctx.Format,
		"out":         ctx.OutPath,
		"frames":      len(ctx.Frames),
		"cols":        cols,
		"rows":        rows,
		"sheet_w":     sheetW,
		"sheet_h":     sheetH,
		"cell_w":      cellW,
		"cell_h":      cellH,
		"padding":     padding,
		"mixed_sizes": mixedSizes,
		"dry_run":     ctx.DryRun,
	}
	text := fmt.Sprintf("%s: %s (%d frames, %dx%d sheet)\n", verb, ctx.OutPath, len(ctx.Frames), sheetW, sheetH)
	return &internalexport.Result{Text: text, Data: data}, nil
}

func init() {
	internalexport.Register(SheetPNG{})
}

func intOption(options map[string]string, key string, defaultValue int) (int, error) {
	raw := options[key]
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid --%s value %q; want an integer", key, raw)
	}
	return value, nil
}

func maxCell(frames []internalexport.Frame) (int, int, bool) {
	cellW := 0
	cellH := 0
	mixedSizes := false
	for i, frame := range frames {
		bounds := frame.Image.Bounds()
		w := bounds.Dx()
		h := bounds.Dy()
		if w > cellW {
			cellW = w
		}
		if h > cellH {
			cellH = h
		}
		if i == 0 {
			continue
		}
		if w != frames[0].Image.Bounds().Dx() || h != frames[0].Image.Bounds().Dy() {
			mixedSizes = true
		}
	}
	return cellW, cellH, mixedSizes
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
