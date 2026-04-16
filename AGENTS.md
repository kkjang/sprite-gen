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
- `segment subjects` is the alternate one-image-to-frame-set path for messy generated canvases; it writes the same `frames + manifest` contract as `slice`, and `manifest.frames[].rect` records source-space component bounds from the original canvas.
- `align frames` writes every output frame onto a shared canvas and sets a common output-space `manifest.frames[].pivot` across the aligned set.

## Release Conventions

- Use plain semver tags: `vX.Y.Z`.
- Bump releases with small PRs that touch `releases.yaml`.
- Never create tags manually; the GitHub release workflow owns tagging.
- Release notes come from PR titles and labels, so keep them clean and intentional.
