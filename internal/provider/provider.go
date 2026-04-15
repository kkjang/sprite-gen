package provider

import "context"

type ImageProvider interface {
	Name() string
	Models() []string
	Sizes(model string) []string
	WithAPIKey(apiKey string) ImageProvider
	Generate(ctx context.Context, req Request) (*Response, error)
}

type Request struct {
	Prompt string
	Model  string
	Size   string
	N      int
	Extra  map[string]any
}

type Response struct {
	Images []Image
	Model  string
	Usage  *Usage
}

type Image struct {
	Bytes   []byte
	Format  string
	Revised string
	Seed    string
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}
