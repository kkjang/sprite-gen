# AGENTS

## Conventions

- Keep command handlers thin; place reusable logic under `internal/`.
- Every command supports `--json` for the `{ok, data, error}` envelope.
- `spec` is the command registry source of truth; each command self-registers from `init()`.
- Provider-backed generation goes through `internal/provider`; command code should depend on the registry interface, not concrete vendors.
- Secrets must never be written to stdout, stderr, JSON output, or surfaced provider errors; redact before returning messages.
- Provider HTTP tests should use `httptest.Server`; CI must not make live API calls.
- Prefer actionable error messages over stack traces.
- Keep changes small and verify assumptions against the current code before expanding scope.

## Release Conventions

- Use plain semver tags: `vX.Y.Z`.
- Bump releases with small PRs that touch `releases.yaml`.
- Never create tags manually; the GitHub release workflow owns tagging.
- Release notes come from PR titles and labels, so keep them clean and intentional.
