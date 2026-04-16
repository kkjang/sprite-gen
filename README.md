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

## Commands

Inspect a whole sheet and let the CLI guess a grid:

```bash
sprite-gen inspect sheet ./sheet.png
```

Inspect a single frame and report bbox and a simple feet pivot hint:

```bash
sprite-gen inspect frame ./frame.png --json
```

`inspect frame` ignores ultra-low-alpha stray pixels by default when
computing the bbox and pivot. Tune it for softer assets with:

```bash
sprite-gen inspect frame ./frame.png --alpha-threshold 1 --json
```

Extract a palette from a PNG to the deterministic default output path:

```bash
sprite-gen palette extract ./sheet.png --max 16
```

Generated outputs are grouped by subject and processing stage under `out/`.
For example, running `snap scale` on `slime3.png` writes to
`out/slime3/snap/native.png`.

Use `--out -` when you want `palette extract` on stdout for piping.

Apply a palette and write to the deterministic default output path:

```bash
sprite-gen palette apply ./sheet.png --palette ./palette.hex
```

Remove soft alpha edges, then snap the remaining visible pixels to a palette:

```bash
sprite-gen snap pixels ./sheet.png --palette ./palette.hex --alpha-threshold 128
```

Detect and undo integer nearest-neighbor upscaling:

```bash
sprite-gen snap scale ./sheet.png --factor auto
```

List the registered command surface:

```bash
sprite-gen spec --markdown
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
