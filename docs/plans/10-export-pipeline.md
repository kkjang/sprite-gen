# Plan 10 — Export Pipeline + Generic Formats

## Goal

Introduce the format registry and the `export` command. Ship two engine-agnostic formats (`gif` for preview, `sheet` for packed sprite sheet + metadata) to prove the registry extends cleanly before adding engine-specific formats in plan 11.

Implemented with two verified adjustments from the original draft:
- `sheet` writes two artifacts into `--out`, which is a directory: `<subject>_sheet.png` and a sibling `<subject>_sheet.json` manifest with the same shared manifest shape used elsewhere plus `sheet` and `sheet_size` fields. The JSON write is best-effort atomic via temp file + rename.
- `sheet` accepts mixed-size frame sets and pads them into max-size cells, so it is a terminal export artifact rather than a lossless trimmed-frame serialization.

## Scope

**In:**
- `internal/export` package: `Format` interface + registry, `ExportContext` struct
- `internal/export/formats/gif`: animated GIF preview
- `internal/export/formats/sheet`: pack frames back into a single sprite sheet PNG plus JSON sidecar
- `sprite-gen export DIR --format=<name>` — single convergence point for all output formats
- `sprite-gen export DIR --list-formats` — print available formats (mirrors `spec` but format-scoped)
- Tests: registry wiring, GIF output, sheet packing behavior

**Out:**
- Any engine-specific formats (plan 11)
- Upload/transfer to any external system

## File plan

```
sprite-gen/
  cmd/sprite-gen/
    cmd_export.go                       # flag parsing; dispatch to format registry
  internal/
    export/
      export.go                         # Format interface + registry
      context.go                        # ExportContext: frames, manifest, options
      formats/
        gif/
          gif.go                        # Format impl: animated GIF
          gif_test.go
        sheet/
          sheet.go                      # Format impl: packed sprite sheet
          sheet_test.go
```

## Export package design

### `internal/export/export.go`

```go
// Package export defines the Format interface and the format registry.
package export

// Format is implemented by each output format (gif, sheet,
// godot-spriteframes, godot-atlas, ...).
type Format interface {
    // Name returns the --format flag value (e.g. "gif", "sheet").
    Name() string
    // Description is shown in sprite-gen export --list-formats.
    Description() string
    // Export writes output files using the provided context and returns
    // a command-friendly summary for text/JSON output.
    Export(ctx *Context) (*Result, error)
}

type Result struct {
    Text string
    Data any
}

// Register adds a format. Called from each format's init().
func Register(f Format)

// Get returns the named format or an error.
func Get(name string) (Format, error)

// All returns all registered formats in sorted order.
func All() []Format
```

### `internal/export/context.go`

```go
// Context carries everything a Format needs to produce output.
type Context struct {
    FrameDir string
    Manifest *manifest.Manifest
    Frames []Frame
    Options map[string]string
    OutPath string
    DryRun bool
    Format string
    Subject string
    ManifestPath string
}

type Frame struct {
    Index int
    Path  string
    Rect  manifest.Rect
    Image *image.NRGBA
}
```

Extra format flags arrive as `Options` strings because the flag set is
defined globally but format-specific flags are unknown until the format
is selected. The command parses `--fps`, `--cols`, `--loop`, `--scale`
etc. as named extras and passes them through. Each format documents
which options it reads.

### `internal/export/formats/gif/gif.go`

```go
package gif

// Options read from ctx.Options:
//   fps   int (default 8)
//   scale int (default 1, up to 8 for visibility at native resolution)
//   loop  bool (default true)

type GIF struct{}

func (g GIF) Name() string        { return "gif" }
func (g GIF) Description() string { return "Animated GIF preview (for visual verification)" }
func (g GIF) Export(ctx *export.Context) error { ... }

func init() { export.Register(GIF{}) }
```

Implementation outline:
1. Load each frame image.
2. For each frame: quantize to a frame-local palette with up to 255 visible
   colors plus transparent index 0. Palette data is derived directly from the
   frame PNGs; it is not stored in `manifest.json`.
3. Build a `*gif.GIF` with per-frame delays derived from `fps`.
4. Upscale each frame by `scale` (integer NN) if scale > 1.
5. Write with `gif.EncodeAll`.

The GIF format has a 256-color limit per frame. For sprites with > 256
colors, quantize per-frame (each frame can have its own palette in GIF).
Because GIF delays are stored in centiseconds, the exporter rounds `fps` to the
nearest representable frame delay.

### `internal/export/formats/sheet/sheet.go`

```go
package sheet

// Options read from ctx.Options:
//   cols    int (default: ceil(sqrt(N)) for roughly square output)
//   padding int (default 0)

type Sheet struct{}

func (s Sheet) Name() string        { return "sheet" }
func (s Sheet) Description() string { return "Pack frames into a sprite sheet PNG plus JSON manifest" }
func (s Sheet) Export(ctx *export.Context) (*export.Result, error) { ... }

func init() { export.Register(Sheet{}) }
```

Implementation outline:
1. Load frame images. Determine max cell W and H across all frames.
2. Pack into a grid of `--cols` columns.
3. Draw each frame at its grid position using `image/draw`.
4. Write `<subject>_sheet.png` and sibling `<subject>_sheet.json` under the output directory.

The sheet manifest reuses the shared `manifest.Manifest` type. For sheet exports,
`frames[].x/y/w/h` record each frame's rectangle inside the packed sheet PNG, and
optional `frames[].duration_ms` / `frames[].tag` fields are preserved when they
already exist on the input frame-set manifest.
The internal Go model still uses `manifest.Rect`; only the JSON schema is flat.

