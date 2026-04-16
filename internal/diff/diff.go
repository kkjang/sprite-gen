package diff

import (
	"image"
	"image/color"
)

type Result struct {
	DiffPixels  int
	TotalPixels int
	Percent     float64
	BBox        image.Rectangle
}

func Compare(a, b image.Image, tolerance uint8) Result {
	bounds := comparisonBounds(a, b)
	result := Result{TotalPixels: bounds.Dx() * bounds.Dy()}
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if !pixelsDiffer(a, b, x, y, tolerance) {
				continue
			}
			result.DiffPixels++
			result.BBox = expandBBox(result.BBox, x, y)
		}
	}
	if result.TotalPixels > 0 {
		result.Percent = float64(result.DiffPixels) * 100 / float64(result.TotalPixels)
	}
	return result
}

func DiffImage(a, b image.Image, tolerance uint8) *image.NRGBA {
	bounds := comparisonBounds(a, b)
	out := image.NewNRGBA(bounds)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if pixelsDiffer(a, b, x, y, tolerance) {
				out.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
				continue
			}
			out.SetNRGBA(x, y, color.NRGBA{R: 128, G: 128, B: 128, A: 64})
		}
	}
	return out
}

func comparisonBounds(a, b image.Image) image.Rectangle {
	aw, ah := imageSize(a)
	bw, bh := imageSize(b)
	return image.Rect(0, 0, maxInt(aw, bw), maxInt(ah, bh))
}

func imageSize(img image.Image) (int, int) {
	if img == nil {
		return 0, 0
	}
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy()
}

func pixelsDiffer(a, b image.Image, x, y int, tolerance uint8) bool {
	ca := samplePixel(a, x, y)
	cb := samplePixel(b, x, y)
	return channelDiff(ca.R, cb.R) > tolerance ||
		channelDiff(ca.G, cb.G) > tolerance ||
		channelDiff(ca.B, cb.B) > tolerance ||
		channelDiff(ca.A, cb.A) > tolerance
}

func samplePixel(img image.Image, x, y int) color.NRGBA {
	if img == nil {
		return color.NRGBA{}
	}
	bounds := img.Bounds()
	px := bounds.Min.X + x
	py := bounds.Min.Y + y
	if px < bounds.Min.X || px >= bounds.Max.X || py < bounds.Min.Y || py >= bounds.Max.Y {
		return color.NRGBA{}
	}
	return color.NRGBAModel.Convert(img.At(px, py)).(color.NRGBA)
}

func channelDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func expandBBox(rect image.Rectangle, x, y int) image.Rectangle {
	point := image.Rect(x, y, x+1, y+1)
	if rect.Empty() {
		return point
	}
	if x < rect.Min.X {
		rect.Min.X = x
	}
	if y < rect.Min.Y {
		rect.Min.Y = y
	}
	if x+1 > rect.Max.X {
		rect.Max.X = x + 1
	}
	if y+1 > rect.Max.Y {
		rect.Max.Y = y + 1
	}
	return rect
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
