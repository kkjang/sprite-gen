package provider

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var defaultRegistry = newRegistry()

func Register(p ImageProvider) {
	defaultRegistry.Register(p)
}

func Get(name string) (ImageProvider, error) {
	return defaultRegistry.Get(name)
}

func Names() []string {
	return defaultRegistry.Names()
}

type registry struct {
	mu        sync.RWMutex
	providers map[string]ImageProvider
}

func newRegistry() *registry {
	return &registry{providers: map[string]ImageProvider{}}
}

func (r *registry) Register(p ImageProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *registry) Get(name string) (ImageProvider, error) {
	r.mu.RLock()
	p, ok := r.providers[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown provider %q; try one of: %s", name, joinNames(r.Names()))
	}
	return p, nil
}

func (r *registry) Names() []string {
	r.mu.RLock()
	out := make([]string, 0, len(r.providers))
	for name := range r.providers {
		out = append(out, name)
	}
	r.mu.RUnlock()
	sort.Strings(out)
	return out
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, fmt.Sprintf("%q", name))
	}
	return strings.Join(quoted, ", ")
}