When input frames are mixed-size, `sheet` pads them into max-size cells.
That preserves visible pixels and frame order, but does not promise exact
trim-preserving round-trip behavior through `slice grid`.

## Command design

### `sprite-gen export DIR --format=<name>`

Flags:
- `--format NAME` (required): registered format name
- `--out PATH` (default: `./out/<subject>/export/`; export formats write their artifacts into this output directory)
- `--dry-run`
- `--fps N` (default 8): passed to gif format
- `--cols N`: passed to sheet format
- `--scale N` (default 1): pixel upscale for gif preview
- `--loop` (default true): for gif
- `--padding N` (default 0): for sheet
- global `--json`

Behavior:
1. Resolve frame dir: if DIR contains `manifest.json`, load it.
   Otherwise glob `frame_*.png` sorted.
2. Look up format by name. Error if unknown, suggest `--list-formats`.
3. Build `ExportContext` and call `format.Export(ctx)`.

Text output (varies by format, shown for gif):

```
wrote: out/walk_4x1/export/walk_4x1_preview.gif (4 frames, 8 fps target, 520ms total)
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "format": "gif",
    "out": "out/walk_4x1/export",
    "gif": "out/walk_4x1/export/walk_4x1_preview.gif",
    "frames": 4,
    "fps": 8,
    "frame_delay_cs": 13,
    "duration_ms": 520,
    "scale": 1,
    "loop": true,
    "dry_run": false
  }
}
```

### `sprite-gen export DIR --list-formats`

Prints available formats without running an export:

```
gif         Animated GIF preview (for visual verification)
sheet       Pack frames into a sprite sheet PNG plus JSON manifest
```

With `--json`:

```json
{
  "ok": true,
  "data": {
      "formats": [
        {"name": "gif", "description": "Animated GIF preview (for visual verification)"},
        {"name": "sheet", "description": "Pack frames into a sprite sheet PNG plus JSON manifest"}
      ]
    }
  }
```

## Testing

`internal/export` registry tests:
- `Register` + `Get` round-trip.
- Duplicate `Register` panics.
- `All` returns formats in sorted order.
- `Get` on unknown name returns an error naming available formats.

`internal/export/formats/gif`:
- Exporting a 4-frame set at 8fps produces a GIF with 4 frames and 13cs delay.
- GIF frame count matches input frame count.
- `--scale 2` on a 32x32 frame set produces 64x64 GIF frames.
- `--dry-run` produces no file.

`internal/export/formats/sheet`:
- Writes `hero_sheet.png` and `hero_sheet.json` into the chosen output directory.
- Sheet manifest has `version: 1`, `sheet: "hero_sheet.png"`, and `sheet_size` matching the PNG bounds.
- Frame order is preserved and each `frames[].x/y/w/h` rectangle falls within `sheet_size`.
- Optional `frames[].duration_ms` / `frames[].tag` fields round-trip when present and are omitted when absent.
- Mixed-size inputs are padded into max-size cells and reported as `mixed_sizes: true`.
- `--padding 2` on a 2x2 10x10 frame set produces a 22x22 sheet.
- `--dry-run` produces no file.

Command-level tests:
- `export DIR --format gif --json` → envelope with `out` ending in `/export` and `gif` ending in `.gif`.
- `export DIR --format sheet --cols 4 --json` → envelope with `out` ending in `/export`, plus `png` and `manifest` paths.
- `export dir --format unknown` → exit non-zero, error lists valid formats.
- `export dir --list-formats --json` → envelope with `formats` array.

## Acceptance criteria

1. `go test ./...` passes.
2. Full chain:
   ```bash
   sprite-gen slice grid testdata/input/slice/walk_4x1.png --cols 4
   sprite-gen export out/walk_4x1/slice --format gif --fps 8 --scale 2
   sprite-gen export out/walk_4x1/slice --format sheet --cols 4
   ```
   All three commands exit 0.
3. The GIF is viewable (valid GIF89a header, correct frame count).
4. `sheet` writes `<subject>_sheet.png` and `<subject>_sheet.json` into `--out`.
5. `sprite-gen export --list-formats` shows exactly two formats.
6. `sprite-gen spec` shows sixteen commands.
7. No new non-stdlib dependencies (stdlib `image/gif` handles GIF encoding).

## Suggested commit message

```
 feat(export): format registry + gif and sheet exporters

Add internal/export with Format interface and init()-based registry.
Single `export DIR --format=<name>` command replaces the previous
compose verbs. Ship two engine-agnostic formats: gif (animated preview)
and sheet (repack plus JSON metadata). Plan 11 adds engine-specific formats.
```

## Notes for the next plan

- Plan 11 adds `godot-spriteframes` and `godot-atlas` formats. They
  self-register in `init()` and require zero changes to `cmd_export.go`
  or `internal/export/export.go` — the registry is the extension point.
- The GIF quantization in this plan is per-frame (each frame has its own
  paletted image with transparent index 0). If visual quality is poor on
  frames that share many colors, a global palette mode can be added later
  without changing the interface.
- `ExportContext.Options` is a `map[string]string` on purpose: it avoids
  a combinatorial flag interface while keeping format-specific options
  self-documenting within each format's source file. If a format needs a
  typed option that can't be parsed from a string, it returns an error
  early in `Export`.
