# Plan 08 — Segment Subjects

## Goal

Turn loosely-structured AI-generated canvases into clean, engine-ready frame sets. Real `gpt-image-1` outputs routinely ignore prompted sprite-sheet constraints: they arrive as oversized canvases with glow halos, soft edges, semi-transparent pixels, and subjects scattered across a much larger area than the requested cell size. `slice grid` and `slice auto` (plan 07) both assume cleanly-structured input and produce unusable frames on this kind of messy canvas.

This plan introduces a segmentation path parallel to `slice`: detect each subject on the canvas as a connected component, normalize every subject into a fixed-size cell with consistent baseline alignment, and write out the same `frames + manifest` contract that `align`, `diff`, and `export` already consume. After this plan, an agent can run `generate image` → `segment subjects` → `align frames` → `export` without any manual cleanup.

## Scope

**In:**
- `internal/segment` package: connected-component labeling over an alpha mask; component bounding boxes; component filtering by minimum area; optional morphological erosion/dilation
- `internal/pixel` extended: `AlphaMask` (threshold to binary mask) and `PlaceInCell` (paste a cropped region into a fixed-size cell with an anchor)
- `sprite-gen segment subjects PATH` — threshold + connected components + per-subject crop + cell normalization + baseline alignment; writes frames and manifest using the plan-07 format
- Tests: unit tests for the labeler on synthetic fixtures, cell-normalization tests for each anchor, golden-frame command-level tests

**Out:**
- Inter-frame drift correction across a sequence (that is `align frames`, plan 09 — `segment` positions each subject in its cell by anchor alone; fine-grained drift is a separate step)
- Subject classification / recognition (e.g., "this blob is a character, that one is a shadow") — out of scope; callers use `--min-area`, sort order, and manual inspection to filter
- Background subtraction beyond alpha thresholding (e.g., color-based keying) — the canvases we see from `gpt-image-1` already come on transparent backgrounds; colored-background canvases are a future enhancement
- Vertical subject detection (subjects stacked top-to-bottom across an image) — the v1 heuristic sorts left-to-right only; multi-row detection can ride on plan 07's grid logic later

## Why this slot in the sequencing

| # | Plan | Why `segment` sits here |
|---|---|---|
| 06 | snap | Introduces alpha thresholding primitives (`ThresholdAlpha`, `CountFractional`). `segment` reuses them. |
| 07 | slice | Introduces the `manifest` package and `pixel.Crop`. `segment` reuses both. |
| **08** | **segment (this plan)** | **Alternate path from one image to `frames + manifest`: connected components instead of a grid.** |
| 09 | align + diff | Operates on any `frames + manifest` directory — works on the output of either `slice` or `segment` with no changes. |
| 10 | export | Same — consumes a frame directory regardless of how it was produced. |

Placing `segment` after `slice` keeps `slice` as the smallest possible PR and lets `segment` build on the manifest + crop primitives introduced there. Both commands produce interchangeable outputs; callers pick by input shape (clean sheet → `slice`, messy canvas → `segment`).

## File plan

```
sprite-gen/
  cmd/sprite-gen/
    cmd_segment.go                      # flag parsing for segment subjects
    cmd_segment_test.go
  internal/
    pixel/
      alpha_mask.go                     # AlphaMask, MorphErode, MorphDilate
      alpha_mask_test.go
      place.go                          # PlaceInCell(img, bbox, cell, anchor)
      place_test.go
    segment/
      components.go                     # Label, Component, Filter, SortLTR
      components_test.go
      normalize.go                      # NormalizeToCell, AutoCell
      normalize_test.go
  testdata/
    # Tests generate synthetic PNG fixtures programmatically rather than
    # committing binary files. See internal/pixel/load_test.go for the
    # in-repo precedent.
```

No new third-party dependencies. The work is all stdlib `image` + integer arithmetic.

## Package design

### `internal/segment/components.go`

```go
// Package segment finds discrete subjects on a transparent-background
// canvas and helps normalize each into a fixed-size cell.
package segment

// Component is one connected region of non-transparent pixels.
type Component struct {
    ID    int             // 1..N, 0 reserved for background
    BBox  image.Rectangle // tight bounds within the source image
    Area  int             // count of pixels belonging to the component
}

// Label runs 4-connected flood labeling over mask. A pixel belongs to
// the foreground when mask.Pix[i] != 0. Returns the per-pixel label
// image (uint16; 0 = background) plus one Component entry per label.
// 4-connectivity is deliberate: it produces tighter, more predictable
// splits on pixel art than 8-connectivity, which tends to merge
// diagonally-touching subjects that look distinct to a human.
func Label(mask *image.Alpha) (labels []uint16, components []Component)

// Filter returns only the components whose Area >= minArea.
func Filter(cs []Component, minArea int) []Component

// SortLTR sorts components left-to-right by BBox.Min.X. Ties break on
// Min.Y (top first). This matches the way humans read a horizontal
// sprite strip and is the order downstream tools will expect.
func SortLTR(cs []Component)
```

