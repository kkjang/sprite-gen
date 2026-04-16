package background

import (
	"image"
	"image/color"
	"testing"
)

func TestRemoveKeyExactBackground(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 6, 6))
	fill(img, img.Bounds(), color.NRGBA{R: 255, B: 255, A: 255})
	fill(img, image.Rect(2, 2, 4, 4), color.NRGBA{G: 255, A: 255})

	result, err := Remove(img, Options{Method: MethodKey, KeyColor: color.NRGBA{R: 255, B: 255, A: 255}, HasKeyColor: true, Tolerance: 0, Connectivity: 4})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if result.Method != MethodKey {
		t.Fatalf("Method = %q, want %q", result.Method, MethodKey)
	}
	if result.Image.NRGBAAt(0, 0).A != 0 {
		t.Fatalf("background alpha = %d, want 0", result.Image.NRGBAAt(0, 0).A)
	}
	if result.Image.NRGBAAt(2, 2).A != 255 {
		t.Fatalf("subject alpha = %d, want 255", result.Image.NRGBAAt(2, 2).A)
	}
	if result.RemovedPixels != 32 {
		t.Fatalf("RemovedPixels = %d, want 32", result.RemovedPixels)
	}
}

func TestRemoveKeyTolerance(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 250, G: 5, B: 250, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 200, A: 255})

	result, err := Remove(img, Options{Method: MethodKey, KeyColor: color.NRGBA{R: 255, B: 255, A: 255}, HasKeyColor: true, Tolerance: 8, Connectivity: 4})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if result.Image.NRGBAAt(0, 0).A != 0 {
		t.Fatalf("near-key alpha = %d, want 0", result.Image.NRGBAAt(0, 0).A)
	}
	if result.Image.NRGBAAt(1, 0).A != 255 {
		t.Fatalf("non-key alpha = %d, want 255", result.Image.NRGBAAt(1, 0).A)
	}
}

func TestRemoveEdgeConnectedBackground(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	fill(img, img.Bounds(), color.NRGBA{R: 32, G: 32, B: 32, A: 255})
	fill(img, image.Rect(2, 2, 6, 6), color.NRGBA{R: 255, G: 255, A: 255})

	result, err := Remove(img, Options{Method: MethodEdge, Tolerance: 0, Connectivity: 4})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if result.Image.NRGBAAt(0, 0).A != 0 {
		t.Fatalf("edge background alpha = %d, want 0", result.Image.NRGBAAt(0, 0).A)
	}
	if result.Image.NRGBAAt(3, 3).A != 255 {
		t.Fatalf("subject alpha = %d, want 255", result.Image.NRGBAAt(3, 3).A)
	}
}

func TestRemoveEdgePreservesDisconnectedInterior(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 9, 9))
	fill(img, img.Bounds(), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	fill(img, image.Rect(2, 2, 7, 7), color.NRGBA{A: 255})
	fill(img, image.Rect(3, 3, 6, 6), color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	result, err := Remove(img, Options{Method: MethodEdge, Tolerance: 0, Connectivity: 4})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if result.Image.NRGBAAt(0, 0).A != 0 {
		t.Fatalf("outer background alpha = %d, want 0", result.Image.NRGBAAt(0, 0).A)
	}
	if result.Image.NRGBAAt(4, 4).A != 255 {
		t.Fatalf("enclosed region alpha = %d, want 255", result.Image.NRGBAAt(4, 4).A)
	}
	if result.Image.NRGBAAt(2, 2).A != 255 {
		t.Fatalf("barrier alpha = %d, want 255", result.Image.NRGBAAt(2, 2).A)
	}
}

func TestRemoveEdgeConnectivityEightTraversesDiagonal(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	fill(img, img.Bounds(), color.NRGBA{A: 255})
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	img.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	result4, err := Remove(img, Options{Method: MethodEdge, Tolerance: 0, Connectivity: 4})
	if err != nil {
		t.Fatalf("Remove(...,4) error = %v", err)
	}
	if result4.Image.NRGBAAt(1, 1).A != 255 {
		t.Fatalf("4-connectivity alpha = %d, want 255", result4.Image.NRGBAAt(1, 1).A)
	}

	result8, err := Remove(img, Options{Method: MethodEdge, Tolerance: 0, Connectivity: 8})
	if err != nil {
		t.Fatalf("Remove(...,8) error = %v", err)
	}
	if result8.Image.NRGBAAt(1, 1).A != 0 {
		t.Fatalf("8-connectivity alpha = %d, want 0", result8.Image.NRGBAAt(1, 1).A)
	}
}

func TestRemoveAutoResolution(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, B: 255, A: 255})

	resultKey, err := Remove(img, Options{Method: MethodAuto, KeyColor: color.NRGBA{R: 255, B: 255, A: 255}, HasKeyColor: true, Connectivity: 4})
	if err != nil {
		t.Fatalf("Remove(auto+key) error = %v", err)
	}
	if resultKey.Method != MethodKey {
		t.Fatalf("auto with key resolved to %q, want %q", resultKey.Method, MethodKey)
	}

	resultEdge, err := Remove(img, Options{Method: MethodAuto, Connectivity: 4})
	if err != nil {
		t.Fatalf("Remove(auto) error = %v", err)
	}
	if resultEdge.Method != MethodEdge {
		t.Fatalf("auto without key resolved to %q, want %q", resultEdge.Method, MethodEdge)
	}
}

func fill(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
