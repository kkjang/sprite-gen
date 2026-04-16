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
