package sheet

import (
	"fmt"
	"image"
	"path/filepath"

	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

type Frame struct {
	Index int
	Path  string
	Rect  image.Rectangle
	Image *image.NRGBA
}

type Result struct {
	Cols   int
	Rows   int
	CellW  int
	CellH  int
	Frames []Frame

	Detected *pixel.Grid
	Manifest *manifest.Manifest
}

func SliceGrid(img image.Image, source string, cols, rows int, trim bool) (*Result, error) {
	if cols <= 0 {
		return nil, fmt.Errorf("--cols must be greater than 0")
	}
	if rows <= 0 {
		return nil, fmt.Errorf("--rows must be greater than 0")
	}

	bounds := img.Bounds()
	if bounds.Dx()%cols != 0 {
		return nil, fmt.Errorf("image width %d is not divisible by %d (remainder %d)", bounds.Dx(), cols, bounds.Dx()%cols)
	}
	if bounds.Dy()%rows != 0 {
		return nil, fmt.Errorf("image height %d is not divisible by %d (remainder %d)", bounds.Dy(), rows, bounds.Dy()%rows)
	}

	cellW := bounds.Dx() / cols
	cellH := bounds.Dy() / rows
	frames := make([]Frame, 0, cols*rows)
	manifestFrames := make([]manifest.Frame, 0, cols*rows)

	index := 0
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			rect := image.Rect(bounds.Min.X+col*cellW, bounds.Min.Y+row*cellH, bounds.Min.X+(col+1)*cellW, bounds.Min.Y+(row+1)*cellH)
			cropped, err := pixel.Crop(img, rect)
			if err != nil {
				return nil, err
			}

			outImg := cropped
			outRect := rect
			if trim {
				trimmedRect, trimmed, err := trimFrame(cropped, rect)
				if err != nil {
					return nil, err
				}
				outImg = trimmed
				outRect = trimmedRect
			}

			path := fmt.Sprintf("frame_%03d.png", index)
			frames = append(frames, Frame{Index: index, Path: path, Rect: outRect, Image: outImg})
			manifestFrames = append(manifestFrames, manifest.Frame{
				Index: index,
				Path:  path,
				Rect:  manifest.Rect{X: outRect.Min.X, Y: outRect.Min.Y, W: outRect.Dx(), H: outRect.Dy()},
				W:     outImg.Bounds().Dx(),
				H:     outImg.Bounds().Dy(),
			})
			index++
		}
	}

	return &Result{
		Cols:   cols,
		Rows:   rows,
		CellW:  cellW,
		CellH:  cellH,
		Frames: frames,
		Manifest: &manifest.Manifest{
			Source: source,
			CellW:  cellW,
			CellH:  cellH,
			Cols:   cols,
			Rows:   rows,
			Frames: manifestFrames,
		},
	}, nil
}

func SliceAuto(img image.Image, source string, minGap int) (*Result, error) {
	if minGap <= 0 {
		return nil, fmt.Errorf("--min-gap must be greater than 0")
	}

	detected := pixel.GuessGridWithMinGap(img, minGap)
	if detected.Confidence < 0.5 {
		return nil, fmt.Errorf("could not detect a reliable grid (confidence %.2f); try slice grid with explicit --cols/--rows", detected.Confidence)
	}

	result, err := SliceGrid(img, source, detected.Cols, detected.Rows, false)
	if err != nil {
		return nil, err
	}
	result.Detected = &detected
	return result, nil
}

func Write(dir string, result *Result) error {
	if result == nil {
		return fmt.Errorf("write slice output %q: result is nil", dir)
	}

	for _, frame := range result.Frames {
		if err := pixel.SavePNG(filepath.Join(dir, frame.Path), frame.Image); err != nil {
			return err
		}
	}
	if err := manifest.Write(filepath.Join(dir, "manifest.json"), result.Manifest); err != nil {
		return err
	}
	return nil
}

func trimFrame(img *image.NRGBA, sourceRect image.Rectangle) (image.Rectangle, *image.NRGBA, error) {
	bbox := pixel.BBox(img, 0)
	if bbox.Empty() {
		return sourceRect, img, nil
	}

	trimmed, err := pixel.Crop(img, bbox)
	if err != nil {
		return image.Rectangle{}, nil, err
	}
	trimmedSource := image.Rect(sourceRect.Min.X+bbox.Min.X, sourceRect.Min.Y+bbox.Min.Y, sourceRect.Min.X+bbox.Max.X, sourceRect.Min.Y+bbox.Max.Y)
	return trimmedSource, trimmed, nil
}
