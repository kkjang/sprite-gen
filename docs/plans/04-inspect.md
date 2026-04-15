# Plan 04 — Inspect

## Goal

Ship the first "real" feature: read-only image analysis. Introduces the `internal/pixel` package, which later plans (palette, snap, slice) will build on. Zero writes — this plan cannot corrupt anything, making it the safest place to start burning in the PNG I/O layer.

## Scope

**In:**
- `internal/pixel` package: PNG load, dimensions, per-pixel iteration, bbox detection, alpha scanning, unique-color counting
- `sprite-gen inspect sheet PATH` — describe a whole image; suspected grid layout
- `sprite-gen inspect frame PATH` — per-image bbox + pivot hint + AA-likelihood score
- Tests: pixel package unit tests + command-level tests with small fixtures

**Out:**
- Any image writing (palette extract is plan 05, which is the first writer)
- Any slicing logic (comes in plan 07)
- Actual palette extraction (plan 05; `inspect sheet` only reports a *count* of unique colors, not the colors themselves)

## File plan

```
sprite-gen/
  cmd/sprite-gen/
    cmd_inspect.go                      # flag parsing + dispatch to sheet/frame
  internal/
    pixel/
      load.go                           # LoadPNG
      bbox.go                           # BBox(img) returns tight non-transparent rect
      stats.go                          # Stats(img): colors, alpha histogram, aa score
      grid.go                           # GuessGrid(img): column/row count from gutters
      load_test.go
      bbox_test.go
      stats_test.go
      grid_test.go
  # Tests may generate small temporary PNG fixtures instead of committing
  # binary files under testdata/.
```

## Pixel package design

```go
// Package pixel provides PNG I/O and read-only analysis primitives
// that higher-level commands (inspect, snap, slice, palette) build on.
package pixel

// LoadPNG decodes a PNG file into an *image.NRGBA. Non-PNG or corrupt
// files return an error whose message names the path.
func LoadPNG(path string) (*image.NRGBA, error)

// BBox returns the tight rectangle containing all pixels with alpha
// > alphaMin. For a fully-opaque image this is img.Bounds(). For a
// fully-transparent image it is image.ZR.
func BBox(img image.Image, alphaMin uint8) image.Rectangle

// Stats summarizes an image: unique colors, fraction of pixels with
// fractional alpha (the "aa likelihood" signal), alpha histogram.
type Stats struct {
    W, H           int
    UniqueColors   int   // capped at some large value (e.g. 4096) for perf
    OpaquePixels   int
    TransparentPx  int
    FractionalPx   int   // alpha not in {0, 255}; high values ≈ AA or smooth gradient
    AAScore        float64 // FractionalPx / (OpaquePixels + FractionalPx)
}

func ComputeStats(img image.Image) Stats

// GuessGrid tries to detect a uniform cell grid by scanning for
// fully-transparent rows and columns (gutters). Returns zero-value
// and Confidence==0 when no grid structure is apparent.
type Grid struct {
    Cols, Rows     int
    CellW, CellH   int
    OffsetX, OffsetY int
    Confidence     float64 // 0..1
}

func GuessGrid(img image.Image) Grid
```

### `GuessGrid` algorithm (heuristic)

1. Scan rows; find runs of fully-transparent rows — these are row gutters.
2. Scan columns; do the same.
3. The content bands between gutters are candidate cells. Check that all
   candidate cells have the same width and same height.
4. If widths and heights match within ±1 pixel, emit the grid with
   `Confidence = 1.0 - mismatch_ratio`.
5. Fallback: if there are no full-transparent gutters (common case —
   opaque backgrounds, no padding), try to find the GCD of
   sub-image bounding boxes. Out of scope here; return Confidence=0.

The v1 of this is deliberately simple. `slice auto` in plan 07 will extend
it; `inspect sheet` just reports what it finds.

## Command design

### `sprite-gen inspect sheet PATH`

Flags: `--grid=auto|WxH|none` (default `auto`), global `--json`.

Text output (default):

