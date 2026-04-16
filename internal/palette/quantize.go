package palette

import (
	"image"
	"image/color"
	"slices"
)

type histColor struct {
	r, g, b uint8
	weight  int
}

type colorBox struct {
	colors []histColor
	weight int
	minR   uint8
	maxR   uint8
	minG   uint8
	maxG   uint8
	minB   uint8
	maxB   uint8
}

func Extract(img image.Image, maxColors int) []color.NRGBA {
	if img == nil || maxColors <= 0 {
		return nil
	}

	hist := histogram(img)
	if len(hist) == 0 {
		return nil
	}
	if len(hist) <= maxColors {
		return histogramPalette(hist)
	}

	boxes := []colorBox{newColorBox(hist)}
	for len(boxes) < maxColors {
		idx := bestSplitBox(boxes)
		if idx < 0 {
			break
		}
		left, right, ok := splitBox(boxes[idx])
		if !ok {
			break
		}
		boxes[idx] = left
		boxes = append(boxes, right)
	}

	pal := make([]paletteColor, 0, len(boxes))
	seen := map[[3]uint8]struct{}{}
	for _, box := range boxes {
		c := averageBox(box)
		key := [3]uint8{c.R, c.G, c.B}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		pal = append(pal, paletteColor{Color: c, Weight: box.weight})
	}

	slices.SortFunc(pal, comparePaletteColor)
	out := make([]color.NRGBA, len(pal))
	for i, c := range pal {
		out[i] = c.Color
	}
	if len(out) > maxColors {
		out = out[:maxColors]
	}
	return out
}

type paletteColor struct {
	Color  color.NRGBA
	Weight int
}

func histogram(img image.Image) []histColor {
	bounds := img.Bounds()
	counts := map[[3]uint8]int{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := rgba8(img.At(x, y))
			if a == 0 {
				continue
			}
			counts[[3]uint8{r, g, b}]++
		}
	}

	hist := make([]histColor, 0, len(counts))
	for rgb, weight := range counts {
		hist = append(hist, histColor{r: rgb[0], g: rgb[1], b: rgb[2], weight: weight})
	}
	slices.SortFunc(hist, compareHistColor)
	return hist
}

func histogramPalette(hist []histColor) []color.NRGBA {
	pal := make([]color.NRGBA, len(hist))
	for i, c := range hist {
		pal[i] = color.NRGBA{R: c.r, G: c.g, B: c.b, A: 255}
	}
	return pal
}

func bestSplitBox(boxes []colorBox) int {
	best := -1
	for i, box := range boxes {
		if len(box.colors) < 2 {
			continue
		}
		if best == -1 || compareBox(box, boxes[best]) < 0 {
			best = i
		}
	}
	return best
}

func splitBox(box colorBox) (colorBox, colorBox, bool) {
	channel := splitChannel(box)
	colors := append([]histColor(nil), box.colors...)
	slices.SortFunc(colors, func(a, b histColor) int {
		return compareByChannel(a, b, channel)
	})

	target := box.weight / 2
	accum := 0
	splitAt := -1
	for i := 0; i < len(colors)-1; i++ {
		accum += colors[i].weight
		if accum >= target {
			splitAt = i + 1
			break
		}
	}
	if splitAt <= 0 || splitAt >= len(colors) {
		return colorBox{}, colorBox{}, false
	}
	return newColorBox(colors[:splitAt]), newColorBox(colors[splitAt:]), true
}

func newColorBox(colors []histColor) colorBox {
	box := colorBox{colors: append([]histColor(nil), colors...), minR: 255, minG: 255, minB: 255}
	for _, c := range colors {
		box.weight += c.weight
		if c.r < box.minR {
			box.minR = c.r
		}
		if c.r > box.maxR {
			box.maxR = c.r
		}
		if c.g < box.minG {
			box.minG = c.g
		}
		if c.g > box.maxG {
			box.maxG = c.g
		}
		if c.b < box.minB {
			box.minB = c.b
		}
		if c.b > box.maxB {
			box.maxB = c.b
		}
	}
	return box
}

func averageBox(box colorBox) color.NRGBA {
	var sumR, sumG, sumB, total int
	for _, c := range box.colors {
		sumR += int(c.r) * c.weight
		sumG += int(c.g) * c.weight
		sumB += int(c.b) * c.weight
		total += c.weight
	}
	if total == 0 {
		return color.NRGBA{A: 255}
	}
	return color.NRGBA{
		R: uint8((sumR + total/2) / total),
		G: uint8((sumG + total/2) / total),
		B: uint8((sumB + total/2) / total),
		A: 255,
	}
}

func compareHistColor(a, b histColor) int {
	if a.weight != b.weight {
		return b.weight - a.weight
	}
	if a.r != b.r {
		return int(a.r) - int(b.r)
	}
	if a.g != b.g {
		return int(a.g) - int(b.g)
	}
	return int(a.b) - int(b.b)
}

func comparePaletteColor(a, b paletteColor) int {
	if a.Weight != b.Weight {
		return b.Weight - a.Weight
	}
	if a.Color.R != b.Color.R {
		return int(a.Color.R) - int(b.Color.R)
	}
	if a.Color.G != b.Color.G {
		return int(a.Color.G) - int(b.Color.G)
	}
	return int(a.Color.B) - int(b.Color.B)
}

func compareBox(a, b colorBox) int {
	if volume(a) != volume(b) {
		return volume(b) - volume(a)
	}
	if a.weight != b.weight {
		return b.weight - a.weight
	}
	if a.minR != b.minR {
		return int(a.minR) - int(b.minR)
	}
	if a.minG != b.minG {
		return int(a.minG) - int(b.minG)
	}
	return int(a.minB) - int(b.minB)
}

func volume(box colorBox) int {
	return maxRange(box)
}

func maxRange(box colorBox) int {
	rRange := int(box.maxR) - int(box.minR)
	gRange := int(box.maxG) - int(box.minG)
	bRange := int(box.maxB) - int(box.minB)
	if rRange >= gRange && rRange >= bRange {
		return rRange
	}
	if gRange >= bRange {
		return gRange
	}
	return bRange
}

func splitChannel(box colorBox) byte {
	rRange := int(box.maxR) - int(box.minR)
	gRange := int(box.maxG) - int(box.minG)
	bRange := int(box.maxB) - int(box.minB)
	if rRange >= gRange && rRange >= bRange {
		return 'r'
	}
	if gRange >= bRange {
		return 'g'
	}
	return 'b'
}

func compareByChannel(a, b histColor, channel byte) int {
	primaryA, primaryB := component(a, channel), component(b, channel)
	if primaryA != primaryB {
		return int(primaryA) - int(primaryB)
	}
	if a.r != b.r {
		return int(a.r) - int(b.r)
	}
	if a.g != b.g {
		return int(a.g) - int(b.g)
	}
	if a.b != b.b {
		return int(a.b) - int(b.b)
	}
	return b.weight - a.weight
}

func component(c histColor, channel byte) uint8 {
	switch channel {
	case 'r':
		return c.r
	case 'g':
		return c.g
	default:
		return c.b
	}
}

func rgba8(c color.Color) (uint8, uint8, uint8, uint8) {
	r, g, b, a := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)
}
