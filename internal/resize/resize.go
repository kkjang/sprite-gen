package resize

import (
	"fmt"
	"image"
)

type Direction string

const (
	Up   Direction = "up"
	Down Direction = "down"
)

type Options struct {
	Direction Direction
	Factor    int
}

func Image(img *image.NRGBA, opts Options) (*image.NRGBA, error) {
	if img == nil {
		return nil, fmt.Errorf("resize image: image is nil")
	}
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	switch opts.Direction {
	case Up:
		return upscale(img, opts.Factor), nil
	case Down:
		return downscale(img, opts.Factor)
	default:
		return nil, fmt.Errorf("invalid resize direction %q; want up or down", opts.Direction)
	}
}

func Frames(imgs []*image.NRGBA, opts Options) ([]*image.NRGBA, error) {
	if len(imgs) == 0 {
		return nil, fmt.Errorf("resize frames: no images provided")
	}
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	out := make([]*image.NRGBA, len(imgs))
	for i, img := range imgs {
		if img == nil {
			return nil, fmt.Errorf("resize frames: image %d is nil", i)
		}

		resized, err := Image(img, opts)
		if err != nil {
			return nil, fmt.Errorf("resize frames: frame %d: %w", i, err)
		}
		out[i] = resized
	}
	return out, nil
}

func validateOptions(opts Options) error {
	if opts.Direction != Up && opts.Direction != Down {
		return fmt.Errorf("invalid resize direction %q; want up or down", opts.Direction)
	}
	if opts.Factor < 1 {
		return fmt.Errorf("invalid resize factor %d; want an integer greater than or equal to 1", opts.Factor)
	}
	return nil
}

func upscale(img *image.NRGBA, factor int) *image.NRGBA {
	if factor == 1 {
		return clone(img)
	}

	bounds := img.Bounds()
	out := image.NewNRGBA(image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+bounds.Dx()*factor, bounds.Min.Y+bounds.Dy()*factor))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			c := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			for yy := 0; yy < factor; yy++ {
				for xx := 0; xx < factor; xx++ {
					out.SetNRGBA(bounds.Min.X+x*factor+xx, bounds.Min.Y+y*factor+yy, c)
				}
			}
		}
	}
	return out
}

func downscale(img *image.NRGBA, factor int) (*image.NRGBA, error) {
	if factor == 1 {
		return clone(img), nil
	}

	bounds := img.Bounds()
	if bounds.Dx()%factor != 0 || bounds.Dy()%factor != 0 {
		return nil, fmt.Errorf("resize down factor %d does not evenly divide image size %dx%d", factor, bounds.Dx(), bounds.Dy())
	}

	outBounds := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+bounds.Dx()/factor, bounds.Min.Y+bounds.Dy()/factor)
	out := image.NewNRGBA(outBounds)
	for y := 0; y < outBounds.Dy(); y++ {
		for x := 0; x < outBounds.Dx(); x++ {
			out.SetNRGBA(outBounds.Min.X+x, outBounds.Min.Y+y, img.NRGBAAt(bounds.Min.X+x*factor, bounds.Min.Y+y*factor))
		}
	}
	return out, nil
}

func clone(img *image.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(img.Bounds())
	copy(out.Pix, img.Pix)
	return out
}
