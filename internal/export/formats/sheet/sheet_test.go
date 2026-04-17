package sheet

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalexport "github.com/kkjang/sprite-gen/internal/export"
	"github.com/kkjang/sprite-gen/internal/manifest"
	"github.com/kkjang/sprite-gen/internal/pixel"
)

func TestSheetExportWritesPNGAndManifest(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "export")
	ctx := testSheetContext(outDir, false, map[string]string{"cols": "4"}, "hero", []image.Point{{32, 32}, {32, 32}, {32, 32}, {32, 32}}, nil)

	result, err := (Sheet{}).Export(ctx)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	pngPath := filepath.Join(outDir, "hero_sheet.png")
	manifestPath := filepath.Join(outDir, "hero_sheet.json")
	data := result.Data.(map[string]any)
	if data["format"] != "sheet" {
		t.Fatalf("data.format = %v, want sheet", data["format"])
	}
	if data["out"] != outDir || data["png"] != pngPath || data["manifest"] != manifestPath {
		t.Fatalf("result paths = %+v, want out=%q png=%q manifest=%q", data, outDir, pngPath, manifestPath)
	}

	img, err := pixel.LoadPNG(pngPath)
	if err != nil {
		t.Fatalf("pixel.LoadPNG() error = %v", err)
	}
	if got := img.Bounds().Dx(); got != 128 {
		t.Fatalf("sheet width = %d, want 128", got)
	}
	if got := img.Bounds().Dy(); got != 32 {
		t.Fatalf("sheet height = %d, want 32", got)
	}

	gotManifest, err := manifest.Read(manifestPath)
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if gotManifest.Version != manifest.CurrentVersion {
		t.Fatalf("manifest version = %d, want %d", gotManifest.Version, manifest.CurrentVersion)
	}
	if gotManifest.Sheet != "hero_sheet.png" {
		t.Fatalf("manifest sheet = %q, want hero_sheet.png", gotManifest.Sheet)
	}
	if gotManifest.SheetSize == nil || *gotManifest.SheetSize != (manifest.Size{W: 128, H: 32}) {
		t.Fatalf("manifest sheet_size = %+v, want 128x32", gotManifest.SheetSize)
	}
	if len(gotManifest.Frames) != 4 {
		t.Fatalf("len(frames) = %d, want 4", len(gotManifest.Frames))
	}
	for i, frame := range gotManifest.Frames {
		wantPath := frameName(i)
		wantRect := manifest.Rect{X: i * 32, Y: 0, W: 32, H: 32}
		if frame.Path != wantPath {
			t.Fatalf("frame %d path = %q, want %q", i, frame.Path, wantPath)
		}
		if frame.Rect != wantRect {
			t.Fatalf("frame %d rect = %+v, want %+v", i, frame.Rect, wantRect)
		}
		if frame.Rect.X < 0 || frame.Rect.Y < 0 || frame.Rect.X+frame.Rect.W > gotManifest.SheetSize.W || frame.Rect.Y+frame.Rect.H > gotManifest.SheetSize.H {
			t.Fatalf("frame %d rect = %+v, want inside sheet %+v", i, frame.Rect, gotManifest.SheetSize)
		}
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	firstFrame := decoded["frames"].([]any)[0].(map[string]any)
	if firstFrame["x"] != float64(0) || firstFrame["y"] != float64(0) || firstFrame["w"] != float64(32) || firstFrame["h"] != float64(32) {
		t.Fatalf("manifest JSON frame = %+v, want flat x/y/w/h", firstFrame)
	}
	if _, exists := firstFrame["rect"]; exists {
		t.Fatalf("manifest JSON frame = %+v, want rect omitted", firstFrame)
	}
	if strings.Contains(string(raw), "duration_ms") || strings.Contains(string(raw), `"tag"`) {
		t.Fatalf("manifest JSON = %s, want no duration_ms/tag when metadata is absent", raw)
	}
}

