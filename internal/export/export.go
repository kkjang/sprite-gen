package export

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Result struct {
	Text string
	Data any
}

type Format interface {
	Name() string
	Description() string
	Export(ctx *Context) (*Result, error)
}

var defaultRegistry = newRegistry()

func Register(f Format) {
	defaultRegistry.Register(f)
}

func Get(name string) (Format, error) {
	return defaultRegistry.Get(name)
}

func All() []Format {
	return defaultRegistry.All()
}

type registry struct {
	mu      sync.Mutex
	formats map[string]Format
}

func newRegistry() *registry {
	return &registry{formats: map[string]Format{}}
}

func (r *registry) Register(f Format) {
	name := strings.TrimSpace(f.Name())
	if name == "" {
		panic("export: empty format name")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.formats[name]; exists {
		panic("export: duplicate format: " + name)
	}
	r.formats[name] = f
}

func (r *registry) Get(name string) (Format, error) {
	r.mu.Lock()
	f, ok := r.formats[name]
	r.mu.Unlock()
	if ok {
		return f, nil
	}

	available := r.names()
	if len(available) == 0 {
		return nil, fmt.Errorf("unknown export format %q; no formats are registered", name)
	}
	return nil, fmt.Errorf("unknown export format %q; available formats: %s", name, strings.Join(available, ", "))
}

func (r *registry) All() []Format {
	r.mu.Lock()
	out := make([]Format, 0, len(r.formats))
	for _, f := range r.formats {
		out = append(out, f)
	}
	r.mu.Unlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

func (r *registry) names() []string {
	formats := r.All()
	names := make([]string, len(formats))
	for i, f := range formats {
		names[i] = f.Name()
	}
	return names
}
