package pixel

import "image"

var scaleCandidates = []int{8, 4, 3, 2}

const detectScaleUniformThreshold = 0.95

// DetectScale guesses the integer upscale factor of img.
func DetectScale(img image.Image) int {
	bounds := img.Bounds()
	if bounds.Dx() <= 1 || bounds.Dy() <= 1 {
		return 1
	}

	for _, factor := range scaleCandidates {
		if factor > bounds.Dx() || factor > bounds.Dy() {
			continue
		}
		if bestUniformity(img, factor) >= detectScaleUniformThreshold {
			return factor
		}
	}

	return 1
}

func bestUniformity(img image.Image, factor int) float64 {
	best := 0.0
	for offsetY := 0; offsetY < factor; offsetY++ {
		for offsetX := 0; offsetX < factor; offsetX++ {
			ratio := blockUniformity(img, factor, offsetX, offsetY)
			if ratio > best {
				best = ratio
			}
		}
	}
	return best
}

func blockUniformity(img image.Image, factor, offsetX, offsetY int) float64 {
	bounds := img.Bounds()
	blocksX := (bounds.Dx() - offsetX) / factor
	blocksY := (bounds.Dy() - offsetY) / factor
	if blocksX <= 0 || blocksY <= 0 {
		return 0
	}

	uniform := 0
	total := 0
	for by := 0; by < blocksY; by++ {
		for bx := 0; bx < blocksX; bx++ {
			total++
			startX := bounds.Min.X + offsetX + bx*factor
			startY := bounds.Min.Y + offsetY + by*factor
			if isUniformBlock(img, startX, startY, factor) {
				uniform++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(uniform) / float64(total)
}

func isUniformBlock(img image.Image, startX, startY, factor int) bool {
	r0, g0, b0, a0 := rgba8(img.At(startX, startY))
	for y := 0; y < factor; y++ {
		for x := 0; x < factor; x++ {
			r, g, b, a := rgba8(img.At(startX+x, startY+y))
			if r != r0 || g != g0 || b != b0 || a != a0 {
				return false
			}
		}
	}
	return true
}

// Downscale reduces img by an integer factor using nearest-neighbor sampling.
func Downscale(img *image.NRGBA, factor int) *image.NRGBA {
	if factor < 1 {
		panic("pixel: downscale factor must be >= 1")
	}
	if factor == 1 {
		copy := image.NewNRGBA(img.Bounds())
		copy.Pix = append(copy.Pix[:0], img.Pix...)
		return copy
	}

	bounds := img.Bounds()
	outBounds := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+bounds.Dx()/factor, bounds.Min.Y+bounds.Dy()/factor)
	out := image.NewNRGBA(outBounds)
	for y := 0; y < outBounds.Dy(); y++ {
		for x := 0; x < outBounds.Dx(); x++ {
			out.SetNRGBA(outBounds.Min.X+x, outBounds.Min.Y+y, img.NRGBAAt(bounds.Min.X+x*factor, bounds.Min.Y+y*factor))
		}
	}
	return out
}
