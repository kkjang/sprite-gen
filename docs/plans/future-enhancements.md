# Future enhancements

A living backlog of features we've thought about but explicitly deferred. Not a commitment — a place to park ideas so they stop cluttering individual plan files. Any item promoted to a real plan gets its own `NN-<name>.md` and comes out of this doc.

## `generate` sub-commands (future)

- **`generate sheet`** — direct-to-spritesheet with grid-aware prompt (prompt the model for a 4×1 walk cycle, etc.).
- **`generate frames`** — multi-prompt animation set (walk/idle/attack) in one invocation, producing a manifest.
- **`generate palette`** — palette from a textual vibe (e.g. "desert wasteland, 16 colors").
- **`generate edit PATH --prompt "..."`** — modify an existing PNG via OpenAI's `/v1/images/edits`.
- **`generate variations PATH`** — N variations of an existing frame.

## Additional providers

- Anthropic (if/when an image endpoint ships).
- Local Stable Diffusion via a running `sd-webui` or `comfyui` HTTP endpoint.
- Replicate / Fal / other aggregator APIs.
- Out-of-tree provider plugins discovered at runtime.

## Secret storage upgrades

- OS keychain backend (`zalando/go-keyring`) as an opt-in alternative to `.env`.
- Short-lived token support (OAuth device flow) for providers that offer it.
- Per-project credential scoping beyond `SPRITE_GEN_*` env vars.

## Generation ergonomics

- Prompt templates / prompt library under `~/.config/sprite-gen/prompts/`.
- Cost & token usage reporting on every call; `--budget $X` guardrail.
- Response caching (by prompt+model+size+seed hash) under a subject-scoped output path such as `./out/<subject>/generate/cache/`.
- Batch mode: one invocation, many prompts from a file.
- Seed control and deterministic replay.

## Export formats

- Unity `SpriteAtlas` / sprite import JSON.
- Aseprite JSON atlas.
- TexturePacker format.
- Web-friendly CSS sprite sheets.

## Processing features

- Aseprite `.ase` ingestion (currently out of scope in `00-overview.md`).
- Skeletal / bone animation (currently out of scope).
- 3D model generation (currently out of scope — different tool).
- Live preview GUI (currently out of scope — GIF preview only in v1).
- Semantic / model-based subject cutout that does not depend on keyed colors or border-connected backgrounds.
- Background despill / matte decontamination to clean color fringes after keyed removal.
- Automatic checkerboard and textured-background detection for fake-transparency artifacts.
- Folding background-removal techniques directly into `segment subjects` as optional preprocessing once the standalone `prep background` command proves out.

## Testing

- Add a small curated set of real PNG fixtures under `testdata/` for heavier integration coverage of messy generated assets: soft alpha halos, scattered subjects, near-touching components, and `segment subjects` tuning paths like `--erode`, `--dilate`, and `--fit crop`. Keep these repo-owned and deterministic rather than depending on local `.cache` assets in normal test runs.

## Engine integration

- [`godot-bridge`](https://github.com/kkjang/godot-bridge) integration (deferred indefinitely per the overview's "Out of scope").
- Unity Editor import hooks.
- Unreal import hooks.
