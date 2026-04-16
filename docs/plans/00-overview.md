# sprite-gen Implementation Plans — Overview

A standalone Go CLI for cleaning up AI-generated pixel art and exporting it to game-engine-native formats. Intended for use by coding agents as part of a pipeline where ChatGPT/gpt-image-1 generates raw pixel art and this tool fixes predictable flaws before importing into a game project.

## Design principles

- **Engine-agnostic core.** Image processing (slice, snap, palette, segment, align, diff) has no engine dependencies and works for Godot, Unity, Unreal, web, anywhere.
- **Pluggable export formats.** A single `export` command with a `--format=<name>` flag dispatches to a format registry. Godot is the first target; Unity, Aseprite JSON, TexturePacker, etc. drop in later without touching the rest of the code.
- **Agent ergonomics first.** Stable `{ok, data, error}` JSON envelope on every command. `sprite-gen spec` discovers the whole command surface. Idempotent writes with deterministic paths. Visual artifacts (GIFs, diff PNGs) alongside structured data so vision models can verify.
- **Minimum viable increments.** Each plan below is one reviewable PR. The first plan ships a skeleton that does almost nothing but proves the architecture. CI comes second. Features stack on top.

## Command hierarchy

Commands are grouped by **what they read** and **what they produce**:

| Read | Produce | Commands |
|---|---|---|
| Nothing | Metadata | `version`, `spec` |
| Prompt | One image | `generate image` |
| One image | Report (no files) | `inspect sheet`, `inspect frame` |
| One image | One image | `snap pixels`, `snap scale`, `prep alpha`, `prep background`, `palette apply` |
| One image | Palette file | `palette extract` |
| One image | Many images + manifest | `slice grid`, `slice auto`, `segment subjects` |
| Many images | Many images + manifest | `align frames` |
| Two images | Report + diff image | `diff frames` |
| Many images + manifest | One artifact (any format) | `export` |

The `generate` command is the single convergence point for provider-backed image creation (prompt → image, via a provider registry). The `export` command is the single convergence point for all output formats (image → artifact, via a format registry). Everything in between is pure image processing.

## Out of scope

