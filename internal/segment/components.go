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

type Row struct {
	Components []Component
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

// SortLTR sorts components in row-major reading order: top-to-bottom rows,
// then left-to-right within each row.
func SortLTR(cs []Component) {
	rows := GroupRows(cs)
	index := 0
	for _, row := range rows {
		for _, component := range row.Components {
			cs[index] = component
			index++
		}
	}
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

// GroupRows clusters components by vertical center to preserve sheet rows even
// when the source has a bit of per-frame drift.
func GroupRows(cs []Component) []Row {
	if len(cs) == 0 {
		return nil
	}

	sorted := append([]Component(nil), cs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		ci := centerY(sorted[i].BBox)
		cj := centerY(sorted[j].BBox)
		if ci != cj {
			return ci < cj
		}
		if sorted[i].BBox.Min.Y != sorted[j].BBox.Min.Y {
			return sorted[i].BBox.Min.Y < sorted[j].BBox.Min.Y
		}
		return sorted[i].BBox.Min.X < sorted[j].BBox.Min.X
	})

	tolerance := rowTolerance(sorted)
	type cluster struct {
		components []Component
		centerY    int
	}
	clusters := []cluster{}
	for _, component := range sorted {
		cy := centerY(component.BBox)
		best := -1
		bestDelta := 0
		for i, row := range clusters {
			delta := abs(cy - row.centerY)
			if delta > tolerance {
				continue
			}
			if best == -1 || delta < bestDelta {
				best = i
				bestDelta = delta
			}
		}
		if best == -1 {
			clusters = append(clusters, cluster{components: []Component{component}, centerY: cy})
			continue
		}
		clusters[best].components = append(clusters[best].components, component)
		clusters[best].centerY = averageCenterY(clusters[best].components)
	}

	rows := make([]Row, len(clusters))
	for i, row := range clusters {
		sort.SliceStable(row.components, func(a, b int) bool {
			if row.components[a].BBox.Min.X != row.components[b].BBox.Min.X {
				return row.components[a].BBox.Min.X < row.components[b].BBox.Min.X
			}
			return row.components[a].BBox.Min.Y < row.components[b].BBox.Min.Y
		})
		rows[i] = Row{Components: row.components}
	}
	return rows
}

func rowTolerance(cs []Component) int {
	heights := make([]int, len(cs))
	for i, c := range cs {
		heights[i] = c.BBox.Dy()
	}
	sort.Ints(heights)
	median := heights[len(heights)/2]
	if median < 1 {
		median = 1
	}
	tolerance := median / 2
	if tolerance < 4 {
		return 4
	}
	return tolerance
}

func averageCenterY(cs []Component) int {
	if len(cs) == 0 {
		return 0
	}
	total := 0
	for _, c := range cs {
		total += centerY(c.BBox)
	}
	return total / len(cs)
}

func centerY(rect image.Rectangle) int {
	return rect.Min.Y + rect.Dy()/2
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func labelIndex(bounds image.Rectangle, x, y int) int {
	return (y-bounds.Min.Y)*bounds.Dx() + (x - bounds.Min.X)
}
