# AGENTS

## Conventions

- Keep command handlers thin; place reusable logic under `internal/`.
- Every command supports `--json` for the `{ok, data, error}` envelope.
- `spec` is the command registry source of truth; each command self-registers from `init()`.
- Prefer actionable error messages over stack traces.
- Keep changes small and verify assumptions against the current code before expanding scope.

## Release Conventions

- Use plain semver tags: `vX.Y.Z`.
- Bump releases with small PRs that touch `releases.yaml`.
- Never create tags manually; the GitHub release workflow owns tagging.
- Release notes come from PR titles and labels, so keep them clean and intentional.
