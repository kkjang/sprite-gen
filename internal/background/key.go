package background

import (
	"fmt"
	"image"
	"image/color"
)

func removeKey(img *image.NRGBA, opts Options) (Result, error) {
	if !opts.HasKeyColor {
		return Result{}, fmt.Errorf("missing required --color for prep background --method key")
	}
	out := cloneNRGBA(img)
	result := Result{Image: out}
	for y := out.Bounds().Min.Y; y < out.Bounds().Max.Y; y++ {
		for x := out.Bounds().Min.X; x < out.Bounds().Max.X; x++ {
			pixel := out.NRGBAAt(x, y)
			if pixel.A == 0 || !withinToleranceRGB(pixel, opts.KeyColor, opts.Tolerance) {
				continue
			}
			out.SetNRGBA(x, y, color.NRGBA{})
			result.RemovedPixels++
			result.ChangedPixels++
		}
	}
	return result, nil
}
