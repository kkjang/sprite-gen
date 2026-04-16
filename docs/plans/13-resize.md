# Plan 13 — Resize for Delivery Size

Status: planned.

## Goal

Add a generic late-stage `resize` command family for integer pixel-preserving size changes that do not change the underlying art detail budget.

This plan separates three different concepts cleanly:
- `snap scale` is corrective: undo accidental integer upscaling in the source image.
- `normalize detail` is stylistic: intentionally reduce source detail toward a target project look.
- `resize` is presentational: make an already-clean image or frame set appear larger or smaller without introducing interpolation blur.

The initial command surface should cover both single images and frame sets:
- `sprite-gen resize image PATH`
- `sprite-gen resize frames DIR`

Both commands should use integer nearest-neighbor resampling only in their first version.

## Scope

**In:**
- New `resize` top-level command with `image` and `frames` subcommands
- Integer nearest-neighbor upscaling and downscaling
- One-image-in / one-image-out PNG workflow for `resize image`
- Many-images-in / many-images-out + manifest workflow for `resize frames`
- Deterministic output paths under `out/<subject>/resize/...`
- Manifest updates for frame-set resizing
- Tests for both image and frame-set resize behavior

**Out:**
- Non-integer resizing factors
- Bilinear, bicubic, Lanczos, or blur-based resampling
- Automatic factor inference from content
- Any attempt to merge `resize` into `normalize detail`
- Any attempt to replace `snap scale`

## Why this is a separate plan

Users often want two different things:

1. Fewer effective pixels in the art itself.
2. Larger final output assets on screen.

`normalize detail` solves (1).
`resize` solves (2).

Keeping them separate makes the pipeline easier to reason about for both humans and coding agents.

## File plan

```text
sprite-gen/
  cmd/sprite-gen/
    cmd_resize.go                       # flag parsing for resize image/frames
    cmd_resize_test.go
  internal/
    resize/
      resize.go                         # integer NN resize helpers
      resize_test.go
```

## Command design

### `sprite-gen resize image PATH`

Resize a single PNG for delivery or presentation size.

Flags:
- `--up N` (required unless `--down` is set): integer nearest-neighbor upscale factor
- `--down N` (required unless `--up` is set): integer nearest-neighbor downscale factor
- `--method nearest` (default `nearest`; only supported value in v1)
- `--out FILE` (default: `./out/<subject>/resize/image.png`)
- `--dry-run`
- global `--json`

Validation:
- Exactly one of `--up` or `--down` must be provided
- Factors must be integers greater than or equal to 1
- `--method` must be `nearest`
- For `--down`, the chosen factor must evenly divide the image dimensions

Behavior:
1. Load the PNG.
2. Apply nearest-neighbor upscaling or downscaling.
3. Write the resized PNG.
4. Report input/output dimensions, direction, factor, and whether the result was unchanged.

### `sprite-gen resize frames DIR`

Resize a frame set after `slice`, `segment`, or `align`.

Flags:
- `--up N` (required unless `--down` is set)
- `--down N` (required unless `--up` is set)
- `--method nearest` (default `nearest`; only supported value in v1)
- `--out DIR` (default: `./out/<subject>/resize/`)
- `--dry-run`
- global `--json`

Behavior:
1. Load the frame set from `manifest.json` when present, otherwise discover `frame_*.png`.
2. Resize every frame image using the same integer factor and method.
3. Write a new frame set.
4. If an input manifest exists, preserve each frame's source-space `rect` exactly.
5. Scale output-space fields:
   - `manifest.cell_w`, `manifest.cell_h`
   - `manifest.frames[].w`, `manifest.frames[].h`
   - `manifest.frames[].pivot` when present
6. Preserve frame order.

Why `rect` stays unchanged: it records source-space coordinates in the original sheet or canvas, not delivery-space output size.

## Internal API

### `internal/resize/resize.go`

```go
package resize

type Direction string

const (
    Up   Direction = "up"
    Down Direction = "down"
)

type Options struct {
    Direction Direction
    Factor    int
}

func Image(img *image.NRGBA, opts Options) (*image.NRGBA, error)
func Frames(imgs []*image.NRGBA, opts Options) ([]*image.NRGBA, error)
```

Implementation notes:
- Reuse or extract the existing nearest-neighbor logic already present in `internal/pixel` and GIF export.
- Keep v1 integer-only. Do not add float scaling until there is a concrete need.
- `resize` should be generic enough for both command surfaces, but no more generic than that.

## Pipeline placement

Preferred placement:

```text
prep background? -> snap scale? -> palette extract? -> snap pixels? -> normalize detail? -> segment/slice -> align -> resize frames? -> export
```

Why late:
- resizing early multiplies noise and halo cleanup work
- resizing late keeps detection/alignment thresholds stable
- one resized frame set can feed multiple export formats consistently

`gif --scale` may remain as a convenience preview knob, but `resize frames` is the generic pipeline step when multiple downstream artifacts should share the same delivery size.

## Testing

`internal/resize/resize_test.go`:
- upscale factor 2 duplicates pixels into 2x2 blocks
- downscale factor 2 halves dimensions using nearest-neighbor sampling
- invalid direction/factor combinations fail with actionable errors

Command-level tests:
- `resize image testdata/input/normalize/knight_native.png --up 2 --json`
- `resize image testdata/input/normalize/knight_native.png --down 2 --json`
- `resize frames out/subject/align --up 2 --json`
- `resize frames out/subject/align --down 2 --json`
- providing both `--up` and `--down` fails

Manifest-specific tests:
- source-space `rect` is preserved exactly after `resize frames`
- output-space `pivot` scales with the factor
- `cell_w`/`cell_h` scale with the factor

## Acceptance criteria

1. `go test ./...` passes.
2. `resize image` can enlarge or shrink a PNG by an integer factor without interpolation.
3. `resize frames` produces a valid frame set with correctly updated manifest output-space fields.
4. The README and `AGENTS.md` explain `resize` as a delivery-size stage distinct from `normalize detail` and `snap scale`.

## Suggested commit message

```text
feat(resize): add generic delivery-size resizing for images and frame sets

Add `resize image` and `resize frames` for integer nearest-neighbor
upscale/downscale so projects can change output presentation size
without conflating delivery size with source detail normalization.
```

## Notes for implementation

- Keep the command name `resize`, not `scale`, to avoid confusion with `snap scale`.
- Favor `--up` / `--down` over one overloaded factor flag; it is easier to explain to users who are new to pixel-art tooling.
- If later we need alternate methods, add them behind `--method`; do not add extra top-level verbs.
