# Plan 03 — LLM Provider Generate

## Goal

Add `sprite-gen generate image "<prompt>"` backed by a pluggable provider interface, and ship one provider: OpenAI `gpt-image-1`. This closes the loop described in the overview ("ChatGPT/gpt-image-1 generates raw pixel art and this tool fixes predictable flaws") — sprite-gen becomes the *source* of pixel art as well as the cleanup pipeline. No image post-processing in this plan; the PNGs produced here feed every downstream plan (inspect, snap, palette, slice, align, diff, export).

## Scope

**In:**
- `internal/provider/` — `ImageProvider` interface + registry (mirrors the format registry pattern planned in `09-export-pipeline.md`).
- `internal/provider/openai/` — `gpt-image-1` client built on stdlib `net/http` + `encoding/json` (no vendor SDK dependency).
- `internal/secrets/` — thin wrapper that calls `godotenv.Load()` (and `godotenv.Load(".env.local")` if present) and reads env vars; redacts the key from any writer passed in.
- `cmd/sprite-gen/cmd_generate.go` — `generate image` subcommand; self-registers via `init()` like existing commands.
- `.gitignore` — add `.env`, `.env.*` (with `!.env.example`), `*.key`, `*.pem`, `credentials`, `out/`.
- `.env.example` — committed template with blank keys.
- README "Generating sprites" section (install, `.env` setup, example invocation).
- AGENTS.md entries: provider-registry pattern, secret-redaction rule, `httptest.Server` testing pattern.
- Tests using `httptest.Server` to fake the OpenAI endpoint — no live API calls in CI.

**Out:**
- Non-OpenAI providers (deferred to `future-enhancements.md`).
- Image edits (`/v1/images/edits`) and variations (deferred).
- Streaming, batching, cost reporting, response caching (deferred).
- OS keychain secret backend (deferred).
- Any image post-processing — that is the job of plans 04–10.

## File plan

```
sprite-gen/
  go.mod                                 # += github.com/joho/godotenv
  .env.example                           # template with OPENAI_API_KEY=
  .gitignore                             # += .env patterns, out/, *.key, *.pem, credentials
  README.md                              # += "Generating sprites" section
  AGENTS.md                              # += provider-registry, redaction, httptest patterns
  cmd/sprite-gen/
    cmd_generate.go                      # `generate image` handler + spec registration
    cmd_generate_test.go
  internal/
    provider/
      provider.go                        # ImageProvider, Request, Response, Image, Usage
      registry.go                        # Register, Get, Names
      registry_test.go
      openai/
        openai.go                        # Provider implementation
        openai_test.go                   # httptest.Server-backed tests
    secrets/
      secrets.go                         # Load + Redact
      secrets_test.go
```

## Interfaces

### `internal/provider`

```go
package provider

import "context"

type ImageProvider interface {
    Name() string                                        // "openai"
    Models() []string                                    // {"gpt-image-1"}
    Generate(ctx context.Context, req Request) (*Response, error)
}

type Request struct {
    Prompt string
    Model  string
    Size   string            // "1024x1024" | "1024x1536" | "1536x1024" | "auto"
    N      int
    Extra  map[string]any    // provider-specific knobs (quality, background, …)
}

type Response struct {
    Images []Image
    Model  string
    Usage  *Usage            // tokens/cost if reported; nil otherwise
}

type Image struct {
    Bytes   []byte           // decoded PNG bytes
    Format  string           // "png"
    Revised string           // revised_prompt, if returned
    Seed    string           // provider seed, if returned
}

type Usage struct {
    InputTokens  int
    OutputTokens int
}

// Registry
func Register(p ImageProvider)
func Get(name string) (ImageProvider, error)
func Names() []string
```

Each provider package calls `provider.Register(...)` from `init()`. `cmd_generate.go` only depends on `provider`, never on a specific vendor package — adding a second provider is a new directory under `internal/provider/<name>/` plus a blank-import in `cmd_generate.go`.

### `internal/provider/openai`

```go
package openai

type Provider struct {
    BaseURL    string        // default: https://api.openai.com
    HTTPClient *http.Client  // default: http.DefaultClient with 60s timeout
    APIKey     string        // read by caller via secrets.Load
}
```

Calls `POST {BaseURL}/v1/images/generations` with JSON body `{model, prompt, n, size, response_format:"b64_json"}`. Parses `data[].b64_json` → PNG bytes. Retries 429/5xx up to 3 times with jittered exponential backoff (stdlib `math/rand` + `time.Sleep`). 4xx errors surface the provider's `error.message` verbatim (key scrubbed).

### `internal/secrets`

```go
package secrets

// Load loads .env and .env.local (if present) into the process env,
// then returns the value of the first matching env var. .env.local
// takes precedence over .env. Missing files are not errors.
func Load(keys ...string) (string, error)

// Redact replaces any occurrence of key in s with "***REDACTED***".
func Redact(s, key string) string
```

`cmd_generate.go` calls `secrets.Load("SPRITE_GEN_OPENAI_API_KEY", "OPENAI_API_KEY")`. Missing key returns an actionable error: `no OpenAI credentials; set OPENAI_API_KEY in your env or ./.env (see README)`.

## Command design — `sprite-gen generate image "<prompt>"`

Command grammar: `generate` + noun, matching existing `inspect sheet`/`snap pixels`/`slice grid` patterns. Reserves room for future `generate sheet`, `generate frames`, `generate palette`, `generate edit`, `generate variations` (tracked in `future-enhancements.md`).

