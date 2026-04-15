# Plan 02 — CI + Releases

## Goal

Lock in automated build/test/release infrastructure before accumulating feature code. Mirrors the patterns in [`godot-bridge/.github/workflows/`](https://github.com/kkjang/godot-bridge/tree/main/.github/workflows) but simplified for a single-component repo (sprite-gen has no plugin or LSP side).

## Scope

**In:**
- `.github/workflows/ci.yml` — build + test on every PR and push to main
- `.github/workflows/release.yml` — auto-tag and release when `releases.yaml` bumps
- `releases.yaml` — single-line version declaration for the CLI
- `.github/release-sprite-gen.yml` — release notes template
- Update `README.md` with a short "Release process" section
- Update `AGENTS.md` with release workflow conventions

**Out:**
- Component labels / path filters (sprite-gen is a single component; no need)
- Matrix builds across OSes (Go stdlib image is portable; single linux runner fine for v1)
- Pre-built release binaries (users install via `go install ...@latest`; uploading artifacts is future work)
- Any feature code

## File plan

```
sprite-gen/
  releases.yaml                         # sprite-gen: v0.1.0
  .github/
    workflows/
      ci.yml                            # lint + vet + test + build
      release.yml                       # tag + GitHub Release on version bump
    release-sprite-gen.yml              # release notes category config
```

## `releases.yaml`

```yaml
sprite-gen: v0.1.0
```

The version declaration is intentionally trivial. The workflow reads this file on pushes to the default branch; if the version string is new (no matching tag exists), it creates the tag and release.

Tag format: `vX.Y.Z` (plain semver — no module path prefix, since this repo is a single module at its root, unlike [`godot-bridge`](https://github.com/kkjang/godot-bridge) which has three submodules).

## `ci.yml`

Triggers:
- `pull_request` against `main`
- `push` to `main`

Jobs:

```yaml
name: CI
on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true
      - name: go vet
        run: go vet ./...
      - name: go test
        run: go test -race -count=1 ./...
      - name: go build
        run: go build -o /tmp/sprite-gen ./cmd/sprite-gen
      - name: smoke test
        run: |
          /tmp/sprite-gen version
          /tmp/sprite-gen spec | python3 -c 'import json,sys; json.load(sys.stdin)'
```

The smoke test double-checks that the binary actually runs and that `spec` emits valid JSON — cheap insurance against regressions that slip past unit tests.

## `release.yml`

Triggers:
- `workflow_run` completion of `CI` on `main`, success only
- (Not `push` directly — we want the tag to only appear after CI has verified the commit)

Logic:

```yaml
name: Release
on:
  workflow_run:
    workflows: [CI]
    types: [completed]
    branches: [main]

jobs:
  release:
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.event.workflow_run.head_sha }}
      - name: Read version
        id: ver
        run: |
          v=$(awk '/^sprite-gen:/ {print $2}' releases.yaml)
          echo "version=$v" >> "$GITHUB_OUTPUT"
      - name: Check if tag exists
        id: check
        run: |
          if git rev-parse "refs/tags/${{ steps.ver.outputs.version }}" >/dev/null 2>&1; then
            echo "exists=true" >> "$GITHUB_OUTPUT"
          else
            echo "exists=false" >> "$GITHUB_OUTPUT"
          fi
      - name: Create tag and release
        if: steps.check.outputs.exists == 'false'
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ steps.ver.outputs.version }}
          generate_release_notes: true
          target_commitish: ${{ github.event.workflow_run.head_sha }}
```

Notes on the pattern:
- "Release only after CI passes on the default branch" is the same guard [`godot-bridge`](https://github.com/kkjang/godot-bridge) uses.
- Unlike [`godot-bridge`](https://github.com/kkjang/godot-bridge), we do not need `component:` labels — there's one component and it's the whole repo.
- We do not pre-build and upload binaries here. If/when we decide to, the step goes after the tag creation and uses a Go build matrix.

## `release-sprite-gen.yml` (release notes template)

GitHub's `generate_release_notes: true` reads `.github/release.yml` by default. We override with an sprite-gen-specific config so release notes group changes cleanly:

```yaml
changelog:
  categories:
    - title: Features
      labels: [feature, enhancement]
    - title: Bug fixes
      labels: [bug, fix]
    - title: Documentation
      labels: [docs]
    - title: Internal
      labels: [chore, refactor, ci]
    - title: Other changes
      labels: ["*"]
```

(File path: `.github/release.yml` — the `-sprite-gen` suffix was a copy-paste from [`godot-bridge`](https://github.com/kkjang/godot-bridge)'s multi-component naming; drop the suffix here since this repo has only one release stream.)

## README additions

Add a "Release process" section:

```markdown
## Release process

1. Open a small PR that bumps `sprite-gen` in `releases.yaml`.
2. Merge after CI passes.
3. The `Release` workflow runs once CI on `main` succeeds; it creates the
   tag and a GitHub Release with auto-generated notes.

Tags use plain semver (`v0.1.0`). Install a specific version with:

    go install github.com/<owner>/sprite-gen/cmd/sprite-gen@v0.1.0

Install latest:

    go install github.com/<owner>/sprite-gen/cmd/sprite-gen@latest
```

## AGENTS.md additions

Add a "Release conventions" section that repeats, for agents:
- Semver tags `vX.Y.Z`, no prefix.
- Bump via small PRs that touch only `releases.yaml`.
- Never tag manually; the workflow owns tagging.
- Release notes come from PR titles + labels. Prefer conventional-commit-style PR titles.

## Acceptance criteria

1. A PR that touches any `.go` file runs `ci.yml` and goes green.
2. Merging a PR that bumps `releases.yaml` triggers `release.yml`, which:
   - creates a tag matching the new version
   - creates a GitHub Release with generated notes
3. Merging a PR that does *not* bump `releases.yaml` does not create a tag.
4. Re-running the release workflow with an already-existing tag is a no-op (the `check` step prevents duplicates).
5. `go install github.com/<owner>/sprite-gen/cmd/sprite-gen@<tag>` resolves after the release exists.

## Manual verification after merge

Once this PR is merged and a second PR bumps `releases.yaml` from `v0.1.0` to `v0.1.1`:

```bash
# After release workflow runs:
gh release list --repo <owner>/sprite-gen
# Should show v0.1.1 with notes

git ls-remote --tags origin | grep v0.1.1
# Should show the tag sha matching the release commit

go install github.com/<owner>/sprite-gen/cmd/sprite-gen@v0.1.1
sprite-gen version
# Should print v0.1.1
```

## Suggested commit message

```
ci: build/test workflow + releases.yaml-driven tagging

Single-component mirror of the [godot-bridge](https://github.com/kkjang/godot-bridge) release pattern: CI on
every PR, auto-tag + GitHub Release when releases.yaml ticks up on
main. No binaries uploaded — go install is the distribution channel.
```

## Notes for the next plan

- The `version` subcommand (from plan 01) currently prints a hardcoded string. A natural follow-up is to inject it via `-ldflags "-X main.version=$(git describe --tags)"` at release time. That is out of scope for plan 02 — if we add it, it belongs in a tiny follow-up PR after plan 02 merges, not bundled here.
- No secrets are required for this workflow — `GITHUB_TOKEN` is automatic.
