package background

import (
	"image"
	"image/color"
)

type edgeNode struct {
	Point image.Point
	Seed  color.NRGBA
}

func removeEdge(img *image.NRGBA, opts Options) (Result, error) {
	out := cloneNRGBA(img)
	bounds := out.Bounds()
	if bounds.Empty() {
		return Result{Image: out}, nil
	}

	seeds := []color.NRGBA{
		out.NRGBAAt(bounds.Min.X, bounds.Min.Y),
		out.NRGBAAt(bounds.Max.X-1, bounds.Min.Y),
		out.NRGBAAt(bounds.Min.X, bounds.Max.Y-1),
		out.NRGBAAt(bounds.Max.X-1, bounds.Max.Y-1),
	}
	visited := make([]bool, bounds.Dx()*bounds.Dy())
	queue := make([]edgeNode, 0, 64)
	push := func(x, y int, seed color.NRGBA) {
		if !image.Pt(x, y).In(bounds) {
			return
		}
		idx := (y-bounds.Min.Y)*bounds.Dx() + (x - bounds.Min.X)
		if visited[idx] {
			return
		}
		if !WithinTolerance(out.NRGBAAt(x, y), seed, opts.Tolerance) {
			return
		}
		visited[idx] = true
		queue = append(queue, edgeNode{Point: image.Pt(x, y), Seed: seed})
	}
	pushIfSeedMatch := func(x, y int) {
		if seed, ok := matchingSeed(out.NRGBAAt(x, y), seeds, opts.Tolerance); ok {
			push(x, y, seed)
		}
	}

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		pushIfSeedMatch(x, bounds.Min.Y)
		pushIfSeedMatch(x, bounds.Max.Y-1)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		pushIfSeedMatch(bounds.Min.X, y)
		pushIfSeedMatch(bounds.Max.X-1, y)
	}

	result := Result{Image: out}
	for head := 0; head < len(queue); head++ {
		node := queue[head]
		current := out.NRGBAAt(node.Point.X, node.Point.Y)
		if current != (color.NRGBA{}) {
			out.SetNRGBA(node.Point.X, node.Point.Y, color.NRGBA{})
			result.RemovedPixels++
			result.ChangedPixels++
		}
		for _, next := range neighbors(node.Point, opts.Connectivity) {
			push(next.X, next.Y, node.Seed)
		}
	}

	return result, nil
}

func matchingSeed(c color.NRGBA, seeds []color.NRGBA, tolerance uint8) (color.NRGBA, bool) {
	for _, seed := range seeds {
		if WithinTolerance(c, seed, tolerance) {
			return seed, true
		}
	}
	return color.NRGBA{}, false
}

func neighbors(p image.Point, connectivity int) []image.Point {
	neighbors := []image.Point{
		p.Add(image.Pt(-1, 0)),
		p.Add(image.Pt(1, 0)),
		p.Add(image.Pt(0, -1)),
		p.Add(image.Pt(0, 1)),
	}
	if connectivity == 8 {
		neighbors = append(neighbors,
			p.Add(image.Pt(-1, -1)),
			p.Add(image.Pt(1, -1)),
			p.Add(image.Pt(-1, 1)),
			p.Add(image.Pt(1, 1)),
		)
	}
	return neighbors
}
