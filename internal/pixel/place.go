package pixel

import (
	"image"
	"image/draw"
)

// PlaceInCell crops src to srcRect and draws it into a transparent cell-sized
// image at the requested offset. Pixels outside the cell are clipped.
func PlaceInCell(src *image.NRGBA, srcRect image.Rectangle, cell image.Point, offset image.Point) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, maxInt(cell.X, 0), maxInt(cell.Y, 0)))
	if src == nil || srcRect.Empty() || cell.X <= 0 || cell.Y <= 0 {
		return out
	}

	dstRect := image.Rectangle{Min: offset, Max: offset.Add(srcRect.Size())}
	draw.Draw(out, dstRect, src, srcRect.Min, draw.Src)
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
