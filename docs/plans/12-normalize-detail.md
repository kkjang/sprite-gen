# Plan 12 — Normalize Detail

Status: implemented.

## Goal

Add `sprite-gen normalize detail PATH` as an intentional style-normalization step for single PNG inputs. The command should help a project converge on a consistent level of geometric detail by scaling a sprite toward a target visible subject height using integer nearest-neighbor resampling.

This is distinct from `snap scale`:
- `snap scale` is corrective: detect and undo accidental integer upscaling.
- `normalize detail` is intentional: make native-resolution assets feel closer to the same project detail budget.

## Scope

**In:**
- New `normalize` top-level command with `detail` subcommand
- One-image-in / one-image-out PNG workflow
- Integer factor selection from either an explicit `--factor` or a target visible height
- Bounding-box-based measurement using an alpha threshold
- Deterministic output path under `out/<subject>/normalize/detail.png`
- Tests for factor selection, bbox measurement, and PNG output

**Out:**
- Any project-wide batch command
- Any slicing, alignment, or multi-frame normalization logic
- Palette extraction or palette snapping inside `normalize detail` itself
- Non-integer scaling, interpolation, or blur-based resampling
- Automatic inference of a project-wide target from a corpus of sprites

## File plan

```text
sprite-gen/
  cmd/sprite-gen/
    cmd_normalize.go                    # flag parsing for normalize detail
  internal/
    detail/
      normalize.go                      # factor selection + bbox-target normalization
      normalize_test.go
    pixel/
      scale.go                          # reuse Downscale; extend tests if needed
  testdata/
    input/
      normalize/
        lantern_walk.png
        knight_native.png
    golden/
      normalize/
        lantern_walk_h48.png
        knight_native_h48.png
```

## Command design

### `sprite-gen normalize detail PATH`

Normalize a single PNG toward a target geometric detail level.

Flags:
- `--target-height N` (required unless `--factor` is set): desired visible subject height in pixels
- `--factor N` (required unless `--target-height` is set): explicit integer downscale factor
- `--alpha-threshold N` (default `8`): minimum alpha used when measuring the visible bbox
- `--out FILE` (default: `./out/<subject>/normalize/detail.png`)
- `--dry-run`
- global `--json`

Validation:
- Exactly one of `--target-height` or `--factor` must be provided
- `--target-height` must be greater than 0
- `--factor` must be an integer greater than or equal to 1
- Chosen factor must evenly divide the image dimensions; otherwise fail with an actionable error

Behavior:
1. Load the PNG.
2. Measure the visible bbox with `pixel.BBox(img, alphaThreshold-1)`.
3. Determine the factor:
   - If `--factor N`: trust the caller.
   - If `--target-height N`: consider integer factors that evenly divide both image dimensions and choose the factor whose output bbox height is closest to `N`.
4. Downscale the full image with nearest-neighbor sampling.
5. Write the normalized PNG.
6. Report input/output dimensions, bbox heights, chosen factor, and whether the result was unchanged.

The command intentionally does not palette-snap. Callers can compose it with:
- `palette extract`
- `snap pixels`
- `palette apply`

## Internal API

### `internal/detail/normalize.go`

```go
package detail

type Options struct {
    TargetHeight   int
    Factor         int
    AlphaThreshold uint8
}

type Result struct {
    Factor         int
    InputW         int
    InputH         int
    OutputW        int
    OutputH        int
    InputBBoxH     int
    OutputBBoxH    int
    Unchanged      bool
    Image          *image.NRGBA
}

// Normalize scales img toward the requested detail target.
// Exactly one of opts.TargetHeight or opts.Factor must be set.
func Normalize(img *image.NRGBA, opts Options) (*Result, error)
```

Implementation notes:
- Reuse `pixel.Downscale` for the actual resampling.
- Keep factor selection inside `internal/detail` rather than expanding `cmd_normalize.go`.
- Prefer the smallest correct surface area: no new general-purpose resizing package unless a second caller appears.

## Output shape

Text output:

```text
wrote: out/lantern_walk/normalize/detail.png
factor: 2
input: 1024x1024
output: 512x512
input_bbox_h: 180
output_bbox_h: 90
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "out": "out/lantern_walk/normalize/detail.png",
    "factor": 2,
    "input_w": 1024,
    "input_h": 1024,
    "output_w": 512,
    "output_h": 512,
    "input_bbox_h": 180,
    "output_bbox_h": 90,
    "alpha_threshold": 8,
    "unchanged": false,
    "dry_run": false
  }
}
```

## Testing

`internal/detail/normalize_test.go`:
- `--factor 1` leaves the image unchanged
- explicit `factor=2` halves width and height
- `target-height` chooses the closest valid integer factor
- invalid option combinations return actionable errors
- empty/fully transparent image returns an actionable error

Command-level tests:
- `normalize detail testdata/input/normalize/lantern_walk.png --factor 2 --json`
- `normalize detail testdata/input/normalize/lantern_walk.png --target-height 48 --json`
- missing both `--factor` and `--target-height` fails
- providing both `--factor` and `--target-height` fails
- non-divisible forced factor fails with a clear error

Golden tests:
- verify deterministic PNG output for a representative asset normalized to a fixed target height

## Acceptance criteria

1. `go test ./...` passes.
2. `sprite-gen normalize detail PATH --factor 2` writes a visibly chunkier PNG with deterministic dimensions.
3. `sprite-gen normalize detail PATH --target-height N` chooses a stable integer factor and reports it in text and JSON output.
4. `sprite-gen spec` includes `normalize detail`.
5. The README and `AGENTS.md` document `normalize detail` as an optional project-consistency step, not a required prerequisite.

## Suggested commit message

```text
feat(normalize): add project detail normalization for PNG sprites

Add `normalize detail` to intentionally reduce geometric detail
toward a target visible height or explicit integer factor. This
complements `snap scale` by making style normalization explicit
instead of overloading accidental upscale recovery.
```

## Notes for the next plan

- Keep `snap scale --factor` even after this lands. It remains the explicit override for corrective scale reversal.
- `normalize detail` should stay single-image and intentional. Do not turn it into a batch workflow yet.
- If nearest-neighbor downscale proves too blunt for some assets, add a later enhancement with alternate block-reduction methods such as `mode`, not as part of this initial plan.
