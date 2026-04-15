# Plan 08 — Export Pipeline + Generic Formats

## Goal

Introduce the format registry and the `export` command. Ship two engine-agnostic formats (`gif` for preview, `sheet-png` for packed sprite sheet) to prove the registry extends cleanly before adding engine-specific formats in plan 09.

## Scope

**In:**
- `internal/export` package: `Format` interface + registry, `ExportContext` struct
- `internal/export/formats/gif`: animated GIF preview
- `internal/export/formats/sheetpng`: pack frames back into a sprite sheet + manifest
- `sprite-gen export DIR --format=<name>` — single convergence point for all output formats
- `sprite-gen export DIR --list-formats` — print available formats (mirrors `spec` but format-scoped)
- Tests: registry wiring, GIF output, sheet-png round-trip

**Out:**
- Any engine-specific formats (plan 09)
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
        sheetpng/
          sheetpng.go                   # Format impl: packed sprite sheet
          sheetpng_test.go
  testdata/
    golden/
      export/
        walk_preview.gif
        walk_sheet.png
        walk_sheet_manifest.json
```

## Export package design

### `internal/export/export.go`

```go
// Package export defines the Format interface and the format registry.
package export

// Format is implemented by each output format (gif, sheet-png,
// godot-spriteframes, godot-atlas, ...).
type Format interface {
    // Name returns the --format flag value (e.g. "gif", "sheet-png").
    Name() string
    // Description is shown in sprite-gen export --list-formats.
    Description() string
    // Export writes output files using the provided context.
    Export(ctx *Context) error
}

var registry = map[string]Format{}

// Register adds a format. Called from each format's init().
func Register(f Format) {
    if _, exists := registry[f.Name()]; exists {
        panic("export: duplicate format: " + f.Name())
    }
    registry[f.Name()] = f
}

// Get returns the named format or an error.
func Get(name string) (Format, error)

