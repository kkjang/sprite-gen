package pixel

import (
	"image"
	"image/color"
	"image/draw"
)

// ThresholdAlpha returns a copy of img with low-alpha pixels cleared.
// Pixels with alpha below hi become fully transparent; pixels with alpha
// at or above hi are preserved as-is.
func ThresholdAlpha(img *image.NRGBA, lo, hi uint8) *image.NRGBA {
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	draw.Draw(out, bounds, img, bounds.Min, draw.Src)

	if hi < lo {
		hi = lo
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := out.NRGBAAt(x, y)
			if c.A >= hi {
				continue
			}
			out.SetNRGBA(x, y, color.NRGBA{})
		}
	}

	return out
}

// CountFractional returns the number of pixels with alpha strictly between 0 and 255.
func CountFractional(img image.Image) int {
	bounds := img.Bounds()
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := rgba8(img.At(x, y))
			if a > 0 && a < 255 {
				count++
			}
		}
	}
	return count
}
