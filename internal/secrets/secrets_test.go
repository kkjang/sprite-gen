package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrefersEnvThenLocalThenDotEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	t.Setenv("SPRITE_GEN_OPENAI_API_KEY", "")
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env"), "OPENAI_API_KEY=dotenv-key\nSPRITE_GEN_OPENAI_API_KEY=dotenv-scoped\n")
	writeFile(t, filepath.Join(dir, ".env.local"), "OPENAI_API_KEY=local-key\nSPRITE_GEN_OPENAI_API_KEY=local-scoped\n")
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	got, err := Load("SPRITE_GEN_OPENAI_API_KEY", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != "local-scoped" {
		t.Fatalf("Load() = %q, want local-scoped", got)
	}

	t.Setenv("SPRITE_GEN_OPENAI_API_KEY", "env-scoped")
	got, err = Load("SPRITE_GEN_OPENAI_API_KEY", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != "env-scoped" {
		t.Fatalf("Load() = %q, want env-scoped", got)
	}
}

func TestLoadMissingFilesIsNoop(t *testing.T) {
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if _, err := Load("OPENAI_API_KEY"); err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestRedactScrubsExactMatchesOnly(t *testing.T) {
	got := Redact("key=sk-secret; keep sk-secret-2", "sk-secret")
	if got != "key=***REDACTED***; keep ***REDACTED***-2" {
		t.Fatalf("Redact() = %q", got)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
