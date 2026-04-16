package pixel

import (
	"image"
	"image/color"
)

// AlphaMask returns a binary alpha mask where pixels with alpha >= threshold
// are foreground and all other pixels are background.
func AlphaMask(src image.Image, threshold uint8) *image.Alpha {
	bounds := src.Bounds()
	out := image.NewAlpha(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := rgba8(src.At(x, y))
			if a >= threshold {
				out.SetAlpha(x, y, color.Alpha{A: 0xff})
			}
		}
	}
	return out
}

// MorphErode shrinks the foreground in mask using a 3x3 square kernel.
func MorphErode(mask *image.Alpha, iterations int) *image.Alpha {
	if mask == nil {
		return nil
	}
	if iterations <= 0 {
		return cloneAlpha(mask)
	}

	out := cloneAlpha(mask)
	for i := 0; i < iterations; i++ {
		out = morphStep(out, true)
	}
	return out
}

// MorphDilate grows the foreground in mask using a 3x3 square kernel.
func MorphDilate(mask *image.Alpha, iterations int) *image.Alpha {
	if mask == nil {
		return nil
	}
	if iterations <= 0 {
		return cloneAlpha(mask)
	}

	out := cloneAlpha(mask)
	for i := 0; i < iterations; i++ {
		out = morphStep(out, false)
	}
	return out
}

func morphStep(mask *image.Alpha, erode bool) *image.Alpha {
	bounds := mask.Bounds()
	out := image.NewAlpha(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if erode {
				if allNeighborsForeground(mask, x, y) {
					out.SetAlpha(x, y, color.Alpha{A: 0xff})
				}
				continue
			}
			if anyNeighborForeground(mask, x, y) {
				out.SetAlpha(x, y, color.Alpha{A: 0xff})
			}
		}
	}
	return out
}

func allNeighborsForeground(mask *image.Alpha, x, y int) bool {
	bounds := mask.Bounds()
	for ny := y - 1; ny <= y+1; ny++ {
		for nx := x - 1; nx <= x+1; nx++ {
			if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
				return false
			}
			if mask.AlphaAt(nx, ny).A == 0 {
				return false
			}
		}
	}
	return true
}

func anyNeighborForeground(mask *image.Alpha, x, y int) bool {
	bounds := mask.Bounds()
	for ny := y - 1; ny <= y+1; ny++ {
		for nx := x - 1; nx <= x+1; nx++ {
			if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
				continue
			}
			if mask.AlphaAt(nx, ny).A != 0 {
				return true
			}
		}
	}
	return false
}

func cloneAlpha(src *image.Alpha) *image.Alpha {
	if src == nil {
		return nil
	}
	out := image.NewAlpha(src.Bounds())
	copy(out.Pix, src.Pix)
	return out
}
