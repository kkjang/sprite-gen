package detail

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"github.com/kkjang/sprite-gen/internal/pixel"
)

func TestNormalizeFactorOneLeavesImageUnchanged(t *testing.T) {
	img := solidImage(8, 6, image.Rect(1, 1, 7, 5), color.NRGBA{R: 255, A: 255})
	result, err := Normalize(img, Options{Factor: 1, AlphaThreshold: 8})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if !result.Unchanged {
		t.Fatal("Unchanged = false, want true")
	}
	if result.OutputW != 8 || result.OutputH != 6 {
		t.Fatalf("output = %dx%d, want 8x6", result.OutputW, result.OutputH)
	}
	if result.InputBBoxH != 4 || result.OutputBBoxH != 4 {
		t.Fatalf("bbox heights = %d/%d, want 4/4", result.InputBBoxH, result.OutputBBoxH)
	}
}

func TestNormalizeExplicitFactorHalvesDimensions(t *testing.T) {
	img := solidImage(12, 8, image.Rect(2, 0, 10, 8), color.NRGBA{G: 255, A: 255})
	result, err := Normalize(img, Options{Factor: 2, AlphaThreshold: 8})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if result.Factor != 2 {
		t.Fatalf("Factor = %d, want 2", result.Factor)
	}
	if result.OutputW != 6 || result.OutputH != 4 {
		t.Fatalf("output = %dx%d, want 6x4", result.OutputW, result.OutputH)
	}
	if result.OutputBBoxH != 4 {
		t.Fatalf("OutputBBoxH = %d, want 4", result.OutputBBoxH)
	}
}

func TestNormalizeTargetHeightChoosesClosestValidFactor(t *testing.T) {
	img := solidImage(12, 12, imgBounds(12, 12), color.NRGBA{B: 255, A: 255})
	result, err := Normalize(img, Options{TargetHeight: 5, AlphaThreshold: 8})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if result.Factor != 2 {
		t.Fatalf("Factor = %d, want 2", result.Factor)
	}
	if result.OutputBBoxH != 6 {
		t.Fatalf("OutputBBoxH = %d, want 6", result.OutputBBoxH)
	}
}

func TestNormalizeAlphaThresholdIgnoresLowAlphaHaze(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 6, 6))
	fillRect(img, image.Rect(2, 2, 4, 5), color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(2, 1, color.NRGBA{R: 255, A: 7})

	result, err := Normalize(img, Options{Factor: 1, AlphaThreshold: 8})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if result.InputBBoxH != 3 {
		t.Fatalf("InputBBoxH = %d, want 3", result.InputBBoxH)
	}

	result, err = Normalize(img, Options{Factor: 1, AlphaThreshold: 1})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if result.InputBBoxH != 4 {
		t.Fatalf("InputBBoxH = %d, want 4", result.InputBBoxH)
	}
}

func TestNormalizeRejectsInvalidOptionCombinations(t *testing.T) {
	img := solidImage(4, 4, imgBounds(4, 4), color.NRGBA{A: 255})
	if _, err := Normalize(img, Options{}); err == nil {
		t.Fatal("Normalize() error = nil, want missing option error")
	}
	if _, err := Normalize(img, Options{TargetHeight: 4, Factor: 2}); err == nil {
		t.Fatal("Normalize() error = nil, want both-options error")
	}
	if _, err := Normalize(img, Options{Factor: -1}); err == nil {
		t.Fatal("Normalize() error = nil, want invalid factor error")
	}
}

func TestNormalizeRejectsFullyTransparentImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	if _, err := Normalize(img, Options{Factor: 1, AlphaThreshold: 8}); err == nil {
		t.Fatal("Normalize() error = nil, want transparent image error")
	}
}

func TestNormalizeGoldenLanternWalkTargetHeight(t *testing.T) {
	ensureNormalizeFixtures(t)
	inputPath := filepath.Join(repoRoot(t), "testdata", "input", "normalize", "lantern_walk.png")
	goldenPath := filepath.Join(repoRoot(t), "testdata", "golden", "normalize", "lantern_walk_h48.png")

	img, err := pixel.LoadPNG(inputPath)
	if err != nil {
		t.Fatalf("LoadPNG(%q) error = %v", inputPath, err)
	}
	got, err := Normalize(img, Options{TargetHeight: 48, AlphaThreshold: 8})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if got.Factor != 2 {
		t.Fatalf("Factor = %d, want 2", got.Factor)
	}
	assertPNGEqualToFile(t, got.Image, goldenPath)
}

func TestNormalizeGoldenKnightTargetHeight(t *testing.T) {
	ensureNormalizeFixtures(t)
	inputPath := filepath.Join(repoRoot(t), "testdata", "input", "normalize", "knight_native.png")
	goldenPath := filepath.Join(repoRoot(t), "testdata", "golden", "normalize", "knight_native_h48.png")

	img, err := pixel.LoadPNG(inputPath)
	if err != nil {
		t.Fatalf("LoadPNG(%q) error = %v", inputPath, err)
	}
	got, err := Normalize(img, Options{TargetHeight: 48, AlphaThreshold: 8})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if got.Factor != 2 {
		t.Fatalf("Factor = %d, want 2", got.Factor)
	}
	assertPNGEqualToFile(t, got.Image, goldenPath)
}

func solidImage(w, h int, rect image.Rectangle, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fillRect(img, rect, c)
	return img
}

func imgBounds(w, h int) image.Rectangle {
	return image.Rect(0, 0, w, h)
}
