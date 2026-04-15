# Plan 01 — Skeleton

## Goal

Ship the absolute minimum that proves the dispatch, JSON envelope, and spec-registry architecture work end-to-end. No image processing yet. Two commands: `version` and `spec`. This PR should be small enough to review in ten minutes.

## Scope

**In:**
- `go.mod` with module path `github.com/<owner>/sprite-gen`
- Single binary at `cmd/sprite-gen/main.go` that dispatches subcommands
- `sprite-gen version` — prints hardcoded version string
- `sprite-gen spec` — prints the machine-readable command registry as JSON (or markdown with `--markdown`)
- `internal/jsonout/` — the `{ok, data, error}` envelope writer
- `internal/specreg/` — the registry that `spec` reads
- Minimal test coverage: envelope shape, spec includes both commands, exit codes
- Top-level `README.md` with install + build + test instructions
- Top-level `AGENTS.md` with conventions agents need to follow

**Out:**
- Any image processing code
- Any external dependencies beyond stdlib
- Any CI configuration (comes in plan 02)
- Any release tooling (comes in plan 02)

## File plan

```
sprite-gen/
  go.mod                            # module declaration, go 1.22
  README.md                         # install + test
  AGENTS.md                         # conventions, envelope shape, registry pattern
  cmd/sprite-gen/
    main.go                         # main(), global flags, dispatch to cmd_*
    cmd_version.go                  # version handler + spec registration
    cmd_spec.go                     # spec handler + spec registration
    main_test.go                    # e2e tests via main() dispatch
  internal/
    jsonout/
      envelope.go                   # Envelope struct, Write/WriteErr helpers
      envelope_test.go
    specreg/
      specreg.go                    # Command, Flag, Arg structs + Register/All
      specreg_test.go
```

## Interfaces

### `internal/jsonout`

```go
type Envelope struct {
    OK    bool        `json:"ok"`
    Data  any         `json:"data,omitempty"`
    Error string      `json:"error,omitempty"`
}

// Write prints text to w.text or JSON to w.json depending on the --json flag
// captured in the caller. Signature sketch:
func Write(w io.Writer, asJSON bool, text string, data any) error
func WriteErr(w io.Writer, asJSON bool, err error) error
```

The pattern matches [`godot-bridge/cli/cmd/godot-bridge/main.go`](https://github.com/kkjang/godot-bridge/blob/main/cli/cmd/godot-bridge/main.go) (see `writeJSON`, `writeJSONLine`). Copy the shape, not the exact code — this repo has no transport layer, so it can be simpler.

### `internal/specreg`

```go
type Command struct {
    Name        string   `json:"name"`        // e.g. "inspect sheet"
    Description string   `json:"description"`
    Args        []Arg    `json:"args,omitempty"`
    Flags       []Flag   `json:"flags,omitempty"`
}

type Arg struct {
    Name        string `json:"name"`
    Required    bool   `json:"required"`
    Description string `json:"description"`
}

type Flag struct {
    Name        string `json:"name"`
    Default     string `json:"default,omitempty"`
    Description string `json:"description"`
}

// Register is called from init() in each cmd_*.go file.
func Register(c Command)

// All returns the registered commands in stable sorted order.
func All() []Command
```

Each `cmd_*.go` has an `init()` that calls `specreg.Register(...)`. The `cmd_spec.go` handler reads `specreg.All()` and emits it.

## Dispatch pattern (main.go)

```go
func main() {
    os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
    // Parse global flags (--json)
    // args[0] is the subject verb or command name
    // Dispatch via a map[string]handlerFunc populated in init()
    // Return exit code
}

type handlerFunc func(args []string, stdout, stderr io.Writer, asJSON bool) error

var handlers = map[string]handlerFunc{}

func registerHandler(name string, h handlerFunc) { handlers[name] = h }
```

Each `cmd_*.go` file registers its handler in `init()`. `main.go` stays tiny — no command-specific logic.

## Testing

In `main_test.go`:

- Call `run([]string{"version"}, ...)` and assert stdout contains the version string and exit code is 0.
- Call `run([]string{"version", "--json"}, ...)` and assert stdout parses as an `Envelope` with `ok: true` and `data.version` non-empty.
- Call `run([]string{"spec"}, ...)` and assert the JSON output includes at least `"version"` and `"spec"` commands.
- Call `run([]string{"bogus"}, ...)` and assert exit code is non-zero and stderr has an actionable message (e.g. `unknown command "bogus"; try: sprite-gen spec`).

In `jsonout/envelope_test.go`:

- `Write` with `asJSON=false` produces the plain text.
- `Write` with `asJSON=true` produces a valid JSON envelope containing the data.
- `WriteErr` with `asJSON=true` produces `{"ok": false, "error": "..."}` and returns the error.

In `specreg/specreg_test.go`:

- `Register` + `All` round-trips commands in sorted order.
- Concurrent `Register` calls are safe (use a mutex).

## Acceptance criteria

1. `go test ./...` passes from the repo root.
2. `go build -o /tmp/sprite-gen ./cmd/sprite-gen && /tmp/sprite-gen version` prints a version line.
3. `/tmp/sprite-gen spec` prints JSON with exactly two commands.
4. `/tmp/sprite-gen spec --markdown` prints a markdown table with two rows (version + spec). This can be a one-liner that iterates `specreg.All()`.
5. `/tmp/sprite-gen bogus` exits non-zero with a helpful message.
6. No non-stdlib imports anywhere.

## Suggested commit message

```
feat: initial skeleton — dispatch, envelope, spec registry

Minimum viable CLI shell with two commands (version, spec) wired
through a shared JSON envelope and a spec registry that each command
self-registers into. Later plans add real image processing.
```

## Notes for the next plan

- The `releases.yaml` file and GitHub Actions workflows come in plan 02. Do not add them here.
- Avoid the temptation to stub in package directories for `pixel/`, `palette/`, etc. — add them when the plans that need them land. Empty packages add noise to review.
- Keep `README.md` short. Just install + test. A full feature table would be premature.
