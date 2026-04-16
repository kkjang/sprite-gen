# Plan 09 — Align + Diff

Status: implemented.

## Goal

Fix sub-pixel drift between animation frames (the most common flaw in AI-generated walk cycles) and verify corrections with a pixel-level diff. Both commands operate on a directory of frames produced by plan 07 (`slice`) or plan 08 (`segment`).

## Scope

**In:**
- `internal/align` package: per-frame pivot computation using centroid, bbox, or feet anchor; offset application
- `internal/diff` package: pixel-level comparison between two images
- `sprite-gen align frames DIR` — re-center all frames on a shared pivot
- `sprite-gen diff frames A B` — pixel diff between two single frames
- Manifest update: `align` writes pivot data back into `manifest.json`
- Package and command tests with synthetic frame sets

**Out:**
- Batch diffing (diff one frame against all others in a set)
- Diff visualization beyond a simple overlay PNG
- Temporal smoothing of pivot paths across frames

## File plan

```
sprite-gen/
  cmd/sprite-gen/
    cmd_align.go                        # flag parsing for align frames
    cmd_diff.go                         # flag parsing for diff frames
    cmd_align_test.go                   # synthetic drifting frame-set command tests
    cmd_diff_test.go                    # synthetic diff command tests
  internal/
    align/
      align.go                          # Pivot, Anchor, ComputePivot, AlignFrames
      align_test.go
    diff/
      diff.go                           # Diff, Result, DiffImage
      diff_test.go
```

## Align package design

```go
// Package align computes per-frame pivot points and adjusts frame
// images so all frames share a consistent pivot position.
package align

// Anchor determines which geometric feature is used as the pivot reference.
type Anchor string

const (
    AnchorCentroid Anchor = "centroid" // center-of-mass of opaque pixels
    AnchorBBox     Anchor = "bbox"     // center of the non-transparent bounding box
    AnchorFeet     Anchor = "feet"     // bottom-center of the bounding box
)

// Pivot is a point in frame-local coordinates.
type Pivot struct {
    X, Y int
}

// ComputePivot returns the pivot point for img using the given anchor.
func ComputePivot(img image.Image, anchor Anchor) Pivot

// AlignFrames takes a slice of frame images and their pivots, computes
// the median pivot across all frames (the "target" pivot), and returns
// new images translated so every frame's pivot coincides with the target.
//
// The target pivot is the component-wise median (not mean) of all frame
// pivots — median is robust to outlier frames with unexpected content.
//
// Returned images are written onto one shared transparent canvas so the
// target pivot lands at the same pixel coordinate in every output frame.
func AlignFrames(imgs []image.Image, pivots []Pivot) (aligned []*image.NRGBA, target Pivot, err error)
```

### Pivot computation details

**Centroid**: sum `(x * alpha, y * alpha)` over all pixels, divide by
total alpha. Integer arithmetic; round to nearest pixel.

**BBox center**: `(bbox.X + bbox.W/2, bbox.Y + bbox.H/2)` using
`pixel.BBox`.

**Feet**: `(bbox.X + bbox.W/2, bbox.Y + bbox.H - 1)` — bottom-center.
This is the most useful anchor for platformer characters standing on
a ground plane.

### Translation without cropping

If a frame's pivot is at `(px, py)` and the target pivot is at `(tx, ty)`,
the frame needs to shift by `(tx-px, ty-py)`. The implementation computes the
union of all translated frame bounds and uses that union as one shared output
canvas. Each frame is then drawn at its translated offset using `image/draw`.

All aligned frames therefore share the same final W/H, and the manifest records
that common size plus a common output-space pivot for each frame.

## Diff package design

```go
// Package diff compares two images pixel-by-pixel.
package diff

// Result summarizes the comparison.
type Result struct {
    DiffPixels  int
    TotalPixels int
    Percent     float64         // DiffPixels/TotalPixels*100
    BBox        image.Rectangle // bounding box of changed pixels; zero if no diff
}

// Compare returns a Result. Pixels differ when any channel (R,G,B,A)
// differs by more than tolerance (0 = exact match).
func Compare(a, b image.Image, tolerance uint8) Result

// DiffImage returns an NRGBA image where:
//   - identical pixels are drawn at 25% opacity (faint gray)
//   - differing pixels are drawn in red (fully opaque)
// If the images have different sizes the smaller is padded with
// transparent pixels for comparison purposes.
func DiffImage(a, b image.Image, tolerance uint8) *image.NRGBA
```

## Command design

### `sprite-gen align frames DIR`

Reads all `frame_*.png` files from DIR (or just the frame list in
`manifest.json` if present) and aligns them.

Flags:
- `--anchor centroid|bbox|feet` (default `feet`)
- `--out DIR` (default: `./out/<subject>/align/`)
- `--dry-run`
- global `--json`

Behavior:
1. Load manifest from DIR (or auto-discover `frame_*.png`).
2. Load all frame images.
3. Call `align.ComputePivot` on each.
4. Call `align.AlignFrames`.
5. Save each aligned frame.
6. Update (or create) `manifest.json` in the output dir with pivot fields populated.
7. Preserve each frame's source-space `rect` from the input manifest when one exists.