**Flags:**
| Flag | Default | Notes |
|---|---|---|
| `--provider` | `openai` | Looked up in `provider` registry |
| `--model` | `gpt-image-1` | Must be in provider's `Models()` list |
| `--size` | `1024x1024` | Validated against provider's accepted sizes |
| `--n` | `1` | Integer ≥ 1 |
| `--out` | `./out/generate/<stem>/image-<idx>.png` | `<stem>` = first 12 hex chars of sha256(prompt); `<idx>` zero-padded |
| `--prompt-file` | — | Read prompt from file path, or `-` for stdin (mutually exclusive with positional prompt) |
| `--dry-run` | `false` | Validate + print request; no API call |

**Text output:** one line per saved PNG with its path.

**JSON output:**
```json
{
  "ok": true,
  "data": {
    "provider": "openai",
    "model": "gpt-image-1",
    "size": "1024x1024",
    "prompt": "red knight, pixel art, 32x32",
    "images": [
      {"path": "out/generate/a1b2c3d4e5f6/image-0.png", "revised_prompt": "..."}
    ]
  }
}
```

**Error surface:**
- Missing key → exit 1, stderr envelope with actionable message.
- Provider 4xx → exit 1, envelope includes provider's error message (with key redacted).
- Provider 5xx → retried per backoff; surfaced if persistent.
- Invalid `--size` for the chosen model → exit 1 before calling the provider.

## Secrets & config

Approach: **`godotenv` + `.gitignore`** (explicit owner decision — PR review gates accidental `.env` commits).

1. `github.com/joho/godotenv` added to `go.mod`. First non-stdlib dep; acceptable because network auth is genuinely outside stdlib scope. Plan 05 (palette) was already going to introduce `go-quantize`; this one precedes it.
2. `cmd_generate.go` calls `secrets.Load(...)` on entry → `godotenv.Load()` + `godotenv.Load(".env.local")` (latter wins on conflict) + `os.Getenv` reads.
3. `secrets.Load` honors `SPRITE_GEN_OPENAI_API_KEY` before `OPENAI_API_KEY` to let users scope keys per-project.
4. **Redaction.** The key is never written to stdout/stderr/`--json` output. HTTP error bodies pass through `secrets.Redact` before surfacing. A unit test asserts no writer ever sees the key.
5. `.gitignore` additions (belt-and-suspenders; review is the primary gate):
   ```
   .env
   .env.*
   !.env.example
   *.key
   *.pem
   credentials
   out/
   ```
6. `.env.example` ships with:
   ```
   # Copy this to .env and fill in. .env is gitignored.
   OPENAI_API_KEY=
   ```

## Testing

- **Provider tests** (`internal/provider/openai/openai_test.go`): `httptest.Server` serves fixture responses. Cases: 200 success (b64_json → PNG bytes), 429 → backoff → 200, 401 → error (key scrubbed), 500 → retry exhausted, malformed body.
- **Registry tests** (`internal/provider/registry_test.go`): round-trip Register/Get/Names; concurrent Register is safe.
- **Secrets tests** (`internal/secrets/secrets_test.go`): temp-dir `.env` populates env; `.env.local` overrides `.env`; missing-file is no-op; `Redact` scrubs exact matches only.
- **Command tests** (`cmd/sprite-gen/cmd_generate_test.go`): `run(["generate","image","test","--dry-run"])` → envelope shape OK; missing key → exit 1 with actionable stderr; redaction test verifies the key never appears in captured writers.
- **No network in CI.** All HTTP goes through injected `BaseURL` + `HTTPClient`.

## Acceptance criteria

1. `go test ./...` passes; no network access during tests.
2. `sprite-gen spec` shows `generate image` with all flags.
3. `OPENAI_API_KEY=sk-... sprite-gen generate image "red knight, pixel art, 32x32" --n 1 --dry-run` prints what it would send; `--json` variant emits a valid envelope.
4. Without a key, the command exits non-zero with an actionable message and never hits the network.
5. Adding a second provider under `internal/provider/<name>/` requires zero changes to `cmd_generate.go` (registry-only).
6. Exactly one new dependency added: `github.com/joho/godotenv`.
7. `README.md` updated with a "Generating sprites" section (install → `.env` setup → example invocation). `AGENTS.md` updated with the provider-registry pattern, secret-redaction rule, and `httptest.Server` testing pattern. Nothing critical from this plan remains *only* in `03-llm-generate.md`.
8. Spot-check: any durable context from plan 01 still living only in `01-skeleton.md` has been migrated into `README.md`/`AGENTS.md` as part of this PR.

## Suggested commit message

```
feat: generate image via provider registry (OpenAI gpt-image-1)

Adds `sprite-gen generate image "<prompt>"` backed by a pluggable
provider interface. First provider: OpenAI gpt-image-1, via stdlib
net/http (no vendor SDK). Secrets loaded from env + .env via
godotenv; key is redacted from all output. httptest.Server covers
the provider; no live API calls in CI.
```

## Notes for the next plan

- Downstream plans (04+) can now pipe real generated PNGs into `inspect`, `palette extract`, etc., for test fixtures.
- Future `generate` sub-commands (`sheet`, `frames`, `palette`, `edit`, `variations`) and future providers are tracked in [`future-enhancements.md`](future-enhancements.md). Promote an entry from there to its own `NN-<name>.md` when it's the next-up plan.
- The provider-registry shape established here is intended to match the format-registry shape that `09-export-pipeline.md` will introduce — keep the two idioms aligned when that plan lands.
