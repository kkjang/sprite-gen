package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/kkjang/sprite-gen/internal/provider"
	"github.com/kkjang/sprite-gen/internal/secrets"
)

const (
	defaultBaseURL = "https://api.openai.com"
	defaultModel   = "gpt-image-1"
)

var supportedModels = []string{defaultModel}

type Provider struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string

	sleep func(time.Duration)
	rand  *rand.Rand
}

func init() {
	provider.Register(&Provider{})
}

func (p *Provider) Name() string {
	return "openai"
}

func (p *Provider) Models() []string {
	return append([]string(nil), supportedModels...)
}

func (p *Provider) Sizes(model string) []string {
	return []string{"1024x1024", "1024x1536", "1536x1024", "auto"}
}

func (p *Provider) WithAPIKey(apiKey string) provider.ImageProvider {
	clone := *p
	clone.APIKey = apiKey
	return &clone
}

func (p *Provider) Generate(ctx context.Context, req provider.Request) (*provider.Response, error) {
	client := p.httpClient()
	body, err := json.Marshal(map[string]any{
		"model":  req.Model,
		"prompt": req.Prompt,
		"n":      req.N,
		"size":   req.Size,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal OpenAI request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := p.doRequest(ctx, client, body, req.Model)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt == 2 {
			break
		}
		p.sleepForAttempt(attempt)
	}
	return nil, lastErr
}

func (p *Provider) doRequest(ctx context.Context, client *http.Client, body []byte, model string) (*provider.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.baseURL(), "/")+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build OpenAI request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call OpenAI image API: %w", err)
	}
	defer httpResp.Body.Close()

	payload, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read OpenAI response: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		return nil, parseAPIError(httpResp.StatusCode, payload, p.APIKey)
	}

	var decoded struct {
		Data []struct {
			B64JSON       string `json:"b64_json"`
			RevisedPrompt string `json:"revised_prompt"`
			Seed          string `json:"seed"`
		} `json:"data"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode OpenAI response: %w", err)
	}

	resp := &provider.Response{Model: reqModelOrDefault(model)}
	if decoded.Usage != nil {
		resp.Usage = &provider.Usage{InputTokens: decoded.Usage.InputTokens, OutputTokens: decoded.Usage.OutputTokens}
	}
	for _, item := range decoded.Data {
		imageBytes, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decode OpenAI image payload: %w", err)
		}
		resp.Images = append(resp.Images, provider.Image{
			Bytes:   imageBytes,
			Format:  "png",
			Revised: item.RevisedPrompt,
			Seed:    item.Seed,
		})
	}
	return resp, nil
}

func (p *Provider) baseURL() string {
	if p.BaseURL != "" {
		return p.BaseURL
	}
	return defaultBaseURL
}

func (p *Provider) httpClient() *http.Client {
	if p.HTTPClient != nil {
		return p.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (p *Provider) sleepForAttempt(attempt int) {
	sleeper := p.sleep
	if sleeper == nil {
		sleeper = time.Sleep
	}
	rng := p.rand
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	base := 200 * time.Millisecond
	backoff := base * time.Duration(1<<attempt)
	jitter := time.Duration(rng.Intn(100)) * time.Millisecond
	sleeper(backoff + jitter)
}

type retryableError struct{ err error }

func (e retryableError) Error() string { return e.err.Error() }

func isRetryable(err error) bool {
	_, ok := err.(retryableError)
	return ok
}

func parseAPIError(statusCode int, payload []byte, apiKey string) error {
	var decoded struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &decoded); err == nil && decoded.Error != nil && decoded.Error.Message != "" {
		message := secrets.Redact(decoded.Error.Message, apiKey)
		if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
			return retryableError{err: fmt.Errorf("OpenAI API error (%d): %s", statusCode, message)}
		}
		return fmt.Errorf("OpenAI API error (%d): %s", statusCode, message)
	}
	message := secrets.Redact(strings.TrimSpace(string(payload)), apiKey)
	if message == "" {
		message = http.StatusText(statusCode)
	}
	if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		return retryableError{err: fmt.Errorf("OpenAI API error (%d): %s", statusCode, message)}
	}
	return fmt.Errorf("OpenAI API error (%d): %s", statusCode, message)
}

func reqModelOrDefault(model string) string {
	if model == "" {
		return defaultModel
	}
	return model
}