Text output:

```
wrote: out/drifting_walk/align/ (4 frames)
anchor: feet
target_pivot: 16,31
frame offsets: [0,2] [-1,0] [0,0] [1,-1]
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "out": "out/drifting_walk/align",
    "anchor": "feet",
    "target_pivot": {"x": 16, "y": 31},
    "frames": [
      {"index": 0, "path": "frame_000.png", "dx": 0, "dy": 2, "pivot": {"x":16,"y":29}},
      {"index": 1, "path": "frame_001.png", "dx": -1, "dy": 0, "pivot": {"x":17,"y":31}},
      {"index": 2, "path": "frame_002.png", "dx": 0, "dy": 0, "pivot": {"x":16,"y":31}},
      {"index": 3, "path": "frame_003.png", "dx": 1, "dy": -1, "pivot": {"x":15,"y":32}}
    ]
  }
}
```

### `sprite-gen diff frames A B`

Compare two single-frame PNG files.

Flags:
- `--tolerance N` (default 0): channel difference threshold to count as a diff
- `--out FILE` (default: `./out/<subjectA>_vs_<subjectB>/diff/diff.png`)
- `--dry-run`
- global `--json`

Behavior:
1. Load both images.
2. Call `diff.Compare(a, b, tolerance)`.
3. Call `diff.DiffImage(a, b, tolerance)`.
4. Save diff overlay PNG.
5. If the image sizes differ, compare on a transparent-padded union canvas and
   report the size mismatch in structured output.

JSON output:

```json
{
  "ok": true,
  "data": {
    "diff_pixels": 47,
    "total_pixels": 1024,
    "percent": 4.59,
    "bbox": {"x": 3, "y": 5, "w": 18, "h": 22},
    "tolerance": 0,
    "out": "out/frame_000_vs_frame_001/diff/diff.png",
    "size_mismatch": {
      "a": {"w": 32, "h": 32},
      "b": {"w": 34, "h": 32}
    }
  }
}
```

## Testing

`internal/align/align_test.go`:
- `ComputePivot` with `AnchorBBox` on a 32x32 image with a 16x28 opaque region
  at offset (8,2) returns `(16, 16)`.
- `ComputePivot` with `AnchorFeet` returns the bottom-center of the bbox.
- `ComputePivot` with `AnchorCentroid` on a uniform solid square returns the
  center pixel.
- `AlignFrames` on two frames offset by (0,2) and (0,-2) returns frames where
  both pivots coincide on a shared canvas.
- `AlignFrames` on already-aligned frames is idempotent (no translation applied).

`internal/diff/diff_test.go`:
- `Compare(img, img, 0)` returns `DiffPixels == 0`.
- `Compare(all_red, all_blue, 0)` returns `DiffPixels == total_pixels`.
- `Compare` with `tolerance=10` ignores differences ≤ 10 per channel.
- `DiffImage` returns a PNG where pixels identified as differing are red.

Command-level tests:
- `align frames <synthetic-dir> --anchor feet --json` → envelope with 4 frames, `target_pivot.y > 0`.
- `align frames` on a non-existent dir → exit non-zero, actionable error.
- `diff frames <synthetic-a.png> <synthetic-b.png> --json` → envelope with `diff_pixels > 0`.
- `diff frames A A --json` → `diff_pixels == 0`, `percent == 0`.
- `diff frames` with mismatched sizes → exit 0 but report size mismatch in JSON.

## Acceptance criteria

1. `go test ./...` passes.
2. Running on a drifting frame set:
   ```bash
   sprite-gen align frames frames/ --anchor feet
   sprite-gen diff frames out/<subject>/align/frame_000.png \
                          out/<subject>/align/frame_001.png
   ```
   The aligned frames have smaller `diff_pixels` than the unaligned originals.
3. Manifest in the output dir has `pivot` fields set for each frame.
4. `--dry-run` on `align frames` exits 0 with no files created.
5. `sprite-gen spec` shows fourteen commands.

## Suggested commit message

```
feat(align,diff): drift correction and frame comparison

Add internal/align (centroid/bbox/feet pivot + translation) and
internal/diff (pixel comparison + diff overlay). Two new verbs:
`align frames` (correct drift, update manifest pivots) and
`diff frames` (pixel-level diff between two frame PNGs).
```

## Notes for the next plan

- Plan 10 (`export`) reads the manifest's `pivot` field when writing
  Godot AtlasTexture resources — the `pivot` data set by `align` is
  meaningful to the engine, not just for visual verification.
- `DiffImage` produces a PNG artifact alongside structured JSON.
  Vision models can examine the diff overlay to verify alignment quality
  without running any code.
- The `--anchor` choice depends on character design: `feet` for
  platformer characters, `centroid` for floating objects (birds, projectiles),
  `bbox` for cases where the character silhouette changes drastically between
  frames and centroid would be unstable.
