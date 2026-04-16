package palette

import (
	"image"
	"image/color"
	"image/draw"
)

func Snap(c color.NRGBA, pal []color.NRGBA) color.NRGBA {
	if len(pal) == 0 {
		return c
	}
	best := pal[0]
	bestDist := rgbDistance(c, best)
	for _, candidate := range pal[1:] {
		dist := rgbDistance(c, candidate)
		if dist < bestDist {
			best = candidate
			bestDist = dist
		}
	}
	best.A = c.A
	return best
}

func Apply(img image.Image, pal []color.NRGBA, dither bool) *image.NRGBA {
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	if len(pal) == 0 {
		draw.Draw(out, bounds, img, bounds.Min, draw.Src)
		return out
	}
	if !dither {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				if c.A == 0 {
					continue
				}
				out.SetNRGBA(x, y, Snap(c, pal))
			}
		}
		return out
	}

	opaque := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			c.A = 255
			opaque.SetNRGBA(x, y, c)
		}
	}

	quantized := image.NewPaletted(bounds, toStdPalette(pal))
	draw.FloydSteinberg.Draw(quantized, bounds, opaque, bounds.Min)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			orig := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if orig.A == 0 {
				continue
			}
			snapped := color.NRGBAModel.Convert(quantized.At(x, y)).(color.NRGBA)
			snapped.A = orig.A
			out.SetNRGBA(x, y, snapped)
		}
	}
	return out
}

func rgbDistance(a, b color.NRGBA) int {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	return dr*dr + dg*dg + db*db
}

func toStdPalette(pal []color.NRGBA) color.Palette {
	std := make(color.Palette, len(pal))
	for i, c := range pal {
		c.A = 255
		std[i] = c
	}
	return std
}
