# Plan 11 — Godot Export Formats

## Goal

Add the first engine-specific export formats: `godot-spriteframes` (a `SpriteFrames` resource for `AnimatedSprite2D`) and `godot-atlas` (per-frame `AtlasTexture` resources). This validates that the format registry from plan 10 extends cleanly to engine-specific output without touching generic code.

## Scope

**In:**
- `internal/export/formats/godot/spriteframes.go` — `SpriteFrames` `.tres` builder
- `internal/export/formats/godot/atlas.go` — `AtlasTexture` `.tres` builder per frame
- `internal/export/formats/godot/tres.go` — shared `.tres` writer primitives (UID gen, escaping)
- `internal/export/formats/godot/uid.go` — deterministic UID from path
- Both formats self-register; `cmd_export.go` is unchanged
- Tests: snapshot-based (golden `.tres` files), UID determinism

**Out:**
- [`godot-bridge`](https://github.com/kkjang/godot-bridge) integration (`exec.LookPath("godot-bridge")`) — deferred indefinitely per the overview
- Any Godot editor API calls
- Any runtime dependency on Godot

## File plan

```
sprite-gen/
  internal/
    export/
      formats/
        godot/
          tres.go                       # Resource, ExtResource, SubResource; WriteTres
          uid.go                        # DeterministicUID(path string) string
          spriteframes.go               # Format: "godot-spriteframes"
          atlas.go                      # Format: "godot-atlas"
          spriteframes_test.go
          atlas_test.go
          uid_test.go
  testdata/
    golden/
      export/
        godot/
          walk.tres                     # expected SpriteFrames output
          frame_000.tres                # expected AtlasTexture output
          frame_001.tres
```

## `.tres` writer design (`tres.go`)

```go
package godot

// Resource represents a parsed/constructed Godot .tres file.
type Resource struct {
    Type        string
    Format      int    // always 3 for Godot 4.x
    UID         string // "uid://..."
    LoadSteps   int
    ExtResources []ExtResource
    SubResources []SubResource
    Properties  []Property     // [resource] section key=value pairs
}

type ExtResource struct {
    ID   string
    Type string
    Path string // "res://..."
}

type SubResource struct {
    ID         string
    Type       string
    Properties []Property
}

type Property struct {
    Key   string
    Value string // pre-formatted Godot expression string
}

// WriteTres writes the resource to w in Godot .tres text format.
// It handles:
//   - [gd_resource ...] header with uid, load_steps, format
//   - [ext_resource ...] sections
//   - [sub_resource ...] sections
//   - [resource] section with top-level properties
func WriteTres(w io.Writer, r Resource) error
```

This is a typed builder, not string templates. All escaping happens in
`WriteTres`. Callers construct the struct; they never interpolate strings
into raw GDScript syntax.

## UID generation (`uid.go`)

Godot 4.x requires a `uid://...` identifier on every `.tres` file. For
idempotent reruns we derive the UID deterministically from the resource path:

```go
// DeterministicUID returns a Godot-compatible uid://xxxxx string derived
// from sha1(path)[:8] encoded in base58. The output is stable for a given
// path and does not require Godot to be running.
func DeterministicUID(path string) string
```

Why base58? Godot's UID format uses a custom alphabet similar to base58.
We only need stability and uniqueness for the files we write; we do not
need to match Godot's internal generation algorithm. If Godot reimports
the file, it may assign a different UID internally — that's fine, the
`.tres` we write is an initial import target.

## `godot-spriteframes` format

Self-registers as `"godot-spriteframes"`.

Options read from `ctx.Options`:
- `anim` (default `"idle"`): animation name
- `fps` (default `"8"`): frames per second (float in the .tres)
- `loop` (default `"true"`): whether the animation loops
- `sheet` (optional): path to the source sprite sheet PNG (written as `res://...` in ext_resource); if absent, each frame PNG is its own ext_resource

Output: a single `.tres` file with:
- One `ext_resource` per unique source texture
- One `sub_resource AtlasTexture` per frame (pointing into the sheet)
- A `[resource]` section with the `animations` array in GDScript dict syntax

### Minimal SpriteFrames structure

```
[gd_resource type="SpriteFrames" load_steps=6 format=3 uid="uid://abc123"]

[ext_resource type="Texture2D" path="res://frames/walk_sheet.png" id="1"]

[sub_resource type="AtlasTexture" id="AtlasTexture_aaa"]
atlas = ExtResource("1")
region = Rect2(0, 0, 32, 32)

[sub_resource type="AtlasTexture" id="AtlasTexture_bbb"]
atlas = ExtResource("1")
region = Rect2(32, 0, 32, 32)

[resource]
animations = [{
"frames": [
{"duration": 1.0, "texture": SubResource("AtlasTexture_aaa")},
{"duration": 1.0, "texture": SubResource("AtlasTexture_bbb")}
],
"loop": true,
"name": &"walk",
"speed": 8.0
}]
```

When `--sheet` is provided: all frames reference the same `ext_resource`
and use `region = Rect2(...)` from the manifest. When no sheet is provided:
each frame PNG is its own `ext_resource` and the `AtlasTexture` covers
the full image (`Rect2(0,0,w,h)`).

### Deterministic SubResource IDs

SubResource IDs are derived from `sha1(frame_path)[:6]` in lowercase hex.
This makes reruns produce byte-identical `.tres` files given the same
inputs, which means Git won't show spurious diffs.

## `godot-atlas` format

Self-registers as `"godot-atlas"`.

Options:
- `sheet` (required): path to the source sprite sheet PNG written as `res://...`
- `manifest` (optional): path to manifest JSON (for Rect data)

Output: one `.tres` file per frame, named `<stem>_NNN.tres`, each containing:

```
[gd_resource type="AtlasTexture" load_steps=2 format=3 uid="uid://..."]

[ext_resource type="Texture2D" path="res://walk_sheet.png" id="1"]

[resource]
atlas = ExtResource("1")
region = Rect2(0, 0, 32, 32)
```

This format is useful when you want each animation frame to be a standalone
`Texture2D` resource (e.g., assigned individually to `Sprite2D` nodes, or
used as items in an `ItemList`).

## Testing

`uid_test.go`:
- `DeterministicUID("res://walk.tres")` is stable across repeated calls.
- Two different paths produce different UIDs (probabilistic; just check 20 cases).
- Output matches the pattern `uid://[a-km-zA-HJ-NP-Z1-9]+`.

`spriteframes_test.go`:
- Building a `Resource` with 2 `SubResource` entries and calling `WriteTres`
  produces output that contains `[sub_resource type="AtlasTexture"` twice.
- Golden snapshot: `godot-spriteframes` export of `walk_4x1` frames matches
  `golden/export/godot/walk.tres` byte-for-byte.
- Rerunning the same export produces identical bytes (idempotent).
- `animations` array in the output contains the correct `name` and `speed` values.

`atlas_test.go`:
- `godot-atlas` export of 4 frames produces 4 `.tres` files.
- Each `.tres` file contains a `Rect2` matching the frame's rect in the manifest.
- Golden snapshot for `frame_000.tres`.

Command-level tests (via `cmd_export.go`, unchanged from plan 10):
- `export out/slice/walk_4x1 --format godot-spriteframes --json` → envelope with `ok: true`, `out` ending in `.tres`.
- `export out/slice/walk_4x1 --format godot-atlas --sheet res://walk_sheet.png --json` → envelope with `tres_paths` array of 4 paths.
- `export --list-formats --json` now shows 4 formats: `gif`, `sheet-png`, `godot-spriteframes`, `godot-atlas`.

## Acceptance criteria

1. `go test ./...` passes including golden `.tres` snapshots.
2. Running the full pipeline:
   ```bash
   sprite-gen slice grid testdata/input/slice/walk_4x1.png --cols 4
   sprite-gen export out/slice/walk_4x1 --format godot-spriteframes \
     --anim walk --fps 8 --out walk.tres
   ```
   produces a `walk.tres` that:
   - Is valid UTF-8 text.
   - Contains `type="SpriteFrames"`.
   - Contains `"name": &"walk"` and `"speed": 8.0`.
   - Is byte-identical on rerun.
3. A Godot 4.3+ project can import `walk.tres` without errors (manual verification step; not in CI).
4. `sprite-gen export --list-formats` shows four formats.
5. `sprite-gen spec` shows sixteen commands.
6. No new non-stdlib dependencies. `sha1` is in stdlib (`crypto/sha1`). Base58 encoding is ~20 lines of pure Go; write it inline in `uid.go`, not as a dependency.

## Suggested commit message

```
feat(godot): godot-spriteframes and godot-atlas export formats

Add two engine-specific export formats via the plan-10 registry:
godot-spriteframes (.tres for AnimatedSprite2D) and godot-atlas
(per-frame AtlasTexture .tres). Deterministic UIDs and SubResource
IDs ensure byte-identical reruns. No changes to cmd_export.go.
```

## Post-plan-11 state

After this plan merges, the full pipeline described in the overview is
functional:

```bash
# Clean up
sprite-gen snap scale   knight.png --factor auto
sprite-gen palette extract out/snap/knight_native.png --max 16 > p.hex
sprite-gen snap pixels  out/snap/knight_native.png --palette p.hex

# Slice
sprite-gen slice grid   out/snap/knight_native_snapped.png --cols 4 --rows 1

# Fix drift and preview
sprite-gen align frames out/slice/knight_native_snapped --anchor feet
sprite-gen export       out/align/knight_native_snapped --format gif --fps 8 --scale 2

# Export to Godot
sprite-gen export       out/align/knight_native_snapped \
  --format godot-spriteframes --anim walk --fps 8 --out knight_walk.tres
```

## Notes for future plans (not scoped here)

- **Tiled/TexturePacker JSON** — add as `sheet-json` format; zero changes to the pipeline.
- **Aseprite `.ase` input** — would replace `internal/pixel/load.go` for that input type only.
- **`--res-prefix` flag** — lets callers control the `res://` path prefix written into `.tres` files without touching the format internals.
- **UID import into a live Godot project** — out of scope; the file we write is the import target. Godot handles UID registration on first import.
