package segment

import (
	"image"
	"image/color"
	"testing"
)

func TestLabelBackgroundOnly(t *testing.T) {
	mask := image.NewAlpha(image.Rect(0, 0, 8, 8))
	labels, components := Label(mask)
	if len(labels) != 64 {
		t.Fatalf("len(labels) = %d, want 64", len(labels))
	}
	if len(components) != 0 {
		t.Fatalf("len(components) = %d, want 0", len(components))
	}
}

func TestLabelTwoDisjointSquares(t *testing.T) {
	mask := image.NewAlpha(image.Rect(0, 0, 24, 12))
	fillMask(mask, image.Rect(1, 2, 9, 10))
	fillMask(mask, image.Rect(14, 2, 22, 10))

	_, components := Label(mask)
	if len(components) != 2 {
		t.Fatalf("len(components) = %d, want 2", len(components))
	}
	if components[0].BBox != image.Rect(1, 2, 9, 10) || components[0].Area != 64 {
		t.Fatalf("components[0] = %+v, want bbox=%v area=64", components[0], image.Rect(1, 2, 9, 10))
	}
	if components[1].BBox != image.Rect(14, 2, 22, 10) || components[1].Area != 64 {
		t.Fatalf("components[1] = %+v, want bbox=%v area=64", components[1], image.Rect(14, 2, 22, 10))
	}
}

func TestLabelCShapeIsSingleComponent(t *testing.T) {
	mask := image.NewAlpha(image.Rect(0, 0, 8, 8))
	fillMask(mask, image.Rect(1, 1, 2, 7))
	fillMask(mask, image.Rect(1, 1, 6, 2))
	fillMask(mask, image.Rect(1, 6, 6, 7))

	_, components := Label(mask)
	if len(components) != 1 {
		t.Fatalf("len(components) = %d, want 1", len(components))
	}
}

func TestFilterKeepsEqualOrGreaterAreas(t *testing.T) {
	components := []Component{{ID: 1, Area: 99}, {ID: 2, Area: 100}, {ID: 3, Area: 101}}
	got := Filter(components, 100)
	if len(got) != 2 {
		t.Fatalf("len(Filter()) = %d, want 2", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("Filter() = %+v, want IDs 2 and 3", got)
	}
}

func TestSortLTRUsesRowMajorReadingOrder(t *testing.T) {
	components := []Component{
		{ID: 1, BBox: image.Rect(10, 20, 12, 22)},
		{ID: 2, BBox: image.Rect(5, 30, 7, 32)},
		{ID: 3, BBox: image.Rect(5, 10, 7, 12)},
	}
	SortLTR(components)
	if got := []int{components[0].ID, components[1].ID, components[2].ID}; got[0] != 3 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("SortLTR() IDs = %v, want [3 1 2]", got)
	}
}

func TestGroupRowsClustersVerticalDriftIntoSharedRows(t *testing.T) {
	components := []Component{
		{ID: 1, BBox: image.Rect(0, 0, 8, 12)},
		{ID: 2, BBox: image.Rect(16, 2, 24, 14)},
		{ID: 3, BBox: image.Rect(1, 30, 9, 42)},
		{ID: 4, BBox: image.Rect(15, 28, 23, 40)},
	}
	rows := GroupRows(components)
	if len(rows) != 2 {
		t.Fatalf("len(GroupRows()) = %d, want 2", len(rows))
	}
	if got := []int{rows[0].Components[0].ID, rows[0].Components[1].ID, rows[1].Components[0].ID, rows[1].Components[1].ID}; got[0] != 1 || got[1] != 2 || got[2] != 3 || got[3] != 4 {
		t.Fatalf("GroupRows() IDs = %v, want [1 2 3 4]", got)
	}
}

func fillMask(mask *image.Alpha, rect image.Rectangle) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			mask.SetAlpha(x, y, color.Alpha{A: 255})
		}
	}
}
