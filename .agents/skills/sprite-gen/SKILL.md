---
name: sprite-gen
description: Use sprite-gen from coding agents to inspect, clean, segment or slice, align, resize, and export sprite assets with deterministic CLI workflows and JSON responses.
license: GPL-3.0-or-later
compatibility: opencode
metadata:
  audience: agents
  cli: sprite-gen
  workflow: sprite-pipeline
---

## What I do

- Help agents choose the right `sprite-gen` pipeline for generated sprite art.
- Keep workflows grounded in the live CLI registry from `sprite-gen --json spec`.
- Prefer deterministic default outputs under `out/<subject>/<stage>/...`.
- Prefer machine-readable `--json` output when the result informs later steps.

## Before using me

- Ensure the `sprite-gen` binary is installed and available on `PATH`.
- Start by running `sprite-gen --json spec` to discover the current command surface.
- Run `sprite-gen export --list-formats` before choosing an export target.
- Do not assume planned commands or engine-specific exporters exist unless the live registry shows them.

## Decision guide

- Use `sprite-gen inspect sheet PATH --json` first for whole-image diagnosis.
- Use `sprite-gen inspect frame PATH --json` when you need bbox or pivot hints for a single frame.
- Use `sprite-gen prep alpha` only when the PNG already has real transparency and needs low-alpha haze removed.
- Use `sprite-gen prep background` for fake or opaque backgrounds. `--method auto` uses keyed removal when `--color` is provided and edge-connected removal otherwise.
- Use `sprite-gen slice grid` when the frame count and layout are known.
- Use `sprite-gen slice auto` only for transparent-gutter sheets. If detection is weak or fails, do not invent a fallback layout; switch to `slice grid` with explicit dimensions or use `segment subjects`.
- Use `sprite-gen segment subjects` for messy generated canvases where subjects are scattered instead of laid out as a real sheet.
- Use `sprite-gen align frames` before export when frames drift and need a shared pivot.
- Use `sprite-gen normalize detail` for intentional visible-height normalization across assets.
- Use `sprite-gen snap scale` only for undoing integer nearest-neighbor upscaling.
- Use `sprite-gen resize image` and `sprite-gen resize frames` late for delivery size only.

## Agent rules

- Prefer `--json` when parsing command output or making workflow decisions.
- Prefer default output paths unless the user asked for explicit locations.
- Use `--dry-run` when you need to confirm paths before writing files.
- Keep command choices aligned with `sprite-gen --json spec` instead of hardcoding stale assumptions.
- The currently shipped export formats are discovered from `sprite-gen export --list-formats`; at the time this skill was written, they are `gif` and `sheet`.
- `inspect` is read-only. Do not expect it to write files or create output directories.

## Canonical pipelines

Start with the short pipeline when the main problem is layout, background cleanup, or frame alignment:

```bash
sprite-gen inspect sheet ./walk.png --json
sprite-gen prep background ./walk.png --method auto
sprite-gen normalize detail ./out/walk/prep/background.png --target-height 48
sprite-gen segment subjects ./out/walk/normalize/detail.png --anchor feet --json
sprite-gen align frames ./out/walk/segment --anchor feet
sprite-gen resize frames ./out/walk/align --up 2
sprite-gen export ./out/walk/resize --format gif --fps 8
```

Use the full pipeline when the image also has palette noise, soft fringes, or shimmer:

```bash
sprite-gen inspect sheet ./walk.png --json
sprite-gen prep background ./walk.png --method auto
sprite-gen snap scale ./out/walk/prep/background.png --factor auto
sprite-gen palette extract ./out/walk/snap/native.png --max 32
sprite-gen snap pixels ./out/walk/snap/native.png --palette ./out/walk/palette/extracted-32.hex
sprite-gen normalize detail ./out/walk/snap/snapped.png --target-height 48
sprite-gen segment subjects ./out/walk/normalize/detail.png --anchor feet --json
sprite-gen align frames ./out/walk/segment --anchor feet
sprite-gen export ./out/walk/align --format sheet --cols 4
```

For a clean sheet with a known grid, prefer slicing over segmentation:

```bash
sprite-gen inspect sheet ./sheet.png --json
sprite-gen slice grid ./sheet.png --cols 4 --rows 1 --json
sprite-gen align frames ./out/sheet/slice --anchor feet
sprite-gen export ./out/sheet/align --format gif --fps 8 --scale 2
```

## Verification

- Re-run `sprite-gen --json spec` if you are unsure whether a command or flag exists.
- Re-run `sprite-gen export --list-formats` if you are unsure whether an exporter exists.
- After writes, use the JSON response plus any generated GIF or sheet artifact to verify the result.
