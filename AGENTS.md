# AGENTS

## Conventions

- Keep command handlers thin; place reusable logic under `internal/`.
- Every command supports `--json` for the `{ok, data, error}` envelope.
- Every file-writing command supports `--dry-run` and uses deterministic subject-first output paths under `out/<subject>/<stage>/...`.
- Frame-set manifests record source-space rectangles; if a sliced frame is trimmed, `manifest.frames[].rect` still points at the trimmed region in the original sheet.
- `spec` is the command registry source of truth; each command self-registers from `init()`.
- Read-only commands like `inspect` should not write files or create default output paths.
- Prefer actionable error messages over stack traces.
- Keep changes small and verify assumptions against the current code before expanding scope.
- `slice auto` should fail on weak grid detection instead of silently guessing a fallback layout.
- `prep alpha` is the explicit cleanup step for transparent PNGs with low-alpha haze before `slice`; later commands may reuse the same primitives but must not require prep as a separate prerequisite.
- `prep background` is the explicit cleanup step for fake or opaque generated backgrounds; it is distinct from `prep alpha` because keyed/edge-connected background removal and alpha-threshold cleanup solve different problems.
- `normalize detail` is an optional single-image project-consistency step that intentionally reduces detail toward a target visible height or explicit integer factor; keep it distinct from corrective `snap scale`.
- `resize image` and `resize frames` are late-stage delivery-size nearest-neighbor steps; keep them distinct from both corrective `snap scale` and stylistic `normalize detail`.
- `segment subjects` is the alternate one-image-to-frame-set path for messy generated canvases; it writes the same `frames + manifest` contract as `slice`, and `manifest.frames[].rect` records source-space component bounds from the original canvas.
- `align frames` writes every output frame onto a shared canvas and sets a common output-space `manifest.frames[].pivot` across the aligned set.
- `export` is the single format-registry entry point for frame-set outputs; new formats self-register from `init()` without changing command dispatch.
- `export --format sheet-png` writes a single PNG artifact at `--out`; it does not write a companion manifest and may pad mixed-size inputs into max-size cells.
- Prefer the short pipeline (`prep background? -> normalize detail? -> segment/slice -> align -> resize? -> export`) when the problem is mostly layout/background cleanup; prefer the full pipeline (`... -> snap scale -> palette extract -> snap pixels -> normalize detail? -> segment/slice -> align -> resize? -> export`) when the image also has palette noise, shimmer, or soft-edge artifacts. Visually validate full-pipeline results on opaque-background inputs because extracted palettes can preserve fringe colors left by incomplete cleanup.

## Release Conventions

- Use plain semver tags: `vX.Y.Z`.
- Bump releases with small PRs that touch `releases.yaml`.
- Never create tags manually; the GitHub release workflow owns tagging.
- Release notes come from PR titles and labels, so keep them clean and intentional.
