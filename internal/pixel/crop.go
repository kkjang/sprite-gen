package pixel

import (
	"fmt"
	"image"
	"image/draw"
)

func Crop(img image.Image, rect image.Rectangle) (*image.NRGBA, error) {
	if rect.Empty() {
		return nil, fmt.Errorf("crop rectangle is empty")
	}
	if !rect.In(img.Bounds()) {
		return nil, fmt.Errorf("crop rectangle %v is outside image bounds %v", rect, img.Bounds())
	}

	out := image.NewNRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(out, out.Bounds(), img, rect.Min, draw.Src)
	return out, nil
}
