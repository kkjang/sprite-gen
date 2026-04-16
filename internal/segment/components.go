package segment

import (
	"image"
	"sort"
)

// Component is one connected foreground region found in a binary alpha mask.
type Component struct {
	ID   int
	BBox image.Rectangle
	Area int
}

// Label runs 4-connected component labeling over a binary alpha mask.
func Label(mask *image.Alpha) ([]uint16, []Component) {
	if mask == nil {
		return nil, nil
	}

	bounds := mask.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return nil, nil
	}

	labels := make([]uint16, w*h)
	components := make([]Component, 0)
	queue := make([]image.Point, 0, 64)
	nextID := uint16(1)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.AlphaAt(x, y).A == 0 {
				continue
			}
			idx := labelIndex(bounds, x, y)
			if labels[idx] != 0 {
				continue
			}

			component := Component{ID: int(nextID), BBox: image.Rect(x, y, x+1, y+1)}
			queue = append(queue[:0], image.Pt(x, y))
			labels[idx] = nextID

			for len(queue) > 0 {
				p := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				component.Area++
				if p.X < component.BBox.Min.X {
					component.BBox.Min.X = p.X
				}
				if p.Y < component.BBox.Min.Y {
					component.BBox.Min.Y = p.Y
				}
				if p.X+1 > component.BBox.Max.X {
					component.BBox.Max.X = p.X + 1
				}
				if p.Y+1 > component.BBox.Max.Y {
					component.BBox.Max.Y = p.Y + 1
				}

				neighbors := [4]image.Point{
					{X: p.X - 1, Y: p.Y},
					{X: p.X + 1, Y: p.Y},
					{X: p.X, Y: p.Y - 1},
					{X: p.X, Y: p.Y + 1},
				}
				for _, neighbor := range neighbors {
					if neighbor.X < bounds.Min.X || neighbor.X >= bounds.Max.X || neighbor.Y < bounds.Min.Y || neighbor.Y >= bounds.Max.Y {
						continue
					}
					if mask.AlphaAt(neighbor.X, neighbor.Y).A == 0 {
						continue
					}
					nidx := labelIndex(bounds, neighbor.X, neighbor.Y)
					if labels[nidx] != 0 {
						continue
					}
					labels[nidx] = nextID
					queue = append(queue, neighbor)
				}
			}

			components = append(components, component)
			nextID++
		}
	}

	return labels, components
}

// Filter returns only components whose area is at least minArea.
func Filter(cs []Component, minArea int) []Component {
	if minArea <= 0 {
		return append([]Component(nil), cs...)
	}
	out := make([]Component, 0, len(cs))
	for _, c := range cs {
		if c.Area >= minArea {
			out = append(out, c)
		}
	}
	return out
}

// SortLTR sorts components left-to-right, then top-to-bottom.
func SortLTR(cs []Component) {
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].BBox.Min.X != cs[j].BBox.Min.X {
			return cs[i].BBox.Min.X < cs[j].BBox.Min.X
		}
		return cs[i].BBox.Min.Y < cs[j].BBox.Min.Y
	})
}

// SortAreaDesc sorts larger components first, with left-to-right tie breaks.
func SortAreaDesc(cs []Component) {
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].Area != cs[j].Area {
			return cs[i].Area > cs[j].Area
		}
		if cs[i].BBox.Min.X != cs[j].BBox.Min.X {
			return cs[i].BBox.Min.X < cs[j].BBox.Min.X
		}
		return cs[i].BBox.Min.Y < cs[j].BBox.Min.Y
	})
}

func labelIndex(bounds image.Rectangle, x, y int) int {
	return (y-bounds.Min.Y)*bounds.Dx() + (x - bounds.Min.X)
}
