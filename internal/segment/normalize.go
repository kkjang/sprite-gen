package segment

import (
	"fmt"
	"image"

	"github.com/kkjang/sprite-gen/internal/pixel"
)

type Anchor string

const (
	AnchorFeet   Anchor = "feet"
	AnchorCenter Anchor = "center"
	AnchorTop    Anchor = "top"
)

type Fit string

const (
	FitError     Fit = "error"
	FitDownscale Fit = "scale"
	FitCrop      Fit = "crop"
)

// NormalizeToCell places the requested source rectangle into a fixed-size cell.
func NormalizeToCell(src *image.NRGBA, srcRect image.Rectangle, cell image.Point, anchor Anchor, fit Fit) (*image.NRGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("normalize subject: source image is nil")
	}
	if cell.X <= 0 || cell.Y <= 0 {
		return nil, fmt.Errorf("normalize subject: cell must be greater than 0x0")
	}
	if srcRect.Empty() {
		return nil, fmt.Errorf("normalize subject: source rectangle is empty")
	}
	if !srcRect.In(src.Bounds()) {
		return nil, fmt.Errorf("normalize subject: source rectangle %v is outside image bounds %v", srcRect, src.Bounds())
	}

	cropped, err := pixel.Crop(src, srcRect)
	if err != nil {
		return nil, err
	}

	if cropped.Bounds().Dx() > cell.X || cropped.Bounds().Dy() > cell.Y {
		switch fit {
		case FitError:
			return nil, fmt.Errorf("subject at src_rect=%v exceeds cell %dx%d; set --fit scale, --fit crop, or --cell WxH", srcRect, cell.X, cell.Y)
		case FitDownscale:
			factor := fitDownscaleFactor(cropped.Bounds().Dx(), cropped.Bounds().Dy(), cell.X, cell.Y)
			cropped = pixel.Downscale(cropped, factor)
		case FitCrop:
			// Keep the original crop and let PlaceInCell clip after anchoring.
		default:
			return nil, fmt.Errorf("invalid fit %q", fit)
		}
	}

	offset, err := anchorOffset(cropped.Bounds().Size(), cell, anchor)
	if err != nil {
		return nil, err
	}
	return pixel.PlaceInCell(cropped, cropped.Bounds(), cell, offset), nil
}

// AutoCell returns a rounded-up cell that fits every component bbox.
func AutoCell(cs []Component, round int) image.Point {
	if round <= 0 {
		round = 8
	}
	maxW, maxH := 0, 0
	for _, c := range cs {
		if c.BBox.Dx() > maxW {
			maxW = c.BBox.Dx()
		}
		if c.BBox.Dy() > maxH {
			maxH = c.BBox.Dy()
		}
	}
	return image.Pt(roundUp(maxW, round), roundUp(maxH, round))
}

func fitDownscaleFactor(w, h, cellW, cellH int) int {
	factor := 1
	for ceilDiv(w, factor) > cellW || ceilDiv(h, factor) > cellH {
		factor++
	}
	return factor
}

func anchorOffset(size image.Point, cell image.Point, anchor Anchor) (image.Point, error) {
	x := (cell.X - size.X) / 2
	switch anchor {
	case AnchorFeet:
		return image.Pt(x, cell.Y-size.Y), nil
	case AnchorCenter:
		return image.Pt(x, (cell.Y-size.Y)/2), nil
	case AnchorTop:
		return image.Pt(x, 0), nil
	default:
		return image.Point{}, fmt.Errorf("invalid anchor %q", anchor)
	}
}

func roundUp(n, multiple int) int {
	if n <= 0 {
		return 0
	}
	if multiple <= 1 {
		return n
	}
	rem := n % multiple
	if rem == 0 {
		return n
	}
	return n + multiple - rem
}

func ceilDiv(n, d int) int {
	return (n + d - 1) / d
}
