package palette

import (
	"bytes"
	"image/color"
	"strings"
	"testing"
)

func TestHexRoundTrip(t *testing.T) {
	pal := []color.NRGBA{{R: 0x12, G: 0x34, B: 0x56, A: 255}, {R: 0xaa, G: 0xbb, B: 0xcc, A: 255}}
	var buf bytes.Buffer
	if err := WriteHex(&buf, pal); err != nil {
		t.Fatalf("WriteHex() error = %v", err)
	}
	got, err := ReadHex(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("ReadHex() error = %v", err)
	}
	if len(got) != len(pal) {
		t.Fatalf("len(ReadHex()) = %d, want %d", len(got), len(pal))
	}
	for i := range pal {
		if got[i] != pal[i] {
			t.Fatalf("color %d = %#v, want %#v", i, got[i], pal[i])
		}
	}
}

func TestGPLRoundTrip(t *testing.T) {
	pal := []color.NRGBA{{R: 1, G: 2, B: 3, A: 255}, {R: 20, G: 30, B: 40, A: 255}}
	var buf bytes.Buffer
	if err := WriteGPL(&buf, "test", pal); err != nil {
		t.Fatalf("WriteGPL() error = %v", err)
	}
	got, err := ReadGPL(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("ReadGPL() error = %v", err)
	}
	if len(got) != len(pal) {
		t.Fatalf("len(ReadGPL()) = %d, want %d", len(got), len(pal))
	}
	for i := range pal {
		if got[i] != pal[i] {
			t.Fatalf("color %d = %#v, want %#v", i, got[i], pal[i])
		}
	}
}

func TestReadHexMalformed(t *testing.T) {
	_, err := ReadHex(strings.NewReader("#zzzzzz\n"))
	if err == nil || !strings.Contains(err.Error(), "invalid hex color") {
		t.Fatalf("ReadHex() error = %v, want invalid hex color", err)
	}
}
