package detail

import (
	"fmt"
	"image"
	"math"

	"github.com/kkjang/sprite-gen/internal/pixel"
)

type Options struct {
	TargetHeight   int
	Factor         int
	AlphaThreshold uint8
}

type Result struct {
	Factor      int
	InputW      int
	InputH      int
	OutputW     int
	OutputH     int
	InputBBoxH  int
	OutputBBoxH int
	Unchanged   bool
	Image       *image.NRGBA
}

// Normalize scales img toward the requested detail target.
// Exactly one of opts.TargetHeight or opts.Factor must be set.
func Normalize(img *image.NRGBA, opts Options) (*Result, error) {
	if img == nil {
		return nil, fmt.Errorf("normalize detail: source image is nil")
	}
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	alphaMin := bboxAlphaMin(opts.AlphaThreshold)
	inputBBox := pixel.BBox(img, alphaMin)
	if inputBBox.Empty() {
		return nil, fmt.Errorf("normalize detail: image has no visible pixels at alpha threshold %d", opts.AlphaThreshold)
	}

	factor, err := resolveFactor(img, inputBBox.Dy(), alphaMin, opts)
	if err != nil {
		return nil, err
	}

	output := pixel.Downscale(img, factor)
	outputBBox := pixel.BBox(output, alphaMin)
	if outputBBox.Empty() {
		return nil, fmt.Errorf("normalize detail: factor %d removes all visible pixels at alpha threshold %d", factor, opts.AlphaThreshold)
	}

	return &Result{
		Factor:      factor,
		InputW:      bounds.Dx(),
		InputH:      bounds.Dy(),
		OutputW:     output.Bounds().Dx(),
		OutputH:     output.Bounds().Dy(),
		InputBBoxH:  inputBBox.Dy(),
		OutputBBoxH: outputBBox.Dy(),
		Unchanged:   factor == 1,
		Image:       output,
	}, nil
}

func validateOptions(opts Options) error {
	if opts.TargetHeight != 0 && opts.Factor != 0 {
		return fmt.Errorf("normalize detail: provide exactly one of target height or factor")
	}
	if opts.TargetHeight == 0 && opts.Factor == 0 {
		return fmt.Errorf("normalize detail: provide exactly one of target height or factor")
	}
	if opts.TargetHeight < 0 {
		return fmt.Errorf("normalize detail: target height must be greater than 0")
	}
	if opts.Factor < 0 {
		return fmt.Errorf("normalize detail: factor must be greater than or equal to 1")
	}
	if opts.TargetHeight == 0 && opts.Factor < 1 {
		return fmt.Errorf("normalize detail: factor must be greater than or equal to 1")
	}
	if opts.Factor == 0 && opts.TargetHeight < 1 {
		return fmt.Errorf("normalize detail: target height must be greater than 0")
	}
	return nil
}

func resolveFactor(img *image.NRGBA, inputBBoxH int, alphaMin uint8, opts Options) (int, error) {
	bounds := img.Bounds()
	if opts.Factor != 0 {
		if err := validateFactor(bounds, opts.Factor); err != nil {
			return 0, err
		}
		return opts.Factor, nil
	}

	bestFactor := 0
	bestDistance := math.MaxInt
	for factor := 1; factor <= min(bounds.Dx(), bounds.Dy()); factor++ {
		if bounds.Dx()%factor != 0 || bounds.Dy()%factor != 0 {
			continue
		}
		candidateBBoxH := inputBBoxH
		if factor > 1 {
			candidateBBox := pixel.BBox(pixel.Downscale(img, factor), alphaMin)
			if candidateBBox.Empty() {
				continue
			}
			candidateBBoxH = candidateBBox.Dy()
		}
		distance := abs(candidateBBoxH - opts.TargetHeight)
		if bestFactor == 0 || distance < bestDistance || (distance == bestDistance && factor < bestFactor) {
			bestFactor = factor
			bestDistance = distance
		}
	}
	if bestFactor == 0 {
		return 0, fmt.Errorf("normalize detail: no valid integer factor evenly divides image size %dx%d", bounds.Dx(), bounds.Dy())
	}
	return bestFactor, nil
}

func validateFactor(bounds image.Rectangle, factor int) error {
	if bounds.Dx()%factor != 0 || bounds.Dy()%factor != 0 {
		return fmt.Errorf("normalize detail: factor %d does not evenly divide image size %dx%d", factor, bounds.Dx(), bounds.Dy())
	}
	return nil
}

func bboxAlphaMin(threshold uint8) uint8 {
	if threshold == 0 {
		return 0
	}
	return threshold - 1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