### `internal/segment/normalize.go`

```go
// Anchor mirrors internal/align.Anchor semantics. Kept as a separate
// type to avoid an import cycle with plan 09 and because the normalize
// step only supports a small subset of anchors meaningful *within* a
// cell (centroid is left for align to compute across frames).
type Anchor string

const (
    AnchorFeet      Anchor = "feet"
    AnchorCenter    Anchor = "center"
    AnchorTop       Anchor = "top"
)

// Fit controls how an oversized subject is handled.
type Fit string

const (
    FitError     Fit = "error"  // default; subject larger than cell is an error
    FitDownscale Fit = "scale"  // integer-nearest downscale until it fits
    FitCrop      Fit = "crop"   // center-crop the subject to the cell
)

// NormalizeToCell places the source rect from src into a new NRGBA image
// of size cell using the given anchor. Transparent pixels fill any space
// not occupied by the subject. Returns an error when Fit == FitError and
// the source rect does not fit inside cell.
func NormalizeToCell(src *image.NRGBA, srcRect image.Rectangle, cell image.Point, anchor Anchor, fit Fit) (*image.NRGBA, error)

// AutoCell picks a target cell size that accommodates every component.
// It returns the bounding cell rounded up to a multiple of `round`
// (default 8 when round <= 0). This keeps generated cells at tidy
// pixel-art sizes (16, 24, 32, 40, …) without requiring the caller to
// measure each canvas manually.
func AutoCell(cs []Component, round int) image.Point
```

Placement math inside `NormalizeToCell`:

- `feet`: subject's `BBox.Max` is anchored to `(cell.X/2, cell.Y)` (bottom-center).
- `center`: subject's BBox midpoint is placed at `(cell.X/2, cell.Y/2)`.
- `top`: subject's `BBox.Min` is placed at `(cell.X/2, 0)` (top-center).

Offsets are clamped so the subject never extends outside the destination cell (a pre-check on oversize triggers the `Fit` policy).

### `internal/pixel/alpha_mask.go`

```go
// AlphaMask returns a binary image.Alpha where every pixel with alpha
// in src >= threshold becomes 255, every other pixel becomes 0. This is
// the input to segment.Label and the place where glow halos get killed.
func AlphaMask(src image.Image, threshold uint8) *image.Alpha

// MorphErode shrinks the foreground in mask by `iterations` using a
// 3x3 square structuring element (standard binary erosion). Useful for
// stripping thin soft-edge halos before labeling.
func MorphErode(mask *image.Alpha, iterations int) *image.Alpha

// MorphDilate grows the foreground by `iterations`. Used to reverse
// over-aggressive erosion if a caller wants to shave a halo without
// losing internal detail.
func MorphDilate(mask *image.Alpha, iterations int) *image.Alpha
```

These live in `internal/pixel` because they are alpha-channel operations and symmetric to the existing `ThresholdAlpha` / `CountFractional` from plan 06. `internal/segment` depends on `pixel` but not the other way around.

### `internal/pixel/place.go`

```go
// PlaceInCell crops src to srcRect and draws the result into a new
// NRGBA image sized cell, with the crop anchored per the offset
// provided by the caller. This is the low-level primitive
// segment.NormalizeToCell uses; exported so other future commands
// (future `pack` variants, etc.) can build on it.
func PlaceInCell(src *image.NRGBA, srcRect image.Rectangle, cell image.Point, offset image.Point) *image.NRGBA
```

## Command design

### `sprite-gen segment subjects PATH`

Reads one PNG, produces a frame directory and manifest. One-shot by default; every step can be tuned via flags.

**Flags:**

| Flag | Default | Notes |
|---|---|---|
| `--alpha-threshold N` | `128` | Pixels with alpha < N become background before labeling. |
| `--erode N` | `0` | Binary erosions before labeling; bump to `1` or `2` to kill thick glow halos. |
| `--dilate N` | `0` | Binary dilations after erosion; use to restore shape detail after aggressive erode. |
| `--min-area N` | auto | Minimum pixel count per component. Default: `max(64, 0.001 * image_area)`. Captures "ignore speckles and stray glow islands". |
| `--expected N` | unset | If set, fail when the detected component count != N. Agent-friendly guardrail. |
| `--cell WxH` | auto | Output cell size. `auto` calls `segment.AutoCell(components, 8)`. |
| `--anchor feet\|center\|top` | `feet` | Placement within the cell. `feet` is the right default for platformer characters. |
| `--fit error\|scale\|crop` | `error` | What to do when a detected subject is larger than the cell. |
| `--sort ltr\|area\|none` | `ltr` | Frame ordering. `ltr` = left-to-right (matches how humans read a walk cycle). |
| `--out DIR` | `./out/<subject>/segment/` | Output directory. |
| `--dry-run` | `false` | Print the plan, write nothing. |
| `--json` | global | `{ok, data, error}` envelope. |

