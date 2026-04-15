package pixel

import "image"

// BBox returns the tight rectangle containing all pixels with alpha > alphaMin.
func BBox(img image.Image, alphaMin uint8) image.Rectangle {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	found := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a16 := img.At(x, y).RGBA()
			if uint8(a16>>8) <= alphaMin {
				continue
			}
			if !found {
				minX, minY = x, y
				maxX, maxY = x+1, y+1
				found = true
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+1 > maxX {
				maxX = x + 1
			}
			if y+1 > maxY {
				maxY = y + 1
			}
		}
	}

	if !found {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX, maxY)
}