// All returns all registered formats in sorted order.
func All() []Format
```

### `internal/export/context.go`

```go
// Context carries everything a Format needs to produce output.
type Context struct {
    // FrameDir is the source directory of frame PNGs.
    FrameDir string
    // Manifest is pre-loaded (may be nil if no manifest.json exists;
    // formats should handle both cases).
    Manifest *manifest.Manifest
    // Frames is the ordered list of frame image paths.
    Frames []string
    // Options contains format-specific key-value pairs parsed from
    // extra flags (--fps, --cols, --loop, etc.)
    Options map[string]string
    // OutPath is the resolved output path (file or dir, format decides).
    OutPath string
    // DryRun when true means write nothing.
    DryRun bool
    // Stdout/Stderr for progress messages.
    Stdout io.Writer
    Stderr io.Writer
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
2. For each frame: quantize to ≤ 256 colors (stdlib `image/gif` requires
   a paletted image). Use the palette from the manifest if present;
   otherwise quantize with `palette.Extract`.
3. Build a `*gif.GIF` with per-frame delays derived from `fps`.
4. Upscale each frame by `scale` (integer NN) if scale > 1.
5. Write with `gif.EncodeAll`.

The GIF format has a 256-color limit per frame. For sprites with > 256
colors, quantize per-frame (each frame can have its own palette in GIF).
This is the default behavior of `gif.EncodeAll`.

### `internal/export/formats/sheetpng/sheetpng.go`

```go
package sheetpng

// Options read from ctx.Options:
//   cols    int (default: ceil(sqrt(N)) for roughly square output)
//   padding int (default 0)

type SheetPNG struct{}

func (s SheetPNG) Name() string        { return "sheet-png" }
func (s SheetPNG) Description() string { return "Pack frames into a sprite sheet PNG + manifest" }
func (s SheetPNG) Export(ctx *export.Context) error { ... }

func init() { export.Register(SheetPNG{}) }
```

Implementation outline:
1. Load frame images. Determine max cell W and H across all frames.
2. Pack into a grid of `--cols` columns.
3. Draw each frame at its grid position using `image/draw`.
4. Write the sheet PNG.
5. Write a `manifest.json` alongside it with `Rect` fields pointing into
   the sheet (so `slice grid` can recover the original frames from it).

The round-trip `slice grid → export sheet-png → slice grid` should
produce pixel-identical frames to the originals (within rounding of
cell padding).

## Command design

### `sprite-gen export DIR --format=<name>`

Flags:
- `--format NAME` (required): registered format name
- `--out PATH` (default: format-specific, usually `./out/export/<stem>.<ext>`)
- `--dry-run`
- `--fps N` (default 8): passed to gif format
- `--cols N`: passed to sheet-png format
- `--scale N` (default 1): pixel upscale for gif preview
- `--loop` (default true): for gif
- global `--json`

Behavior:
1. Resolve frame dir: if DIR contains `manifest.json`, load it.
   Otherwise glob `frame_*.png` sorted.
2. Look up format by name. Error if unknown, suggest `--list-formats`.
3. Build `ExportContext` and call `format.Export(ctx)`.

Text output (varies by format, shown for gif):

```
wrote: out/export/walk_preview.gif (4 frames, 8fps, 500ms total)
```

JSON output:

```json
{
  "ok": true,
  "data": {
    "format": "gif",
    "out": "out/export/walk_preview.gif",
    "frames": 4,
    "fps": 8,
    "duration_ms": 500,
    "dry_run": false
  }
}
```

### `sprite-gen export DIR --list-formats`

Prints available formats without running an export:

```
gif         Animated GIF preview (for visual verification)
sheet-png   Pack frames into a sprite sheet PNG + manifest
```

With `--json`:

```json
{
  "ok": true,
  "data": {
    "formats": [
      {"name": "gif", "description": "Animated GIF preview (for visual verification)"},
      {"name": "sheet-png", "description": "Pack frames into a sprite sheet PNG + manifest"}
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
- Exporting a 4-frame set at 8fps produces a GIF with 4 frames and 125ms delay.
- GIF frame count matches input frame count.
- Golden: export of `walk_4x1/` frames matches `golden/export/walk_preview.gif`.
- `--scale 2` on a 32x32 frame set produces 64x64 GIF frames.
- `--dry-run` produces no file.

`internal/export/formats/sheetpng`:
- Round-trip: `sheetpng.Export` → `slice grid` produces pixel-identical frames.
- Sheet dimensions: 4 frames, 32x32 each, `--cols 4` → sheet 128x32.
- Golden: sheet PNG + manifest match golden fixtures.
- `--dry-run` produces no file.

Command-level tests:
- `export testdata/golden/slice/walk_4x1 --format gif --json` → envelope with `frames: 4`.
- `export testdata/golden/slice/walk_4x1 --format sheet-png --cols 4 --json` → envelope with `out` ending in `.png`.
- `export dir --format unknown` → exit non-zero, error lists valid formats.
- `export dir --list-formats --json` → envelope with `formats` array.

## Acceptance criteria

1. `go test ./...` passes including golden comparisons.
2. Full chain:
   ```bash
   sprite-gen slice grid testdata/input/slice/walk_4x1.png --cols 4
   sprite-gen export out/slice/walk_4x1 --format gif --fps 8 --scale 2
   sprite-gen export out/slice/walk_4x1 --format sheet-png --cols 4
   ```
   All three commands exit 0.
3. The GIF is viewable (valid GIF89a header, correct frame count).
4. The sheet-png export followed by `slice grid` round-trips to pixel-identical frames.
5. `sprite-gen export --list-formats` shows exactly two formats.
6. `sprite-gen spec` shows fourteen commands.
7. No new non-stdlib dependencies (stdlib `image/gif` handles GIF encoding).

## Suggested commit message

```
feat(export): format registry + gif and sheet-png exporters

Add internal/export with Format interface and init()-based registry.
Single `export DIR --format=<name>` command replaces the previous
compose verbs. Ship two engine-agnostic formats: gif (animated preview)
and sheet-png (repack). Plan 09 adds engine-specific formats.
```

## Notes for the next plan

- Plan 09 adds `godot-spriteframes` and `godot-atlas` formats. They
  self-register in `init()` and require zero changes to `cmd_export.go`
  or `internal/export/export.go` — the registry is the extension point.
- The GIF quantization in this plan is per-frame (each frame has its own
  256-color palette). If visual quality is poor on frames that share many
  colors, a global palette mode can be added later without changing the
  interface.
- `ExportContext.Options` is a `map[string]string` on purpose: it avoids
  a combinatorial flag interface while keeping format-specific options
  self-documenting within each format's source file. If a format needs a
  typed option that can't be parsed from a string, it returns an error
  early in `Export`.