**Behavior:**

1. Load the source image with `pixel.LoadPNG`.
2. Build a binary alpha mask with `pixel.AlphaMask(src, alphaThreshold)`.
3. Optionally `MorphErode`/`MorphDilate` the mask.
4. `segment.Label` the mask → components.
5. `segment.Filter` by `--min-area`.
6. Validate `--expected` if set; error actionably when the count is wrong.
7. Resolve the target cell (either user-supplied or `segment.AutoCell`).
8. `segment.SortLTR` (or other sort mode).
9. For each component: `pixel.Crop` to component BBox → `segment.NormalizeToCell` into a fresh cell-sized NRGBA → save as `frame_NNN.png`.
10. Write `manifest.json` with the plan-07 format: `CellW`, `CellH`, `Cols = len(frames)`, `Rows = 1`, and a `Frame` per output. `Frame.Rect` records the *source* rectangle on the original canvas (so `manifest.Read` gives an agent enough info to re-open the original and see what got picked up).

**Text output (default):**

```
wrote: out/knight_canvas/segment/ (4 frames, 32x32 each)
detected: 4 components (min_area=64, threshold=128, erode=0)
anchor: feet
```

**JSON output:**

```json
{
  "ok": true,
  "data": {
    "out": "out/knight_canvas/segment/",
    "cell": {"w": 32, "h": 32},
    "anchor": "feet",
    "threshold": 128,
    "erode": 0,
    "dilate": 0,
    "min_area": 64,
    "components_detected": 7,
    "components_kept": 4,
    "frames": [
      {"index": 0, "path": "frame_000.png", "src_rect": {"x": 64, "y": 312, "w": 27, "h": 30}, "area": 612},
      {"index": 1, "path": "frame_001.png", "src_rect": {"x": 198, "y": 315, "w": 26, "h": 29}, "area": 587},
      {"index": 2, "path": "frame_002.png", "src_rect": {"x": 334, "y": 310, "w": 28, "h": 31}, "area": 640},
      {"index": 3, "path": "frame_003.png", "src_rect": {"x": 470, "y": 314, "w": 27, "h": 30}, "area": 605}
    ],
    "dry_run": false
  }
}
```

`components_detected` vs `components_kept` makes the min-area filter visible — an agent inspecting the envelope can tell whether to loosen the filter if too many subjects were dropped.

**Error surface:**

- `--expected 4` and only 3 components detected → exit 1 with `expected 4 subjects, found 3; lower --min-area (now 64) or raise --alpha-threshold (now 128)`.
- Subject larger than cell with `--fit error` → exit 1 with `subject at src_rect=(…) exceeds cell 32x32; set --fit scale or --cell WxH`.
- Zero components survive the filter → exit 1 with the filter parameters in the message.
- `--cell 32x32` with `--alpha-threshold 0` and a full-opaque canvas → one giant component → likely actionable error recommending a sensible threshold.

### Interaction with `inspect`

`inspect sheet` remains the diagnostic entry point. A natural agent workflow is:

```bash
sprite-gen inspect sheet messy.png --json
# -> reports aa_score, non-trivial fractional-alpha count, huge bbox
sprite-gen segment subjects messy.png --cell 32x32 --expected 4 --json
# -> writes 4 clean 32x32 frames to out/messy/segment/
```

No code dependency between the two commands; they just compose well because both operate on one image and speak the same envelope shape.

## Testing

