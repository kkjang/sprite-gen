package sheet

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"path/filepath"
	"strconv"

	internalexport "github.com/kkjang/sprite-gen/internal/export"
	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
	internalsegment "github.com/kkjang/sprite-gen/internal/segment"
)

type Sheet struct{}

func (Sheet) Name() string {
	return "sheet"
}

func (Sheet) Description() string {
	return "Pack frames into a sprite sheet PNG plus JSON manifest"
}

func (Sheet) Export(ctx *internalexport.Context) (*internalexport.Result, error) {
	if len(ctx.Frames) == 0 {
		return nil, fmt.Errorf("export sheet: no frames to export")
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
	cellW, cellH, mixedSizes := maxCell(ctx.Frames)
	layout := packFrames(ctx, cols, cellW, cellH, padding)
	rows := layout.Rows
	cols = layout.Cols
	sheetW := layout.SheetW
	sheetH := layout.SheetH
	sheet := image.NewNRGBA(image.Rect(0, 0, sheetW, sheetH))

	for i, frame := range ctx.Frames {
		bounds := frame.Image.Bounds()
		frameRect := layout.FrameRects[i]
		dst := image.Rect(frameRect.X, frameRect.Y, frameRect.X+bounds.Dx(), frameRect.Y+bounds.Dy())
		draw.Draw(sheet, dst, frame.Image, bounds.Min, draw.Src)
	}

	pngPath, manifestPath := sheetOutputPaths(ctx.Subject, ctx.OutPath)
	sheetName := filepath.Base(pngPath)

	verb := "wrote"
	if ctx.DryRun {
		verb = "would write"
	} else if err := pixel.SavePNG(pngPath, sheet); err != nil {
		return nil, err
	} else if err := manifest.Write(manifestPath, buildSheetManifest(ctx, layout, sheetName, sheetW, sheetH, cols, rows, cellW, cellH)); err != nil {
		return nil, fmt.Errorf("write sheet manifest %q after PNG %q: %w", manifestPath, pngPath, err)
	}

	data := map[string]any{
		"format":      ctx.Format,
		"out":         ctx.OutPath,
		"png":         pngPath,
		"manifest":    manifestPath,
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
	text := fmt.Sprintf("%s: %s and %s (%d frames, %dx%d sheet)\n", verb, pngPath, manifestPath, len(ctx.Frames), sheetW, sheetH)
	return &internalexport.Result{Text: text, Data: data}, nil
}

func init() {
	internalexport.Register(Sheet{})
}

type packedLayout struct {
	Cols       int
	Rows       int
	SheetW     int
	SheetH     int
	FrameRects []manifest.Rect
	FrameRows  []*int
	FrameCols  []*int
}

func packFrames(ctx *internalexport.Context, cols, cellW, cellH, padding int) packedLayout {
	if cols == 0 && hasFrameGridPositions(ctx.Frames) {
		return packPositionedFrames(ctx.Frames, cellW, cellH, padding)
	}
	if cols == 0 {
		layout, ok := packInferredSourceGrid(ctx.Frames, cellW, cellH, padding)
		if ok {
			return layout
		}
	}
	if cols == 0 && len(ctx.FrameRows) > 0 {
		return packFrameRows(ctx, cols, cellW, cellH, padding)
	}
	if cols == 0 {
		cols = int(math.Ceil(math.Sqrt(float64(len(ctx.Frames)))))
		if cols < 1 {
			cols = 1
		}
	}
	frames := ctx.Frames
	rows := (len(frames) + cols - 1) / cols
	layout := packedLayout{
		Cols:       cols,
		Rows:       rows,
		SheetW:     cols*cellW + max(0, cols-1)*padding,
		SheetH:     rows*cellH + max(0, rows-1)*padding,
		FrameRects: make([]manifest.Rect, len(frames)),
		FrameRows:  make([]*int, len(frames)),
		FrameCols:  make([]*int, len(frames)),
	}
	for i, frame := range frames {
		col := i % cols
		row := i / cols
		bounds := frame.Image.Bounds()
		layout.FrameRects[i] = manifest.Rect{
			X: col * (cellW + padding),
			Y: row * (cellH + padding),
			W: bounds.Dx(),
			H: bounds.Dy(),
		}
		layout.FrameRows[i] = intPtr(row)
		layout.FrameCols[i] = intPtr(col)
	}
	return layout
}

func packPositionedFrames(frames []internalexport.Frame, cellW, cellH, padding int) packedLayout {
	maxRow := 0
	maxCol := 0
	for _, frame := range frames {
		if *frame.Row > maxRow {
			maxRow = *frame.Row
		}
		if *frame.Col > maxCol {
			maxCol = *frame.Col
		}
	}
	layout := packedLayout{
		Cols:       maxCol + 1,
		Rows:       maxRow + 1,
		SheetW:     (maxCol+1)*cellW + max(0, maxCol)*padding,
		SheetH:     (maxRow+1)*cellH + max(0, maxRow)*padding,
		FrameRects: make([]manifest.Rect, len(frames)),
		FrameRows:  make([]*int, len(frames)),
		FrameCols:  make([]*int, len(frames)),
	}
	for i, frame := range frames {
		bounds := frame.Image.Bounds()
		layout.FrameRects[i] = manifest.Rect{
			X: *frame.Col * (cellW + padding),
			Y: *frame.Row * (cellH + padding),
			W: bounds.Dx(),
			H: bounds.Dy(),
		}
		layout.FrameRows[i] = intPtr(*frame.Row)
		layout.FrameCols[i] = intPtr(*frame.Col)
	}
	return layout
}

func packInferredSourceGrid(frames []internalexport.Frame, cellW, cellH, padding int) (packedLayout, bool) {
	if !hasSourceGridRects(frames) {
		return packedLayout{}, false
	}
	components := make([]internalsegment.Component, len(frames))
	for i, frame := range frames {
		components[i] = internalsegment.Component{
			ID:   i,
			BBox: image.Rect(frame.Rect.X, frame.Rect.Y, frame.Rect.X+frame.Rect.W, frame.Rect.Y+frame.Rect.H),
		}
	}
	rows := internalsegment.GroupRows(components)
	if len(rows) <= 1 {
		return packedLayout{}, false
	}

	rowByIndex := make([]int, len(frames))
	colByIndex := make([]int, len(frames))
	maxCols := 0
	for rowIndex, row := range rows {
		if len(row.Components) > maxCols {
			maxCols = len(row.Components)
		}
		for colIndex, component := range row.Components {
			rowByIndex[component.ID] = rowIndex
			colByIndex[component.ID] = colIndex
		}
	}
	layout := packedLayout{
		Cols:       maxCols,
		Rows:       len(rows),
		SheetW:     maxCols*cellW + max(0, maxCols-1)*padding,
		SheetH:     len(rows)*cellH + max(0, len(rows)-1)*padding,
		FrameRects: make([]manifest.Rect, len(frames)),
		FrameRows:  make([]*int, len(frames)),
		FrameCols:  make([]*int, len(frames)),
	}
	for i, frame := range frames {
		bounds := frame.Image.Bounds()
		row := rowByIndex[i]
		col := colByIndex[i]
		layout.FrameRects[i] = manifest.Rect{
			X: col * (cellW + padding),
			Y: row * (cellH + padding),
			W: bounds.Dx(),
			H: bounds.Dy(),
		}
		layout.FrameRows[i] = intPtr(row)
		layout.FrameCols[i] = intPtr(col)
	}
	return layout, true
}

func packFrameRows(ctx *internalexport.Context, cols, cellW, cellH, padding int) packedLayout {
	maxCols := cols
	for _, row := range ctx.FrameRows {
		if row.Count > maxCols {
			maxCols = row.Count
		}
	}
	if maxCols < 1 {
		maxCols = 1
	}
	layout := packedLayout{
		Cols:       maxCols,
		Rows:       len(ctx.FrameRows),
		SheetW:     maxCols*cellW + max(0, maxCols-1)*padding,
		SheetH:     len(ctx.FrameRows)*cellH + max(0, len(ctx.FrameRows)-1)*padding,
		FrameRects: make([]manifest.Rect, len(ctx.Frames)),
		FrameRows:  make([]*int, len(ctx.Frames)),
		FrameCols:  make([]*int, len(ctx.Frames)),
	}
	for rowIndex, row := range ctx.FrameRows {
		for i := 0; i < row.Count; i++ {
			frameIndex := row.Start + i
			bounds := ctx.Frames[frameIndex].Image.Bounds()
			layout.FrameRects[frameIndex] = manifest.Rect{
				X: i * (cellW + padding),
				Y: rowIndex * (cellH + padding),
				W: bounds.Dx(),
				H: bounds.Dy(),
			}
			layout.FrameRows[frameIndex] = intPtr(rowIndex)
			layout.FrameCols[frameIndex] = intPtr(i)
		}
	}
	return layout
}

func sheetOutputPaths(subject, outDir string) (string, string) {
	base := subject + "_sheet"
	return filepath.Join(outDir, base+".png"), filepath.Join(outDir, base+".json")
}

func buildSheetManifest(ctx *internalexport.Context, layout packedLayout, sheetName string, sheetW, sheetH, cols, rows, cellW, cellH int) *manifest.Manifest {
	out := &manifest.Manifest{}
	if ctx.Manifest != nil {
		*out = *ctx.Manifest
	}
	if out.Source == "" {
		out.Source = ctx.FrameDir
	}
	out.CellW = cellW
	out.CellH = cellH
	out.Cols = cols
	out.Rows = rows
	out.Sheet = sheetName
	out.SheetSize = &manifest.Size{W: sheetW, H: sheetH}
	out.Frames = make([]manifest.Frame, len(ctx.Frames))
	for i, frame := range ctx.Frames {
		out.Frames[i] = manifest.Frame{
			Index: frame.Index,
			Path:  frame.Path,
			Rect:  layout.FrameRects[i],
			Row:   layout.FrameRows[i],
			Col:   layout.FrameCols[i],
			Tag:   frame.Tag,
		}
		if ctx.Manifest == nil || i >= len(ctx.Manifest.Frames) {
			continue
		}
		meta := ctx.Manifest.Frames[i]
		if meta.Path != "" {
			out.Frames[i].Path = meta.Path
		}
		if meta.Row != nil {
			out.Frames[i].Row = meta.Row
		}
		if meta.Col != nil {
			out.Frames[i].Col = meta.Col
		}
		out.Frames[i].Pivot = meta.Pivot
		out.Frames[i].DurationMS = meta.DurationMS
		out.Frames[i].Tag = meta.Tag
	}
	return out
}

func hasSourceGridRects(frames []internalexport.Frame) bool {
	if len(frames) == 0 {
		return false
	}
	first := frames[0].Rect
	for _, frame := range frames[1:] {
		if frame.Rect.X != first.X || frame.Rect.Y != first.Y || frame.Rect.W != first.W || frame.Rect.H != first.H {
			return true
		}
	}
	return false
}

func intPtr(value int) *int {
	return &value
}

func hasFrameGridPositions(frames []internalexport.Frame) bool {
	if len(frames) == 0 {
		return false
	}
	for _, frame := range frames {
		if frame.Row == nil || frame.Col == nil {
			return false
		}
	}
	return true
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
