package provider

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

type testProvider struct{ name string }

func (p testProvider) Name() string                                         { return p.name }
func (p testProvider) Models() []string                                     { return []string{"model-a"} }
func (p testProvider) Sizes(string) []string                                { return []string{"1024x1024"} }
func (p testProvider) WithAPIKey(string) ImageProvider                      { return p }
func (p testProvider) Generate(context.Context, Request) (*Response, error) { return &Response{}, nil }

func TestRegistryRoundTrip(t *testing.T) {
	r := newRegistry()
	r.Register(testProvider{name: "beta"})
	r.Register(testProvider{name: "alpha"})

	got, err := r.Get("alpha")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != "alpha" {
		t.Fatalf("provider name = %q, want alpha", got.Name())
	}

	names := r.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Fatalf("Names() = %#v, want [alpha beta]", names)
	}
}

func TestRegistryConcurrentRegister(t *testing.T) {
	r := newRegistry()
	const total = 32
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func(i int) {
			defer wg.Done()
			r.Register(testProvider{name: fmt.Sprintf("provider-%02d", i)})
		}(i)
	}
	wg.Wait()

	if got := len(r.Names()); got != total {
		t.Fatalf("len(Names()) = %d, want %d", got, total)
	}
}
