package pixel

import "image"

type Grid struct {
	Cols       int     `json:"cols"`
	Rows       int     `json:"rows"`
	CellW      int     `json:"cell_w"`
	CellH      int     `json:"cell_h"`
	OffsetX    int     `json:"offset_x"`
	OffsetY    int     `json:"offset_y"`
	Confidence float64 `json:"confidence"`
}

func GuessGrid(img image.Image) Grid {
	bounds := img.Bounds()
	colBands := occupiedBands(bounds.Dx(), func(i int) bool { return columnHasAlpha(img, bounds.Min.X+i, 0) })
	rowBands := occupiedBands(bounds.Dy(), func(i int) bool { return rowHasAlpha(img, bounds.Min.Y+i, 0) })

	if len(colBands) == 0 || len(rowBands) == 0 {
		return Grid{}
	}
	if len(colBands) == 1 && len(rowBands) == 1 {
		return Grid{}
	}

	cellW, offsetX, confX := inferAxisGrid(colBands, bounds.Dx())
	cellH, offsetY, confY := inferAxisGrid(rowBands, bounds.Dy())
	if cellW == 0 || cellH == 0 || confX == 0 || confY == 0 {
		return Grid{}
	}

	cols := bounds.Dx() / cellW
	rows := bounds.Dy() / cellH
	if cols == 0 || rows == 0 {
		return Grid{}
	}

	confidence := confX
	if confY < confidence {
		confidence = confY
	}

	return Grid{
		Cols:       cols,
		Rows:       rows,
		CellW:      cellW,
		CellH:      cellH,
		OffsetX:    offsetX,
		OffsetY:    offsetY,
		Confidence: confidence,
	}
}

type band struct {
	Start int
	End   int
}

func occupiedBands(length int, occupied func(i int) bool) []band {
	var bands []band
	start := -1
	for i := 0; i < length; i++ {
		if occupied(i) {
			if start == -1 {
				start = i
			}
			continue
		}
		if start != -1 {
			bands = append(bands, band{Start: start, End: i})
			start = -1
		}
	}
	if start != -1 {
		bands = append(bands, band{Start: start, End: length})
	}
	return bands
}

func inferAxisGrid(bands []band, total int) (cell, offset int, confidence float64) {
	if len(bands) == 0 {
		return 0, 0, 0
	}
	if len(bands) == 1 {
		if bands[0].Start != 0 || total-bands[0].End > 1 {
			return 0, 0, 0
		}
		return total, 0, 1
	}

	cell = bands[1].Start - bands[0].Start
	if cell <= 0 || total%cell != 0 {
		return 0, 0, 0
	}

	offset = bands[0].Start % cell
	if offset != bands[0].Start {
		return 0, 0, 0
	}

	penalty := 0.0
	for i := 0; i < len(bands); i++ {
		wantStart := offset + i*cell
		gotStart := bands[i].Start
		gotEnd := bands[i].End
		if abs(wantStart-gotStart) > 1 {
			penalty += 0.25
		}

		cellEnd := wantStart + cell
		if gotEnd > cellEnd || gotStart < wantStart {
			return 0, 0, 0
		}
		if cellEnd-gotEnd > 1 {
			penalty += 0.1
		}
	}

	confidence = 1 - penalty/float64(len(bands))
	if confidence < 0 {
		confidence = 0
	}
	return cell, offset, confidence
}

func columnHasAlpha(img image.Image, x int, alphaMin uint8) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		_, _, _, a16 := img.At(x, y).RGBA()
		if uint8(a16>>8) > alphaMin {
			return true
		}
	}
	return false
}

func rowHasAlpha(img image.Image, y int, alphaMin uint8) bool {
	bounds := img.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		_, _, _, a16 := img.At(x, y).RGBA()
		if uint8(a16>>8) > alphaMin {
			return true
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
