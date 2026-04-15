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

## Generating Sprites

1. Copy `.env.example` to `.env`.
2. Set `OPENAI_API_KEY` or `SPRITE_GEN_OPENAI_API_KEY`.
3. Run `sprite-gen generate image`.

Example:

```bash
sprite-gen generate image "red knight, pixel art, 32x32" --dry-run
sprite-gen generate image "red knight, pixel art, 32x32" --n 2 --out out/knight
```

`--dry-run` validates the request and prints the output paths without calling the provider. Use `--prompt-file path/to/prompt.txt` or `--prompt-file -` to read the prompt from a file or stdin.

## Release process

1. Open a small PR that bumps `sprite-gen` in `releases.yaml`.
2. Merge after CI passes.
3. The `Release` workflow runs after `CI` succeeds on `main`, then creates the tag and GitHub Release.

Tags use plain semver like `v0.1.0`.
