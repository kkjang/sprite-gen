# Plan 05 — Palette Ops

## Goal

Introduce the `internal/palette` package with two commands: extract a palette from an image and apply an existing palette to an image. These are standalone-useful and are prerequisites for plan 06 (`snap pixels`), which needs a target palette to snap to.

## Scope

**In:**
- `internal/palette` package: deterministic palette extraction, palette file I/O (hex + GPL), pixel-distance snapping
- `sprite-gen palette extract PATH` — output a palette file from an image's dominant colors
- `sprite-gen palette apply PATH --palette FILE` — recolor an image to a given palette
- Optional Floyd-Steinberg dithering when applying a palette
- Tests with golden palette files

**Out:**
- Anti-aliasing removal (plan 06)
- Integer rescaling (plan 06)
- Dithering modes beyond none and Floyd-Steinberg

## File plan

```
sprite-gen/
  cmd/sprite-gen/
    cmd_palette.go                      # flag parsing for extract + apply
  internal/
    palette/
      quantize.go                       # deterministic palette extraction; return []color.NRGBA
      snap.go                           # nearest-color snapping (no AA removal here)
      file.go                           # read/write .hex and .gpl palette files
      quantize_test.go
      snap_test.go
      file_test.go
  testdata/
    input/
      palette/
        knight_16color.png              # small PNG with ~16 distinct colors
        simple_4color.png               # 4 flat colors, no alpha
    golden/
      palette/
        knight_16.hex                   # expected .hex output for knight_16color.png
        simple_4.hex
        simple_4.gpl
        knight_applied.png              # knight_16color.png recolored to its own palette
```

## Palette package design

```go
// Package palette provides color extraction, palette file I/O,
// and nearest-color pixel snapping.
package palette

import "image/color"

// Extract returns up to maxColors dominant colors from img using
// a deterministic in-repo quantizer.
func Extract(img image.Image, maxColors int) []color.NRGBA

// Snap returns the color in pal closest to c by Euclidean distance
// in NRGBA space (ignores alpha channel for distance).
func Snap(c color.NRGBA, pal []color.NRGBA) color.NRGBA

// Apply returns a new *image.NRGBA with every opaque pixel snapped to
// the nearest color in pal. Transparent pixels (alpha == 0) are left
// transparent. Partially-transparent pixels are snapped and their alpha
// is preserved unchanged (AA removal is a separate concern in plan 06).
func Apply(img image.Image, pal []color.NRGBA, dither bool) *image.NRGBA

// --- File I/O ---

// ReadHex parses a .hex palette file: one "#rrggbb" per line, comments
// with '#' only if they come after whitespace (rare; just skip blank
// and '#'-prefixed lines gracefully).
func ReadHex(r io.Reader) ([]color.NRGBA, error)

// WriteHex writes colors as "#rrggbb\n" lines (lowercase hex).
func WriteHex(w io.Writer, pal []color.NRGBA) error

// ReadGPL parses a GIMP Palette (.gpl) file; extracts only the
// "R G B name" rows and ignores the header.
func ReadGPL(r io.Reader) ([]color.NRGBA, error)

// WriteGPL writes a valid .gpl file. Name field is the hex string.
func WriteGPL(w io.Writer, name string, pal []color.NRGBA) error
```

One exported `Snap` for a single pixel, one exported `Apply` that loops
the whole image. This separation makes `snap pixels` (plan 06) easy: it
calls `Apply` but first removes fractional-alpha pixels via a separate
alpha-threshold pass.

## Command design

### `sprite-gen palette extract PATH`

Flags:
- `--max N` (default 16): maximum colors in the output palette
- `--format hex|gpl` (default `hex`): output format
- `--out FILE` (default: stdout for hex/gpl text)
- `--dry-run`: validate and report what would be written when `--out` is set
- global `--json`

Behavior:
- Load image with `pixel.LoadPNG`
- Call `palette.Extract(img, max)`
- Write to `--out` or stdout

Text output (to stdout, default):

```
#1a1c2c
#5d275d
#b13e53
...
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "colors": ["#1a1c2c", "#5d275d", "#b13e53"],
    "count": 16,
    "format": "hex",
    "out": "-"
  }
}
```

