# sprite-gen

Minimal Go CLI for cleaning up AI-generated pixel art and exporting it to game-engine-native formats.

## Build

```bash
go build ./cmd/sprite-gen
```

## Test

```bash
go test ./...
```

## Install

```bash
go install ./cmd/sprite-gen
```

Install the latest tagged release:

```bash
go install github.com/kkjang/sprite-gen/cmd/sprite-gen@latest
```

Install a specific release:

```bash
go install github.com/kkjang/sprite-gen/cmd/sprite-gen@v0.1.0
```

## Release process

1. Open a small PR that bumps `sprite-gen` in `releases.yaml`.
2. Merge after CI passes.
3. The `Release` workflow runs after `CI` succeeds on `main`, then creates the tag and GitHub Release.

Tags use plain semver like `v0.1.0`.
