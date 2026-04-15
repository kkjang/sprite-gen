# Plan 05 — Pixel Snap

## Goal

Complete the "clean up a single PNG" story with two snap operations: remove anti-aliasing by palette-snapping fractional-alpha pixels, and reverse accidental upscaling by downsampling to native resolution. Together with plan 04 (palette), this gives an agent the full single-image cleanup chain.

## Scope

**In:**
- `internal/pixel` extended: alpha-threshold pass, integer nearest-neighbor downscale
- `sprite-gen snap pixels PATH` — remove AA (threshold alpha, then palette snap)
- `sprite-gen snap scale PATH` — detect and undo accidental 2×/3×/4× upscaling
- Tests with golden PNG fixtures

**Out:**
- Any slicing, alignment, or multi-frame logic
- Sub-pixel or Lanczos resampling (nearest-neighbor only)
- Dithering during snap (that's `palette apply --dither`, not snap pixels)

## File plan

```
sprite-gen/
  cmd/sprite-gen/
    cmd_snap.go                         # flag parsing for pixels + scale
  internal/
    pixel/
      alpha.go                          # ThresholdAlpha, CountFractional
      scale.go                          # DetectScale, Downscale (integer NN)
      alpha_test.go
      scale_test.go
  testdata/
    input/
      snap/
        aa_knight.png                   # knight with AA edges (fractional alpha)
        upscaled_4x.png                 # 128x128 that is actually 32x32 @ 4x
        upscaled_2x.png                 # 64x64 that is actually 32x32 @ 2x
    golden/
      snap/
        aa_knight_snapped.png
        upscaled_4x_scaled.png          # expected 32x32
        upscaled_2x_scaled.png          # expected 32x32
```

## Pixel package extensions

### `alpha.go`

```go
// ThresholdAlpha converts all pixels with alpha below lo to fully
// transparent and all pixels with alpha above hi to fully opaque.
// Pixels between lo and hi (fractional-alpha / AA pixels) are set
// transparent when snap is false, or left for the caller to snap
// when snap is true.
//
// For AA removal: call with lo=0, hi=128; fractional pixels → transparent.
// For hard cutoff: call with lo=hi=128; below → 0, above → 255.
func ThresholdAlpha(img *image.NRGBA, lo, hi uint8) *image.NRGBA

// CountFractional returns the number of pixels with alpha strictly
// between 0 and 255.
func CountFractional(img image.Image) int
```

### `scale.go`

```go
// DetectScale guesses the integer upscale factor of img. It looks for
// the largest N ∈ {2,3,4,8} such that every N×N block of pixels is
// uniform in color. Returns 1 if no upscaling is detected.
func DetectScale(img image.Image) int

// Downscale reduces img by an integer factor using nearest-neighbor
// sampling (take the top-left pixel of each factor×factor block).
// Panics if factor < 1.
func Downscale(img *image.NRGBA, factor int) *image.NRGBA
```

`DetectScale` algorithm:
1. For each candidate factor F in {8, 4, 3, 2} (largest first):
   - Sample a grid of W/F × H/F blocks.
   - For each block, check whether all F×F pixels share the same NRGBA value.
   - If ≥ 95% of blocks are uniform, return F.
2. Return 1 (no detected upscaling).

The 95% threshold handles the edge case where a character's outline pixels
straddle block boundaries after a non-pixel-aligned crop.

## Command design

### `sprite-gen snap pixels PATH`

Removes AA from a sprite by:
1. Loading the image.
2. Running `pixel.ThresholdAlpha(img, 0, alphaThreshold)` to zero out
   fractional-alpha pixels (they're AA; we make them transparent).
3. Running `palette.Apply(img, pal, false)` to snap remaining opaque
   pixels to the nearest palette color.
4. Saving the result.

Flags:
- `--palette FILE` (required): .hex or .gpl file to snap to
- `--alpha-threshold N` (default 128): pixels with alpha < N become transparent
- `--out FILE` (default: `./out/snap/<stem>_snapped.png`)
- `--dry-run`
- global `--json`

Text output:

```
wrote: out/snap/aa_knight_snapped.png
fractional_pixels_zeroed: 342
changed_pixels: 187
palette_size: 16
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "out": "out/snap/aa_knight_snapped.png",
    "fractional_pixels_zeroed": 342,
    "changed_pixels": 187,
    "palette_size": 16,
    "alpha_threshold": 128,
    "dry_run": false
  }
}
```

### `sprite-gen snap scale PATH`

Flags:
- `--factor auto|N` (default `auto`): force a specific factor or detect
- `--out FILE` (default: `./out/snap/<stem>_native.png`)
- `--dry-run`
- global `--json`

Behavior:
- When `--factor auto`: call `pixel.DetectScale(img)`.
- When `--factor N`: use N directly (skip detection, trust the caller).
- If factor is 1: write a warning and still exit 0 (idempotent).
- Call `pixel.Downscale(img, factor)`.
- Save result.

Text output:

```
wrote: out/snap/upscaled_4x_native.png
detected_factor: 4
in:  128x128
out: 32x32
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "out": "out/snap/upscaled_4x_native.png",
    "detected_factor": 4,
    "forced_factor": null,
    "in_w": 128, "in_h": 128,
    "out_w": 32, "out_h": 32,
    "dry_run": false
  }
}
```

## Testing

`internal/pixel/alpha_test.go`:
- `ThresholdAlpha` with lo=0, hi=128 on a 1x1 pixel with alpha=64 → alpha becomes 0.
- `ThresholdAlpha` on a 1x1 pixel with alpha=200 → unchanged.
- `CountFractional` on an image with 5 AA pixels returns 5.
- Golden: `ThresholdAlpha(aa_knight.png, 0, 128)` matches `golden/snap/aa_knight_thresh.png`.

`internal/pixel/scale_test.go`:
- `DetectScale` on a 1x1 image returns 1.
- `DetectScale` on a synthetically 4x-upscaled image returns 4.
- `DetectScale` on a natural photo (random pixels) returns 1.
- `Downscale(img, 1)` is identity.
- `Downscale(img, 4)` on a 128x128 uniform-block image produces the expected 32x32.
- `Downscale` followed by `pixel.LoadPNG` on a written file round-trips dimensions.
- Golden: `Downscale(upscaled_4x.png, 4)` matches `golden/snap/upscaled_4x_scaled.png`.

Command-level tests:
- `snap scale testdata/input/snap/upscaled_4x.png --json` → envelope with `detected_factor: 4`, `out_w: 32`.
- `snap scale testdata/input/snap/upscaled_4x.png --factor 4 --json` → same result, `forced_factor: 4`.
- `snap pixels testdata/input/snap/aa_knight.png --palette golden/palette/knight_16.hex --json` → envelope with `ok: true`.
- `snap pixels` without `--palette` → exit code 2, stderr mentions `--palette`.
- `snap scale` on a non-PNG → exit code non-zero, actionable error.

## Acceptance criteria

1. `go test ./...` passes including golden comparisons.
2. Running the typical cleanup chain on test fixtures:
   ```bash
   sprite-gen snap scale testdata/input/snap/upscaled_4x.png --factor auto
   sprite-gen palette extract out/snap/upscaled_4x_native.png --max 16 > /tmp/p.hex
   sprite-gen snap pixels out/snap/upscaled_4x_native.png --palette /tmp/p.hex
   ```
   exits 0 at each step and produces images that differ from the inputs.
3. `sprite-gen snap scale` on an already-native image (factor=1) exits 0 with a note.
4. `sprite-gen spec` shows eight commands.
5. No new non-stdlib dependencies (go-quantize was added in plan 04; `scale.go` uses only stdlib `image`).

## Suggested commit message

```
feat(snap): pixel AA removal and integer scale reversal

Extend internal/pixel with alpha thresholding and integer
nearest-neighbor downscaling. Two new verbs: `snap pixels`
(threshold + palette snap) and `snap scale` (detect/undo 2x-8x
upscaling). Completes the single-image cleanup chain.
```

## Notes for the next plan

- `snap pixels` and `palette apply` are closely related but distinct:
  `snap pixels` does an alpha pre-pass; `palette apply` does not.
  Do not merge them — callers who only want color quantization without
  AA removal need `palette apply` directly.
- `DetectScale` is heuristic and can misfire on certain inputs (e.g.,
  a sprite with large uniform color blocks that aren't from upscaling).
  `--factor auto` is a hint; `--factor N` is the escape hatch.
- Plan 06 (`slice`) will need `pixel.Downscale` indirectly if we ever
  want to export scaled-down frames. Do not expose it publicly outside
  `internal/pixel` for now.
