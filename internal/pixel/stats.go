package pixel

import "image"

const MaxUniqueColors = 4096

type Stats struct {
	W             int     `json:"w"`
	H             int     `json:"h"`
	UniqueColors  int     `json:"unique_colors"`
	OpaquePixels  int     `json:"opaque"`
	TransparentPx int     `json:"transparent"`
	FractionalPx  int     `json:"fractional"`
	AAScore       float64 `json:"aa_score"`
}

func ComputeStats(img image.Image) Stats {
	bounds := img.Bounds()
	stats := Stats{W: bounds.Dx(), H: bounds.Dy()}
	seen := make(map[[4]uint8]struct{}, min(bounds.Dx()*bounds.Dy(), MaxUniqueColors))

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := rgba8(img.At(x, y))
			if len(seen) < MaxUniqueColors {
				seen[[4]uint8{r, g, b, a}] = struct{}{}
			}

			switch a {
			case 0:
				stats.TransparentPx++
			case 255:
				stats.OpaquePixels++
			default:
				stats.FractionalPx++
			}
		}
	}

	stats.UniqueColors = len(seen)
	denom := stats.OpaquePixels + stats.FractionalPx
	if denom > 0 {
		stats.AAScore = float64(stats.FractionalPx) / float64(denom)
	}
	return stats
}

func rgba8(c colorLike) (uint8, uint8, uint8, uint8) {
	r, g, b, a := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)
}

type colorLike interface {
	RGBA() (r, g, b, a uint32)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