```
path: knight_sheet.png
size: 128x32
colors: 18 (capped at 4096)
alpha: 512 transparent, 3584 opaque, 0 fractional
aa_score: 0.00
grid: 4x1 (cell 32x32, offset 0,0, confidence 1.00)
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "path": "knight_sheet.png",
    "w": 128, "h": 32,
    "unique_colors": 18,
    "alpha": {"transparent": 512, "opaque": 3584, "fractional": 0},
    "aa_score": 0.0,
    "grid": {"cols": 4, "rows": 1, "cell_w": 32, "cell_h": 32,
             "offset_x": 0, "offset_y": 0, "confidence": 1.0}
  }
}
```

When `--grid=none` the `grid` field is omitted. When `--grid=WxH` is
explicit, the handler reports the user-declared grid unchanged (no
detection) but still runs `GuessGrid` and notes any mismatch in a
`grid_warning` field.

### `sprite-gen inspect frame PATH`

Same shape but for a single-frame image. No grid fields. Adds:

```json
{
  "ok": true,
  "data": {
    "path": "knight_walk_1.png",
    "w": 32, "h": 32,
    "bbox": {"x": 4, "y": 2, "w": 24, "h": 28},
    "pivot_hint": {"x": 16, "y": 30, "anchor": "feet"},
    "unique_colors": 11,
    "aa_score": 0.03,
    "alpha": {"transparent": 600, "opaque": 424, "fractional": 0}
  }
}
```

`pivot_hint` here is a trivial default — bottom-center of the bbox for
`feet` anchor. Real pivot computation across a frame set lives in
plan 08 (`align`). Inspect reports only a single-frame guess.

## Testing

`internal/pixel` unit tests:
- `LoadPNG` on non-existent file returns error.
- `LoadPNG` on a valid 1x1 PNG returns `*image.NRGBA` with correct color.
- `BBox` on fully-opaque image == image bounds.
- `BBox` on 32x32 image with a 16x16 opaque square centered returns the expected rect.
- `ComputeStats` on `solid_16x16.png` reports `UniqueColors=1`, `AAScore=0`.
- `ComputeStats` on `aa_sample.png` reports `AAScore > 0.05`.
- `GuessGrid` on `grid_4x1_32px.png` returns `{Cols:4, Rows:1, CellW:32, CellH:32, Confidence:1.0}`.
- `GuessGrid` on `solid_16x16.png` returns `Confidence == 0` (no grid structure).

Command-level tests in `cmd_inspect_test.go`:
- `inspect sheet testdata/input/inspect/grid_4x1_32px.png --json` parses as an envelope with `ok: true`, `data.grid.cols == 4`.
- `inspect frame testdata/input/inspect/padded_16x16_in_32x32.png --json` reports `bbox.w == 16`, `bbox.h == 16`.
- Missing path argument → exit code 2, stderr has "missing path" or similar.
- Non-PNG file → exit code non-zero, actionable error.

## Acceptance criteria

1. `go test ./...` passes.
2. `sprite-gen inspect sheet testdata/input/inspect/grid_4x1_32px.png` prints the 4x1 grid detection.
3. `sprite-gen inspect sheet some_random.png --json | jq .` round-trips and `ok` is `true`.
4. `sprite-gen spec` now shows four commands: `version`, `spec`, `inspect sheet`, `inspect frame`.
5. No writes happen in any code path of this plan (the command never calls `os.Create`, `os.WriteFile`, etc).
6. `internal/pixel` has no imports outside stdlib except `golang.org/x/image/draw` if we need nearest-neighbor; even that we can avoid in plan 04 since inspect never resizes.

## Suggested commit message

```
feat(inspect): read-only sheet and frame analysis

Introduce internal/pixel with load + bbox + stats + grid-guess
primitives. Two new verbs: `inspect sheet` and `inspect frame`.
No writes. First plan that exercises the JSON envelope with a
real data payload.
```

## Notes for the next plan

- `palette extract` (plan 05) will reuse the unique-color counter but
  return the actual colors, not just the count. Factor it so that the
  core quantize loop is reachable by both without duplicating code.
- `GuessGrid` is deliberately unsophisticated. Plan 07 (`slice`) will
  likely add a GCD-based mode for images without transparent gutters.
  Don't over-engineer it here.
- `Stats.AAScore` is an *indicator*, not a verdict. Don't tune thresholds
  in this plan; plan 06 (`snap pixels`) is where AA removal lives and
  will define what "too much AA" means in context.