`internal/segment/components_test.go`:
- `Label` on an all-background mask returns zero components.
- `Label` on a mask with two disjoint 8x8 squares returns two components with the expected bounding boxes and areas (64 each).
- `Label` on a C-shaped mask returns a single component (interior hole doesn't split the component under 4-connectivity).
- `Filter(components, 100)` drops components below the threshold but keeps equal-to.
- `SortLTR` is stable on ties: two equal-X components sort by Y ascending.

`internal/segment/normalize_test.go`:
- `AutoCell` on components of bbox 27x30, 26x29, 28x31 with `round=8` returns (32, 32).
- `AutoCell` with `round=1` returns the exact max (28, 31).
- `NormalizeToCell` with `anchor=feet` places a 16x16 subject so its bottom-center aligns to (cell.W/2, cell.H) — verified by finding the opaque pixel nearest the cell bottom.
- `NormalizeToCell` with `anchor=center` places a 16x16 subject centered in a 32x32 cell — bbox min is (8,8).
- `NormalizeToCell` with `fit=error` on an oversized subject returns an error.
- `NormalizeToCell` with `fit=scale` on a 2x-too-large subject returns a cell-sized image whose opaque area equals 1/4 of the input's opaque area (nearest-neighbor half-size).

`internal/pixel/alpha_mask_test.go`:
- `AlphaMask` with threshold=128 on a pixel with alpha=127 → background (0); alpha=128 → foreground (255).
- `MorphErode` by 1 on an 8x8 square mask shrinks it to 6x6.
- `MorphDilate` by 1 on a single-pixel mask grows it to a 3x3 square.
- `MorphErode(MorphDilate(mask, 1), 1)` is not guaranteed to equal `mask` (document this; opening/closing is lossy at the mask boundary). The test just exercises both directions for coverage.

`internal/pixel/place_test.go`:
- `PlaceInCell` with offset=(0,0) copies srcRect into the top-left of the output.
- `PlaceInCell` with an offset that would place the subject outside the cell leaves the cell transparent where the subject would have been clipped. (Clipping, not an error — callers do the pre-check.)

Command-level tests in `cmd/sprite-gen/cmd_segment_test.go` (following the `cmd_inspect_test.go` pattern, generating PNG fixtures at test time):
- Synthetic 512x256 canvas with four 24x28 blobs spaced along a row; `segment subjects --cell 32x32 --expected 4 --json` → envelope with 4 frames, `cell: {w:32, h:32}`, frames sorted left-to-right.
- Same canvas with `--expected 5` → exit 1, stderr names both counts and suggests loosening `--min-area` or raising threshold.
- `--dry-run` → exit 0, envelope with `dry_run: true`, no files written (verified by stat-ing the output dir).
- Missing path argument → exit 2.
- Non-PNG file → exit non-zero with actionable message (delegates to `pixel.LoadPNG` like existing commands).
- `--cell 16x16` on 24x28 blobs with `--fit error` → exit 1 with `subject … exceeds cell 16x16`.
- Same with `--fit scale` → exit 0, output frames are 16x16.

Round-trip with plan 07 (`slice`):
- Generate a 128x32 sheet with `sheet-png` export (post-plan-10, not required for this plan's tests; documented as a manual verification).

## Acceptance criteria

1. `go test ./...` passes.
2. `sprite-gen segment subjects synthetic_4_blobs.png --cell 32x32 --expected 4` writes 4 `frame_NNN.png` files and a valid `manifest.json` to `./out/synthetic_4_blobs/segment/`.
3. `manifest.Read` on the output parses cleanly and `Frame.Rect` fields point to the *source* rectangles on the input canvas.
4. `sprite-gen spec` shows `segment subjects` with all documented flags, so an agent discovering the command surface via `spec` finds it alongside `slice grid` / `slice auto`.
5. Re-running the same command overwrites the output deterministically (byte-identical PNGs given byte-identical input).
6. `--dry-run` produces no files and exits 0.
7. No new non-stdlib dependencies.
8. The existing `inspect`, `slice`, `align`, `diff`, and `export` commands continue to pass their tests unchanged (segment's additions to `internal/pixel` are additive — no existing exports are renamed or removed).

## Suggested commit message

```
feat(segment): subject segmentation for messy generated canvases

Add internal/segment (connected components, cell normalization) and
extend internal/pixel with binary alpha masks and morphological
erode/dilate. One new verb: `segment subjects` — threshold, label,
filter, crop, and baseline-align each subject into a fixed-size cell,
writing frames + manifest in the plan-07 format. Alternative path to
the frame-set contract from a messy AI-generated canvas; composes
with align, diff, and export unchanged.
```

## Notes for the next plan

- Plan 09 (`align frames`) takes the output of `segment subjects` without modification. The per-subject baseline alignment `segment` does is cell-local; `align frames` still earns its keep by fixing cross-frame drift that anchor-based placement doesn't catch (e.g., a walk cycle where the bbox changes shape from frame to frame).
- The morphological primitives (`MorphErode`, `MorphDilate`) are a stronger cleanup hammer than plan 06's `ThresholdAlpha` alone. If later work wants a dedicated `clean` command to apply erode/dilate without segmenting, it can build on these primitives directly.
- `segment subjects` is horizontal-first. A follow-up `segment grid` that recovers a rough row/column structure from scattered subjects (2D layout detection) is a reasonable future plan, but intentionally not in scope here.
- If a caller has a multi-row canvas (e.g., walk + idle on the same sheet), the current recommendation is to crop rows manually with `slice grid --rows N` first, then run `segment subjects` on each row. Built-in multi-row support is tracked in `future-enhancements.md`.
- The component labels produced by `segment.Label` are kept internal for now. If future work needs a visual overlay of what got segmented (for agent verification), a `segment preview` command could emit a colorized labels PNG alongside the frames. Don't add it until a caller needs it.
