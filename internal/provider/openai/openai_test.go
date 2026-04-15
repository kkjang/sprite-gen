package openai

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kkjang/sprite-gen/internal/provider"
)

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aX9kAAAAASUVORK5CYII="

func TestProviderGenerateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("path = %q, want /v1/images/generations", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("Authorization = %q, want Bearer sk-test", got)
		}
		io.WriteString(w, `{"data":[{"b64_json":"`+tinyPNGBase64+`","revised_prompt":"pixel knight","seed":"42"}],"usage":{"input_tokens":12,"output_tokens":34}}`)
	}))
	defer server.Close()

	p := &Provider{BaseURL: server.URL, HTTPClient: server.Client(), APIKey: "sk-test"}
	resp, err := p.Generate(context.Background(), provider.Request{Prompt: "knight", Model: defaultModel, Size: "1024x1024", N: 1})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Model != defaultModel {
		t.Fatalf("model = %q, want %q", resp.Model, defaultModel)
	}
	if len(resp.Images) != 1 || len(resp.Images[0].Bytes) == 0 {
		t.Fatalf("images = %#v, want one decoded image", resp.Images)
	}
	if resp.Images[0].Revised != "pixel knight" || resp.Images[0].Seed != "42" {
		t.Fatalf("image metadata = %#v, want revised prompt and seed", resp.Images[0])
	}
	if resp.Usage == nil || resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 34 {
		t.Fatalf("usage = %#v, want token counts", resp.Usage)
	}
}

func TestProviderRetries429(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			io.WriteString(w, `{"error":{"message":"rate limited for sk-test"}}`)
			return
		}
		io.WriteString(w, `{"data":[{"b64_json":"`+tinyPNGBase64+`"}]}`)
	}))
	defer server.Close()

	p := &Provider{BaseURL: server.URL, HTTPClient: server.Client(), APIKey: "sk-test", sleep: func(_ time.Duration) {}, rand: rand.New(rand.NewSource(1))}
	_, err := p.Generate(context.Background(), provider.Request{Prompt: "knight", Model: defaultModel, Size: "1024x1024", N: 1})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestProviderRedactsAPIKeyOn401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"message":"bad key sk-secret"}}`)
	}))
	defer server.Close()

	p := &Provider{BaseURL: server.URL, HTTPClient: server.Client(), APIKey: "sk-secret"}
	_, err := p.Generate(context.Background(), provider.Request{Prompt: "knight", Model: defaultModel, Size: "1024x1024", N: 1})
	if err == nil {
		t.Fatal("Generate() error = nil, want non-nil")
	}
	if strings.Contains(err.Error(), "sk-secret") {
		t.Fatalf("error = %q, should redact API key", err)
	}
	if !strings.Contains(err.Error(), "***REDACTED***") {
		t.Fatalf("error = %q, want redacted marker", err)
	}
}

func TestProviderRetryExhaustedOn500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error":{"message":"server exploded"}}`)
	}))
	defer server.Close()

	var sleeps int
	p := &Provider{BaseURL: server.URL, HTTPClient: server.Client(), APIKey: "sk-test", sleep: func(_ time.Duration) { sleeps++ }, rand: rand.New(rand.NewSource(1))}
	_, err := p.Generate(context.Background(), provider.Request{Prompt: "knight", Model: defaultModel, Size: "1024x1024", N: 1})
	if err == nil {
		t.Fatal("Generate() error = nil, want non-nil")
	}
	if sleeps != 2 {
		t.Fatalf("sleep calls = %d, want 2", sleeps)
	}
	if !strings.Contains(err.Error(), "server exploded") {
		t.Fatalf("error = %q, want provider message", err)
	}
}

func TestProviderMalformedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"b64_json":"%%%"}]}`)
	}))
	defer server.Close()

	p := &Provider{BaseURL: server.URL, HTTPClient: server.Client(), APIKey: "sk-test"}
	_, err := p.Generate(context.Background(), provider.Request{Prompt: "knight", Model: defaultModel, Size: "1024x1024", N: 1})
	if err == nil {
		t.Fatal("Generate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "decode OpenAI image payload") {
		t.Fatalf("error = %q, want decode failure", err)
	}
}
