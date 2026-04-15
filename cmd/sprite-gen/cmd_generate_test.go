package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/provider"
)

type stubProvider struct {
	response *provider.Response
	err      error
	req      provider.Request
	apiKey   string
}

func (p *stubProvider) Name() string                                    { return "stub" }
func (p *stubProvider) Models() []string                                { return []string{"stub-model"} }
func (p *stubProvider) Sizes(string) []string                           { return []string{"1024x1024"} }
func (p *stubProvider) WithAPIKey(apiKey string) provider.ImageProvider { p.apiKey = apiKey; return p }
func (p *stubProvider) Generate(_ context.Context, req provider.Request) (*provider.Response, error) {
	p.req = req
	if p.err != nil {
		return nil, p.err
	}
	return p.response, nil
}

func TestRunGenerateImageDryRunJSON(t *testing.T) {
	originalGetProvider := getProvider
	originalLoadSecret := loadSecret
	t.Cleanup(func() {
		getProvider = originalGetProvider
		loadSecret = originalLoadSecret
	})
	getProvider = func(name string) (provider.ImageProvider, error) { return &stubProvider{}, nil }
	loadSecret = func(keys ...string) (string, error) {
		t.Fatalf("loadSecret should not be called during dry-run")
		return "", nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"generate", "image", "pixel knight", "--provider", "stub", "--model", "stub-model", "--dry-run", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	data := got.Data.(map[string]any)
	if data["provider"] != "stub" {
		t.Fatalf("provider = %v, want stub", data["provider"])
	}
	images := data["images"].([]any)
	if !strings.Contains(images[0].(map[string]any)["path"].(string), filepath.Join("out", "generate")) {
		t.Fatalf("image path = %v, want default output path", images[0])
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunGenerateImageMissingKey(t *testing.T) {
	originalGetProvider := getProvider
	originalLoadSecret := loadSecret
	t.Cleanup(func() {
		getProvider = originalGetProvider
		loadSecret = originalLoadSecret
	})
	getProvider = func(name string) (provider.ImageProvider, error) { return &stubProvider{}, nil }
	loadSecret = func(keys ...string) (string, error) { return "", errors.New("missing") }

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"generate", "image", "pixel knight", "--provider", "stub", "--model", "stub-model"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "no OpenAI credentials") {
		t.Fatalf("stderr = %q, want actionable credential message", stderr.String())
	}
}

func TestRunGenerateImageRedactsProviderError(t *testing.T) {
	originalGetProvider := getProvider
	originalLoadSecret := loadSecret
	t.Cleanup(func() {
		getProvider = originalGetProvider
		loadSecret = originalLoadSecret
	})
	getProvider = func(name string) (provider.ImageProvider, error) {
		return &stubProvider{err: errors.New("bad key sk-secret")}, nil
	}
	loadSecret = func(keys ...string) (string, error) { return "sk-secret", nil }

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"generate", "image", "pixel knight", "--provider", "stub", "--model", "stub-model"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if strings.Contains(stderr.String(), "sk-secret") {
		t.Fatalf("stderr = %q, should redact API key", stderr.String())
	}
}

func TestRunGenerateImageWritesFiles(t *testing.T) {
	originalGetProvider := getProvider
	originalLoadSecret := loadSecret
	t.Cleanup(func() {
		getProvider = originalGetProvider
		loadSecret = originalLoadSecret
	})

	stub := &stubProvider{response: &provider.Response{Model: "stub-model", Images: []provider.Image{{Bytes: []byte("png-a")}, {Bytes: []byte("png-b")}}}}
	getProvider = func(name string) (provider.ImageProvider, error) { return stub, nil }
	loadSecret = func(keys ...string) (string, error) { return "sk-test", nil }

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"generate", "image", "pixel knight", "--provider", "stub", "--model", "stub-model", "--n", "2", "--out", "custom"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if stub.apiKey != "sk-test" {
		t.Fatalf("apiKey = %q, want injected key", stub.apiKey)
	}
	for _, rel := range []string{filepath.Join("custom", "image-000.png"), filepath.Join("custom", "image-001.png")} {
		payload, err := os.ReadFile(rel)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", rel, err)
		}
		if len(payload) == 0 {
			t.Fatalf("file %q is empty", rel)
		}
	}
	if !strings.Contains(stdout.String(), filepath.Join("custom", "image-000.png")) {
		t.Fatalf("stdout = %q, want written paths", stdout.String())
	}
}