When `--out FILE` is given, `data.out` is the file path and nothing is
written to stdout. This allows piping: `sprite-gen palette extract img.png > palette.hex`.
If `--dry-run` is set, no file is written and the response reports the
target path.

### `sprite-gen palette apply PATH`

Flags:
- `--palette FILE` (required): .hex or .gpl palette file
- `--dither` (default false): enable Floyd-Steinberg dithering
- `--out FILE` (default: `./out/palette/<stem>/applied.png`)
- `--dry-run`: print what would be written, don't write
- global `--json`

Behavior:
- Load image with `pixel.LoadPNG`
- Detect palette file format by extension (`.hex` or `.gpl`); error if unknown
- Call `palette.Apply(img, pal, dither)`
- Save result with `pixel.SavePNG`

Text output:

```
  wrote: out/palette/knight/applied.png
colors in: 847
colors out: 16
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "out": "out/palette/knight/applied.png",
    "colors_in": 847,
    "colors_out": 16,
    "dither": false,
    "dry_run": false
  }
}
```

## Deterministic output paths

Default output path pattern: `./out/palette/<stem>/applied.png` where
`<stem>` is the input filename without extension.

Example: `knight_16color.png` → `out/palette/knight_16color/applied.png`.

This is the same `./out/<subject>/<stem>/...` convention from the overview.

## Quantization approach

Use a small deterministic in-repo quantizer implemented in
`internal/palette/quantize.go`. `image/draw` provides the `draw.Quantizer`
interface and Floyd-Steinberg dithering, but it does not provide a
built-in palette extraction implementation we can instantiate directly.
Keep extraction internal so we can swap algorithms later without touching
the command surface.

## Testing

`internal/palette/quantize_test.go`:
- `Extract` on a 4-color image returns exactly 4 colors (or fewer if `max < 4`).
- `Extract` with `max=16` on a gradient image returns ≤ 16 colors.
- Colors returned are stable (same input → same output); verify this
  property explicitly.

`internal/palette/snap_test.go`:
- `Snap(red, [red, blue, green])` returns red.
- `Snap(color close to blue, [red, blue, green])` returns blue.
- `Apply` on an image already using only palette colors is idempotent.
- `Apply` with `dither=false` produces no new colors outside the palette.

`internal/palette/file_test.go`:
- `WriteHex` → `ReadHex` round-trip preserves all colors.
- `WriteGPL` → `ReadGPL` round-trip preserves all colors.
- `ReadHex` on malformed input returns a useful error.
- Golden file test: `Extract(knight_16color.png, 16)` → `WriteHex` → compare to `golden/palette/knight_16.hex`.

Command-level tests:
- `palette extract testdata/input/palette/simple_4color.png --max 4 --json` → envelope with 4 colors.
- `palette apply testdata/input/palette/knight_16color.png --palette testdata/golden/palette/knight_16.hex --json` → envelope with `ok: true`, `out` path populated.
- `palette apply` without `--palette` → non-zero exit code, stderr mentions `--palette`.

## Acceptance criteria

1. `go test ./...` passes.
2. `sprite-gen palette extract testdata/input/palette/simple_4color.png --max 4` prints 4 hex lines.
3. `sprite-gen palette apply testdata/input/palette/knight_16color.png --palette palette.hex --dry-run` prints what it would write and exits 0.
4. The applied image passes a round-trip: `sprite-gen palette apply img.png --palette p.hex && sprite-gen palette extract applied.png --max 16` produces a subset of the original palette.
5. `sprite-gen spec` shows six commands.
6. No new third-party packages are added.

## Suggested commit message

```
feat(palette): extract and apply palette operations

Add internal/palette with deterministic palette extraction,
nearest-color snap, and hex/GPL file I/O. Two new verbs:
`palette extract` and `palette apply`.
```

## Notes for the next plan

- Plan 06 (`snap pixels`) will call `palette.Apply` but adds a
  pre-pass that thresholds fractional-alpha pixels to 0 or 255 before
  snapping. That pre-pass belongs in `internal/pixel` (alpha ops),
  not in `internal/palette`.
- `snap scale` (also plan 06) uses only `internal/pixel` and does not
  need `internal/palette` at all — keep them separate.
- Do not add dithering modes beyond Floyd-Steinberg in this plan. The
  `--dither` flag is boolean for now; if named modes are needed later,
  the flag signature changes to `--dither=none|fs`.
