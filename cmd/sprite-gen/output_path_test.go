package main

import (
	"path/filepath"
	"testing"
)

func TestOutputSubjectFromSourcePath(t *testing.T) {
	if got := outputSubject(filepath.Join(".cache", "slime3.png")); got != "slime3" {
		t.Fatalf("outputSubject() = %q, want %q", got, "slime3")
	}
}

func TestOutputSubjectFromLegacyOutPath(t *testing.T) {
	inPath := filepath.Join("out", "snap", "slime3", "native.png")
	if got := outputSubject(inPath); got != "slime3" {
		t.Fatalf("outputSubject(%q) = %q, want %q", inPath, got, "slime3")
	}
}

func TestOutputSubjectFromSubjectFirstOutPath(t *testing.T) {
	inPath := filepath.Join("out", "slime3", "snap", "native.png")
	if got := outputSubject(inPath); got != "slime3" {
		t.Fatalf("outputSubject(%q) = %q, want %q", inPath, got, "slime3")
	}
}

func TestDefaultPaletteExtractOutPath(t *testing.T) {
	inPath := filepath.Join("out", "slime3", "snap", "native.png")
	want := filepath.Join("out", "slime3", "palette", "extracted-16.hex")
	if got := defaultPaletteExtractOutPath(inPath, "hex", 16); got != want {
		t.Fatalf("defaultPaletteExtractOutPath() = %q, want %q", got, want)
	}
}

func TestDefaultPrepAlphaOutPath(t *testing.T) {
	inPath := filepath.Join("out", "slime3", "snap", "native.png")
	want := filepath.Join("out", "slime3", "prep", "clean.png")
	if got := defaultPrepAlphaOutPath(inPath); got != want {
		t.Fatalf("defaultPrepAlphaOutPath() = %q, want %q", got, want)
	}
}

func TestDefaultPrepBackgroundOutPath(t *testing.T) {
	inPath := filepath.Join("out", "slime3", "snap", "native.png")
	want := filepath.Join("out", "slime3", "prep", "background.png")
	if got := defaultPrepBackgroundOutPath(inPath); got != want {
		t.Fatalf("defaultPrepBackgroundOutPath() = %q, want %q", got, want)
	}
}

func TestDefaultAlignOutDir(t *testing.T) {
	inPath := filepath.Join("out", "slime3", "slice")
	want := filepath.Join("out", "slime3", "align")
	if got := defaultAlignOutDir(inPath); got != want {
		t.Fatalf("defaultAlignOutDir() = %q, want %q", got, want)
	}
}

func TestDefaultDiffOutPath(t *testing.T) {
	aPath := filepath.Join("out", "slime3", "align", "frame_000.png")
	bPath := filepath.Join("out", "slime3", "align", "frame_001.png")
	want := filepath.Join("out", "slime3_vs_slime3", "diff", "diff.png")
	if got := defaultDiffOutPath(aPath, bPath); got != want {
		t.Fatalf("defaultDiffOutPath() = %q, want %q", got, want)
	}
}

func TestDefaultExportOutPathGIF(t *testing.T) {
	inPath := filepath.Join("out", "slime3", "align")
	want := filepath.Join("out", "slime3", "export", "slime3_preview.gif")
	if got := defaultExportOutPath(inPath, "gif"); got != want {
		t.Fatalf("defaultExportOutPath() = %q, want %q", got, want)
	}
}

func TestDefaultExportOutPathSheetPNG(t *testing.T) {
	inPath := filepath.Join("out", "slime3", "segment")
	want := filepath.Join("out", "slime3", "export", "slime3_sheet.png")
	if got := defaultExportOutPath(inPath, "sheet-png"); got != want {
		t.Fatalf("defaultExportOutPath() = %q, want %q", got, want)
	}
}
