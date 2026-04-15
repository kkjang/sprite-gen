# sprite-gen Implementation Plans — Overview

A standalone Go CLI for cleaning up AI-generated pixel art and exporting it to game-engine-native formats. Intended for use by coding agents as part of a pipeline where ChatGPT/gpt-image-1 generates raw pixel art and this tool fixes predictable flaws before importing into a game project.

## Design principles

- **Engine-agnostic core.** Image processing (slice, snap, palette, align, diff) has no engine dependencies and works for Godot, Unity, Unreal, web, anywhere.
- **Pluggable export formats.** A single `export` command with a `--format=<name>` flag dispatches to a format registry. Godot is the first target; Unity, Aseprite JSON, TexturePacker, etc. drop in later without touching the rest of the code.
- **Agent ergonomics first.** Stable `{ok, data, error}` JSON envelope on every command. `sprite-gen spec` discovers the whole command surface. Idempotent writes with deterministic paths. Visual artifacts (GIFs, diff PNGs) alongside structured data so vision models can verify.
- **Minimum viable increments.** Each plan below is one reviewable PR. The first plan ships a skeleton that does almost nothing but proves the architecture. CI comes second. Features stack on top.

## Command hierarchy

Commands are grouped by **what they read** and **what they produce**:

| Read | Produce | Commands |
|---|---|---|
| Nothing | Metadata | `version`, `spec` |
| One image | Report (no files) | `inspect sheet`, `inspect frame` |
| One image | One image | `snap pixels`, `snap scale`, `palette apply` |
| One image | Palette file | `palette extract` |
| One image | Many images + manifest | `slice grid`, `slice auto` |
| Many images | Many images + manifest | `align frames` |
| Two images | Report + diff image | `diff frames` |
| Many images + manifest | One artifact (any format) | `export` |

The `export` command is the single convergence point for all output formats. Everything else is pure preprocessing.

## Out of scope

- **[`godot-bridge`](https://github.com/kkjang/godot-bridge) integration (bridgex).** Deferred indefinitely. The tool writes `.tres` files to disk; the user (or their agent) imports them manually or via their own wiring. Keeps sprite-gen decoupled from any specific editor-control tool.
- **Aseprite `.ase` file parsing.** Ingestion is always from PNG + optional manifest JSON.
- **Live preview window.** Static GIF preview only. No GUI in v1.
- **3D model generation.** Different tool, different day.
- **Skeletal / bone animation.** Frame-based only.

## Plan ordering and rationale

Plans must be executed in this order. Earlier plans are prerequisites for later ones.

| # | Plan | Why this position | File |
|---|---|---|---|
| 01 | Skeleton | Proves the dispatch, envelope, and spec architecture. No real features. Tiny PR, fast review. | `01-skeleton.md` |
| 02 | CI + Releases | Lock in build/test/release before accumulating code. Mirrors [`godot-bridge`](https://github.com/kkjang/godot-bridge) patterns. Second-smallest PR. | `02-ci-releases.md` |
| 03 | Inspect | Introduces the `pixel` package (foundational). Zero writes — read-only, easiest to test. First "real" feature. | `03-inspect.md` |
| 04 | Palette ops | Introduces `palette` package. `palette extract` and `palette apply` are standalone-useful and unblock snap. | `04-palette.md` |
| 05 | Pixel snap | Depends on `palette` (snap uses a target palette). Completes the "clean up a single PNG" story. | `05-snap.md` |
| 06 | Slice | Introduces `sheet` and `manifest` packages. Turns one sheet into many frames + manifest — gateway to everything animation-related. | `06-slice.md` |
| 07 | Align + Diff | Frame-level ops that depend on slice having run. Align fixes drift; diff verifies results. | `07-align-diff.md` |
| 08 | Export pipeline + generic formats | Introduces the format registry and the `export` command. Ships `gif` and `sheet-png` formats (both engine-agnostic). | `08-export-pipeline.md` |
| 09 | Godot export formats | First engine-specific formats: `godot-spriteframes` and `godot-atlas`. Validates that the registry extends cleanly. | `09-godot-export.md` |

## Pipeline this builds toward

After all nine plans are merged, an agent can run the full pipeline on AI-generated input:

```bash
# Clean up (plans 04, 05)
sprite-gen snap scale   knight.png --factor auto
sprite-gen palette extract knight.png --max 16 > palette.hex
sprite-gen snap pixels  knight.png --palette palette.hex

# Slice into frames (plan 06)
sprite-gen slice grid   knight_snapped.png --cols 4 --rows 1 --out frames/

# Fix drift, verify (plans 07, 08)
sprite-gen align frames frames/ --anchor feet
sprite-gen export       frames/ --format gif --fps 8 --out preview.gif

# Export to Godot (plan 09)
sprite-gen export       frames/ --format godot-spriteframes --anim walk:*.png --out walk.tres
```

Each intermediate step is independently useful; the full chain is the happy path.

## Repository layout (post plan 09)

```
sprite-gen/
  go.mod
  README.md
  AGENTS.md
  releases.yaml
  .github/workflows/
    ci.yml
    release.yml
  cmd/sprite-gen/
    main.go              # dispatch + global flags (THIN)
    cmd_version.go
    cmd_spec.go
    cmd_inspect.go
    cmd_slice.go
    cmd_snap.go
    cmd_palette.go
    cmd_align.go
    cmd_diff.go
    cmd_export.go
    main_test.go
  internal/
    pixel/               # load/save PNG, scale, bbox, alpha
    palette/             # extract, quantize, snap, read/write palette files
    sheet/               # grid detection, slice, pack
    align/               # centroid/bbox/feet anchors, pivot computation
    diff/                # frame comparison
    manifest/            # shared JSON manifest format for frame sets
    jsonout/             # {ok, data, error} envelope, writer helpers
    specreg/             # CLI spec registry, populated at init()
    export/
      export.go          # Format interface + registry
      context.go         # ExportContext (frames, manifest, options)
      formats/
        gif/
          gif.go
        sheetpng/
          sheetpng.go
        godot/
          spriteframes.go
          atlas.go
          tres.go        # shared .tres writer primitives
  testdata/
    input/               # messy sheets, drifting walk cycles, etc.
    golden/              # expected PNGs, palettes, .tres snapshots
```

Every `cmd_*.go` file stays under ~150 lines — they are thin flag parsers that call into `internal/*`. All heavy logic is in `internal/` packages that are independently testable.

## Conventions (apply to every plan)

- Go 1.22+
- `flag.NewFlagSet` per subcommand (no CLI framework)
- Package `main` for everything under `cmd/sprite-gen/`; one file per top-level verb
- `internal/*` packages: one responsibility each, exported API minimal
- Error messages are imperative and actionable, never stack traces
- `--json` flag on every command emits `{"ok": bool, "data": {...}, "error": "string"}`
- `--dry-run` on every file-writing command
- Default output paths are deterministic: `./out/<subject>/<stem>/...`
- Golden test files under `testdata/golden/` regenerable with `go test -update`
- No dependencies on [`godot-bridge`](https://github.com/kkjang/godot-bridge) at build time or runtime
