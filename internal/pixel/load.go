package pixel

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
)

// LoadPNG decodes a PNG file into an *image.NRGBA.
func LoadPNG(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open PNG %q: %w", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode PNG %q: %w", path, err)
	}

	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	draw.Draw(out, bounds, img, bounds.Min, draw.Src)
	return out, nil
}