- **[`godot-bridge`](https://github.com/kkjang/godot-bridge) integration (bridgex).** Deferred indefinitely. The tool writes `.tres` files to disk; the user (or their agent) imports them manually or via their own wiring. Keeps sprite-gen decoupled from any specific editor-control tool.
- **Aseprite `.ase` file parsing.** Ingestion is always from PNG + optional manifest JSON.
- **Live preview window.** Static GIF preview only. No GUI in v1.
- **3D model generation.** Different tool, different day.
- **Skeletal / bone animation.** Frame-based only.

See [`future-enhancements.md`](future-enhancements.md) for the broader backlog (additional `generate` sub-commands, more providers, alternate secret backends, more export formats, engine-specific integrations).

## Plan ordering and rationale

Plans must be executed in this order. Earlier plans are prerequisites for later ones.

Status legend: ✅ Done · 🚧 In progress · 📋 Planned

| # | Status | Plan | Why this position | File |
|---|---|---|---|---|
| 01 | ✅ Done | Skeleton | Proves the dispatch, envelope, and spec architecture. No real features. Tiny PR, fast review. Merged in [#2](https://github.com/kkjang/sprite-gen/pull/2). | `01-skeleton.md` |
| 02 | ✅ Done | CI + Releases | Lock in build/test/release before accumulating code. Mirrors [`godot-bridge`](https://github.com/kkjang/godot-bridge) patterns. Second-smallest PR. Merged in [#4](https://github.com/kkjang/sprite-gen/pull/4). | `02-ci-releases.md` |
| 03 | 📋 Planned | LLM Provider Generate | Adds `generate image` + provider registry (first provider: OpenAI `gpt-image-1`). Establishes external-API and secret-handling patterns before image-processing code lands; unblocks dogfooding — every downstream plan can generate its own fixtures. | `03-llm-generate.md` |
| 04 | ✅ Done | Inspect | Introduces the `pixel` package (foundational). Zero writes — read-only, easiest to test. First image-processing feature. Merged in [#6](https://github.com/kkjang/sprite-gen/pull/6). | `04-inspect.md` |
| 05 | ✅ Done | Palette ops | Introduces `palette` package. `palette extract` and `palette apply` are standalone-useful and unblock snap. | `05-palette.md` |
| 06 | ✅ Done | Pixel snap | Depends on `palette` (snap uses a target palette). Completes the "clean up a single PNG" story. | `06-snap.md` |
| 07 | ✅ Done | Slice | Introduces `sheet` and `manifest` packages. Turns one sheet into many frames + manifest — gateway to everything animation-related. | `07-slice.md` |
| 08 | ✅ Done | Segment subjects | Alternate path to `frames + manifest` from a *messy* AI-generated canvas: threshold alpha, connected-component label each subject, normalize into fixed-size cells with baseline alignment. Composes with align/diff/export unchanged. | `08-segment.md` |
| 08.5 | ✅ Done | Background cleanup | Adds `prep background` for fake transparency and opaque generated backgrounds using extensible cleanup methods (`key`, `edge`). | `08.5-background-cleanup.md` |
| 09 | ✅ Done | Align + Diff | Frame-level ops that depend on slice or segment having run. Align fixes drift; diff verifies results. | `09-align-diff.md` |
| 10 | ✅ Done | Export pipeline + generic formats | Introduces the format registry and the `export` command. Ships `gif` and `sheet-png` formats (both engine-agnostic). | `10-export-pipeline.md` |
| 11 | 📋 Planned | Godot export formats | First engine-specific formats: `godot-spriteframes` and `godot-atlas`. Validates that the registry extends cleanly. | `11-godot-export.md` |

## Pipeline this builds toward

After all twelve plans are merged, an agent can run the full pipeline end-to-end without leaving the CLI. There are two canonical entry paths into the frame-set world, picked by input shape:

### Happy path — clean sheet input

```bash
# Generate (plan 03)
sprite-gen generate image "knight walk cycle, 4 frames, 32x32, pixel art" \
    --n 1 --size 1024x1024 --out knight.png

# Clean up (plans 05, 06, 07, 08.5)
sprite-gen snap scale   knight.png --factor auto
sprite-gen palette extract out/knight/snap/native.png --max 16
sprite-gen snap pixels  out/knight/snap/native.png --palette out/knight/palette/extracted-16.hex
sprite-gen prep background out/knight/snap/snapped.png --method auto
sprite-gen prep alpha   out/knight/prep/background.png --alpha-threshold 128

# Slice into frames (plan 07)
sprite-gen slice grid   out/knight/prep/clean.png --cols 4 --rows 1 --out frames/

# Fix drift, verify (plans 09, 10)
sprite-gen align frames frames/ --anchor feet
sprite-gen export       frames/ --format gif --fps 8 --out preview.gif

# Export to Godot (plan 11)
sprite-gen export       frames/ --format godot-spriteframes --anim walk:*.png --out walk.tres
```

### Messy path — real `gpt-image-1` output

`gpt-image-1` tends to ignore prompted sprite-sheet constraints: subjects end up scattered across an oversized canvas with glow halos, soft edges, and sometimes fully opaque fake backgrounds. `segment subjects` (plan 08) salvages these once the background is actually transparent:

```bash
# Inspect to diagnose the mess
sprite-gen inspect sheet knight.png --json
#   -> huge bbox, high aa_score, many fractional-alpha pixels, or fully opaque fake background

# Remove an opaque fake background when needed
sprite-gen prep background knight.png --method auto

# Segment the canvas directly into normalized frames (plan 08)
sprite-gen segment subjects out/knight/prep/background.png --cell 32x32 --expected 4 --anchor feet \
    --out frames/

# Continue with the same align → export pipeline
sprite-gen align frames frames/ --anchor feet
sprite-gen export       frames/ --format godot-spriteframes --out walk.tres
```

Each intermediate step is independently useful; the full chain is the happy path.

## Repository layout (post plan 11)

```
sprite-gen/
  go.mod
  .env.example           # template for provider API keys
  .gitignore             # ignores .env, out/, *.key, *.pem, credentials
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
    cmd_generate.go
    cmd_inspect.go
    cmd_slice.go
    cmd_segment.go
    cmd_snap.go
    cmd_palette.go
    cmd_prep.go
    cmd_align.go
    cmd_diff.go
    cmd_export.go
    main_test.go
  internal/
    pixel/               # load/save PNG, scale, bbox, alpha, alpha masks, morphology
    background/          # fake/opaque background removal methods
    palette/             # extract, quantize, snap, read/write palette files
    sheet/               # grid detection, slice, pack
    segment/             # connected-component labeling, cell normalization
    align/               # centroid/bbox/feet anchors, pivot computation
    diff/                # frame comparison
    manifest/            # shared JSON manifest format for frame sets
    jsonout/             # {ok, data, error} envelope, writer helpers
    specreg/             # CLI spec registry, populated at init()
    secrets/             # godotenv-backed loader + redaction helper
    provider/            # ImageProvider interface + registry
      openai/            # gpt-image-1 client (stdlib net/http)
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
- Default output paths are deterministic: `./out/<subject>/<stage>/...`
- Golden test files under `testdata/golden/` regenerable with `go test -update`
- No dependencies on [`godot-bridge`](https://github.com/kkjang/godot-bridge) at build time or runtime
- Every plan-implementation PR must fold the plan's durable decisions into `AGENTS.md` (contributor conventions) and/or `README.md` (user surface) before merging. The `docs/plans/*.md` files are scaffolding and will be removed once all plans are complete; anything worth keeping must migrate out of them.