func TestSheetExportRoundTripsOptionalFrameMetadata(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "export")
	ctx := testSheetContext(outDir, false, map[string]string{"cols": "2", "padding": "2"}, "hero", []image.Point{{16, 16}, {10, 12}}, []manifest.Frame{
		{Index: 0, Path: frameName(0), DurationMS: intPtr(120), Tag: "idle"},
		{Index: 1, Path: frameName(1), DurationMS: intPtr(80), Tag: "idle"},
	})

	if _, err := (Sheet{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	gotManifest, err := manifest.Read(filepath.Join(outDir, "hero_sheet.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if gotManifest.SheetSize == nil || *gotManifest.SheetSize != (manifest.Size{W: 34, H: 16}) {
		t.Fatalf("manifest sheet_size = %+v, want 34x16", gotManifest.SheetSize)
	}
	if gotManifest.Frames[0].DurationMS == nil || *gotManifest.Frames[0].DurationMS != 120 {
		t.Fatalf("frame 0 duration_ms = %+v, want 120", gotManifest.Frames[0].DurationMS)
	}
	if gotManifest.Frames[0].Tag != "idle" || gotManifest.Frames[1].Tag != "idle" {
		t.Fatalf("frame tags = [%q, %q], want [idle, idle]", gotManifest.Frames[0].Tag, gotManifest.Frames[1].Tag)
	}
	if gotManifest.Frames[1].Rect != (manifest.Rect{X: 18, Y: 0, W: 10, H: 12}) {
		t.Fatalf("frame 1 rect = %+v, want {X:18 Y:0 W:10 H:12}", gotManifest.Frames[1].Rect)
	}

	raw, err := os.ReadFile(filepath.Join(outDir, "hero_sheet.json"))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	frameJSON := decoded["frames"].([]any)[0].(map[string]any)
	if frameJSON["x"] != float64(0) || frameJSON["y"] != float64(0) || frameJSON["w"] != float64(16) || frameJSON["h"] != float64(16) {
		t.Fatalf("raw frame JSON = %+v, want flat x/y/w/h", frameJSON)
	}
	if _, exists := frameJSON["rect"]; exists {
		t.Fatalf("raw frame JSON = %+v, want rect omitted", frameJSON)
	}
	frames := decoded["frames"].([]any)
	first := frames[0].(map[string]any)
	if first["duration_ms"] != float64(120) || first["tag"] != "idle" {
		t.Fatalf("raw frame metadata = %+v, want duration_ms=120 and tag=idle", first)
	}
	if _, exists := first["rect"]; exists {
		t.Fatalf("raw frame metadata = %+v, want rect omitted", first)
	}
}

func TestSheetExportUsesManifestGridPositions(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "export")
	ctx := testSheetContext(outDir, false, nil, "hero", []image.Point{{16, 16}, {16, 16}, {16, 16}, {16, 16}}, []manifest.Frame{
		{Index: 0, Path: frameName(0), Row: intPtr(0), Col: intPtr(0)},
		{Index: 1, Path: frameName(1), Row: intPtr(0), Col: intPtr(1)},
		{Index: 2, Path: frameName(2), Row: intPtr(0), Col: intPtr(2)},
		{Index: 3, Path: frameName(3), Row: intPtr(1), Col: intPtr(0)},
	})

	if _, err := (Sheet{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	gotManifest, err := manifest.Read(filepath.Join(outDir, "hero_sheet.json"))
	if err != nil {
		t.Fatalf("manifest.Read() error = %v", err)
	}
	if gotManifest.Cols != 3 || gotManifest.Rows != 2 {
		t.Fatalf("manifest cols/rows = %d/%d, want 3/2", gotManifest.Cols, gotManifest.Rows)
	}
	if gotManifest.SheetSize == nil || *gotManifest.SheetSize != (manifest.Size{W: 48, H: 32}) {
		t.Fatalf("manifest sheet_size = %+v, want 48x32", gotManifest.SheetSize)
	}
	if gotManifest.Frames[2].Rect != (manifest.Rect{X: 32, Y: 0, W: 16, H: 16}) {
		t.Fatalf("frame 2 rect = %+v, want top row third column", gotManifest.Frames[2].Rect)
	}
	if gotManifest.Frames[3].Rect != (manifest.Rect{X: 0, Y: 16, W: 16, H: 16}) {
		t.Fatalf("frame 3 rect = %+v, want second row first column", gotManifest.Frames[3].Rect)
	}
}

func TestSheetExportPadsMixedSizes(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "export")
	ctx := testSheetContext(outDir, false, map[string]string{"cols": "2"}, "hero", []image.Point{{16, 16}, {32, 24}}, nil)

	result, err := Sheet{}.Export(ctx)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	data := result.Data.(map[string]any)
	if got := data["mixed_sizes"]; got != true {
		t.Fatalf("mixed_sizes = %v, want true", got)
	}

	img, err := pixel.LoadPNG(filepath.Join(outDir, "hero_sheet.png"))
	if err != nil {
		t.Fatalf("pixel.LoadPNG() error = %v", err)
	}
	if got := img.Bounds().Dx(); got != 64 {
		t.Fatalf("sheet width = %d, want 64", got)
	}
	if got := img.Bounds().Dy(); got != 24 {
		t.Fatalf("sheet height = %d, want 24", got)
	}
}

func TestSheetExportDryRunDoesNotWrite(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "export")
	ctx := testSheetContext(outDir, true, nil, "hero", []image.Point{{16, 16}}, nil)

	if _, err := (Sheet{}).Export(ctx); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "hero_sheet.png")); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(png) error = %v, want not exists", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "hero_sheet.json")); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(json) error = %v, want not exists", err)
	}
}

func testSheetContext(outDir string, dryRun bool, options map[string]string, subject string, sizes []image.Point, metadata []manifest.Frame) *internalexport.Context {
	frames := make([]internalexport.Frame, len(sizes))
	manifestFrames := make([]manifest.Frame, len(sizes))
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}}
	for i, size := range sizes {
		img := image.NewNRGBA(image.Rect(0, 0, size.X, size.Y))
		fillSheet(img, image.Rect(0, 0, size.X, size.Y), colors[i%len(colors)])
		frames[i] = internalexport.Frame{
			Index: i,
			Path:  frameName(i),
			Rect:  manifest.Rect{W: size.X, H: size.Y},
			Image: img,
		}
		manifestFrames[i] = manifest.Frame{Index: i, Path: frameName(i), Rect: manifest.Rect{W: size.X, H: size.Y}}
		if metadata == nil || i >= len(metadata) {
			continue
		}
		if metadata[i].Path != "" {
			manifestFrames[i].Path = metadata[i].Path
			frames[i].Path = metadata[i].Path
		}
		frames[i].Row = metadata[i].Row
		frames[i].Col = metadata[i].Col
		frames[i].Tag = metadata[i].Tag
		manifestFrames[i].Row = metadata[i].Row
		manifestFrames[i].Col = metadata[i].Col
		manifestFrames[i].DurationMS = metadata[i].DurationMS
		manifestFrames[i].Tag = metadata[i].Tag
		manifestFrames[i].Pivot = metadata[i].Pivot
	}
	return &internalexport.Context{
		Format:   "sheet",
		OutPath:  outDir,
		Frames:   frames,
		Options:  options,
		DryRun:   dryRun,
		Subject:  subject,
		FrameDir: filepath.Join("fixtures", subject),
		Manifest: &manifest.Manifest{Source: "input.png", Frames: manifestFrames},
	}
}

func frameName(index int) string {
	return fmt.Sprintf("frame_%03d.png", index)
}

func fillSheet(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
