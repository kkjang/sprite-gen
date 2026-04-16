# Plan 07 — Slice

## Goal

Turn one sprite sheet PNG into individually-named frame PNGs plus a JSON manifest. This is the gateway to everything animation-related: plans 07 (align+diff) and 08 (export) all start from a directory of frames + manifest.

## Scope

**In:**
- `internal/manifest` package: frame manifest JSON structure, read/write
- `internal/pixel` extended: crop/extract sub-image by rectangle
- `sprite-gen prep alpha PATH` — remove low-alpha background haze before slicing while preserving canvas layout
- `sprite-gen slice grid PATH` — cut by explicit column/row count
- `sprite-gen slice auto PATH` — detect rows via transparent gutters
- Tests with fixtures and golden frame files

**Out:**
- Alignment or pivot computation (plan 07)
- Packing frames back into a sheet (plan 08)
- Alpha-based sprite detection within cells (that's a polish feature, not v1)

## File plan

```
sprite-gen/
  cmd/sprite-gen/
    cmd_prep.go                         # flag parsing for prep alpha
    cmd_slice.go                        # flag parsing for grid + auto
  internal/
    manifest/
      manifest.go                       # Frame, Manifest structs; Read/Write
      manifest_test.go
    pixel/
      crop.go                           # Crop(img, rect) *image.NRGBA
      crop_test.go
  testdata/
    input/
      slice/
        walk_4x1.png                    # 128x32 sheet, 4 frames
        run_2x2.png                     # 64x64 sheet, 4 frames in 2 rows
        gutter_strip.png                # frames separated by transparent gutters
    golden/
      slice/
        walk_4x1/
          frame_000.png
          frame_001.png
          frame_002.png
          frame_003.png
          manifest.json
        gutter_strip/
          frame_000.png
          frame_001.png
          manifest.json
```

## Manifest package design

```go
// Package manifest defines the shared frame-set format written by slice
// and read by align, diff, and export.
package manifest

// Frame describes one extracted frame.
type Frame struct {
    Index   int            `json:"index"`
    Path    string         `json:"path"`    // relative to manifest dir
    Rect    Rect           `json:"rect"`    // source rect in the original sheet
    W, H    int            `json:"w,omitempty"`
    Pivot   *Point         `json:"pivot,omitempty"` // set by align (plan 07)
}

type Rect struct {
    X, Y, W, H int
}

type Point struct {
    X, Y int
}

// Manifest is written alongside the frame PNGs.
type Manifest struct {
    Version  int     `json:"version"`   // always 1 for now
    Source   string  `json:"source"`    // original sheet path (relative or abs)
    CellW    int     `json:"cell_w"`
    CellH    int     `json:"cell_h"`
    Cols     int     `json:"cols"`
    Rows     int     `json:"rows"`
    Frames   []Frame `json:"frames"`
}

func Read(path string) (*Manifest, error)
func Write(path string, m *Manifest) error
```

`Manifest.Version` starts at 1. If we need backward-incompatible changes
later, bump it and add a migration shim. The `Path` field in each `Frame`
is always relative to the directory containing `manifest.json` so the
frame set is portable.

## Command design

### `sprite-gen prep alpha PATH`

Flags:
- `--alpha-threshold N` (default 128): pixels with alpha below N become transparent
- `--out FILE` (default: `./out/<subject>/prep/clean.png`)
- `--dry-run`
- global `--json`

Behavior:
1. Load the source image.
2. Run `pixel.ThresholdAlpha(img, 0, alphaThreshold)`.
3. Write a cleaned PNG with the same canvas size.

This is intentionally minimal. It is for sheet-shaped inputs that are almost
sliceable but still have low-alpha haze or glow in the gutters. It does not do
component detection, morphology, trimming, or cropping.

### `sprite-gen slice grid PATH`

Flags:
- `--cols N` (required)
- `--rows N` (default 1)
- `--trim` (default false): crop transparent border from each frame
- `--out DIR` (default: `./out/<subject>/slice/`)
- `--dry-run`
- global `--json`

Behavior:
1. Load image with `pixel.LoadPNG`.
2. Compute `cellW = img.W / cols`, `cellH = img.H / rows`.
3. For each `(col, row)` pair: crop the cell rectangle, optionally trim,
   save as `frame_NNN.png` (zero-padded to 3 digits).
4. Write `manifest.json` alongside the frames.

Text output:

```
wrote: out/walk_4x1/slice/ (4 frames, 32x32 each)
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "out": "out/walk_4x1/slice/",
    "frames": [
      {"index": 0, "path": "frame_000.png", "rect": {"x":0,"y":0,"w":32,"h":32}},
      {"index": 1, "path": "frame_001.png", "rect": {"x":32,"y":0,"w":32,"h":32}},
      {"index": 2, "path": "frame_002.png", "rect": {"x":64,"y":0,"w":32,"h":32}},
      {"index": 3, "path": "frame_003.png", "rect": {"x":96,"y":0,"w":32,"h":32}}
    ],
    "cols": 4, "rows": 1, "cell_w": 32, "cell_h": 32,
    "dry_run": false
  }
}
```

Error if `img.W % cols != 0`: report the pixel remainder so the caller
can diagnose a mismatched grid count.

### `sprite-gen slice auto PATH`

Detects row boundaries by looking for runs of fully-transparent rows,
then uses `GuessGrid` logic from plan 03 on the resulting bands.

Flags:
- `--min-gap N` (default 1): minimum transparent-row run to count as a gutter
- `--out DIR` (default: `./out/<subject>/slice/`)
- `--dry-run`
- global `--json`

Behavior:
1. Load image.
2. Call `pixel.GuessGrid(img)`. If `Confidence < 0.5`, fail actionably instead
   of guessing a fallback layout.
3. Proceed as `slice grid` using the detected grid parameters.
4. Report `detected` grid alongside output.

JSON output adds `"detected": {"cols":4,"rows":1,"confidence":0.97}`.

## Testing

`internal/manifest/manifest_test.go`:
- `Write` → `Read` round-trip preserves all fields.
- `Read` on a missing file returns an error naming the path.
- `Read` on a malformed file returns an actionable error.
- `Manifest.Version` defaults to 1.

`internal/pixel/crop_test.go`:
- `Crop` on a 32x32 image with rect `{0,0,16,16}` returns a 16x16 image.
- `Crop` with a rect that extends outside image bounds returns an error.
- `Crop` on a solid-color image returns the same color in all pixels.

Command-level tests:
- `prep alpha testdata/input/slice/noisy_sheet.png --alpha-threshold 128 --json` → envelope with `changed_pixels > 0`.
- `slice grid testdata/input/slice/walk_4x1.png --cols 4 --json` → envelope with 4 frames, `cell_w: 32`.
- Golden: frame files from `walk_4x1.png --cols 4` match `golden/slice/walk_4x1/frame_*.png`.
- `slice grid` with `--cols 5` on a 128px-wide image → exit non-zero with "128 is not divisible by 5".
- `slice auto testdata/input/slice/gutter_strip.png --json` → envelope with `detected.confidence > 0.8`.
- `slice grid` with `--dry-run` → exit 0, no files created.

## Acceptance criteria

1. `go test ./...` passes including golden frame comparisons.
2. `sprite-gen prep alpha walk_4x1.png --alpha-threshold 128` writes `out/walk_4x1/prep/clean.png` without changing canvas size.
3. `sprite-gen slice grid walk_4x1.png --cols 4` creates 4 frame PNGs + `manifest.json` in `out/walk_4x1/slice/`.
4. Re-running the same command overwrites without creating duplicates.
5. The manifest JSON is valid and `manifest.Read` parses it cleanly.
6. `sprite-gen slice auto gutter_strip.png` produces the same output as `slice grid` with the correct explicit grid.
7. `sprite-gen spec` shows eleven commands.

## Suggested commit message

```
feat(slice): sheet-to-frames slicing with manifest

Add internal/manifest for the shared frame-set JSON format.
Extend internal/pixel with crop and add `prep alpha` for
low-alpha sheet cleanup. Two new slicing verbs: `slice grid`
(explicit cell count) and `slice auto` (transparent-gutter
detection). Manifest written alongside frames; consumed by align,
diff, and export in later plans.
```

## Notes for the next plan

- The manifest's `Pivot` field is populated by plan 07 (`align`). Slice
  leaves it null; export reads it and falls back to frame center if null.
- `prep alpha` is intentionally preserved as an explicit step even though plan
  08 (`segment subjects`) also thresholds alpha internally. Slice callers may
  want a cleaned sheet artifact on disk; segment callers should not be forced
  to run prep first.
- Frame filenames are zero-padded to 3 digits (`frame_000.png`) to allow
  natural sort up to 999 frames. If a sheet needs more, the padding width
  should grow automatically based on frame count. Address if/when needed.
- Plan 08 (`export`) reads a directory of frames + manifest; it does not
  depend on knowing how the frames were sliced. The manifest is the contract.
