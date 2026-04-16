package align

import (
	"fmt"
	"image"
	"image/draw"
	"sort"
	"strings"

	"github.com/kkjang/sprite-gen/internal/pixel"
)

type Anchor string

const (
	AnchorCentroid Anchor = "centroid"
	AnchorBBox     Anchor = "bbox"
	AnchorFeet     Anchor = "feet"
)

type Pivot struct {
	X int
	Y int
}

func ParseAnchor(raw string) (Anchor, error) {
	switch strings.ToLower(raw) {
	case string(AnchorCentroid):
		return AnchorCentroid, nil
	case string(AnchorBBox):
		return AnchorBBox, nil
	case string(AnchorFeet):
		return AnchorFeet, nil
	default:
		return "", fmt.Errorf("invalid --anchor value %q; want centroid, bbox, or feet", raw)
	}
}

func ComputePivot(img image.Image, anchor Anchor) Pivot {
	if img == nil {
		return Pivot{}
	}

	switch anchor {
	case AnchorCentroid:
		if pivot, ok := centroidPivot(img); ok {
			return pivot
		}
	case AnchorBBox:
		return bboxCenterPivot(img)
	case AnchorFeet:
		return feetPivot(img)
	}

	return feetPivot(img)
}

func AlignFrames(imgs []image.Image, pivots []Pivot) ([]*image.NRGBA, Pivot, error) {
	if len(imgs) == 0 {
		return nil, Pivot{}, fmt.Errorf("align frames: no images provided")
	}
	if len(imgs) != len(pivots) {
		return nil, Pivot{}, fmt.Errorf("align frames: got %d images and %d pivots", len(imgs), len(pivots))
	}

	target := Pivot{X: medianComponent(pivots, func(p Pivot) int { return p.X }), Y: medianComponent(pivots, func(p Pivot) int { return p.Y })}

	minX, minY := 0, 0
	maxX, maxY := 0, 0
	for i, img := range imgs {
		if img == nil {
			return nil, Pivot{}, fmt.Errorf("align frames: image %d is nil", i)
		}
		bounds := img.Bounds()
		dx := target.X - pivots[i].X
		dy := target.Y - pivots[i].Y
		if dx < minX {
			minX = dx
		}
		if dy < minY {
			minY = dy
		}
		if dx+bounds.Dx() > maxX {
			maxX = dx + bounds.Dx()
		}
		if dy+bounds.Dy() > maxY {
			maxY = dy + bounds.Dy()
		}
	}

	canvas := image.Rect(0, 0, maxX-minX, maxY-minY)
	aligned := make([]*image.NRGBA, len(imgs))
	adjustedTarget := Pivot{X: target.X - minX, Y: target.Y - minY}
	for i, img := range imgs {
		out := image.NewNRGBA(canvas)
		bounds := img.Bounds()
		dx := target.X - pivots[i].X - minX
		dy := target.Y - pivots[i].Y - minY
		dst := image.Rectangle{Min: image.Pt(dx, dy), Max: image.Pt(dx+bounds.Dx(), dy+bounds.Dy())}
		draw.Draw(out, dst, img, bounds.Min, draw.Src)
		aligned[i] = out
	}

	return aligned, adjustedTarget, nil
}

func centroidPivot(img image.Image) (Pivot, bool) {
	bounds := img.Bounds()
	var sumX, sumY, totalAlpha int64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a16 := img.At(x, y).RGBA()
			alpha := int64(a16 >> 8)
			if alpha == 0 {
				continue
			}
			sumX += int64(x-bounds.Min.X) * alpha
			sumY += int64(y-bounds.Min.Y) * alpha
			totalAlpha += alpha
		}
	}
	if totalAlpha == 0 {
		return Pivot{}, false
	}
	return Pivot{
		X: int((sumX + totalAlpha/2) / totalAlpha),
		Y: int((sumY + totalAlpha/2) / totalAlpha),
	}, true
}

func bboxCenterPivot(img image.Image) Pivot {
	bbox := pixel.BBox(img, 0)
	if bbox.Empty() {
		bbox = localBounds(img)
	}
	return Pivot{X: bbox.Min.X + bbox.Dx()/2, Y: bbox.Min.Y + bbox.Dy()/2}
}

func feetPivot(img image.Image) Pivot {
	bbox := pixel.BBox(img, 0)
	if bbox.Empty() {
		bbox = localBounds(img)
	}
	if bbox.Empty() {
		return Pivot{}
	}
	return Pivot{X: bbox.Min.X + bbox.Dx()/2, Y: bbox.Max.Y - 1}
}

func localBounds(img image.Image) image.Rectangle {
	bounds := img.Bounds()
	return image.Rect(0, 0, bounds.Dx(), bounds.Dy())
}

func medianComponent(pivots []Pivot, pick func(Pivot) int) int {
	values := make([]int, len(pivots))
	for i, pivot := range pivots {
		values[i] = pick(pivot)
	}
	sort.Ints(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}
