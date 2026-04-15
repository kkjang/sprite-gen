package specreg

import (
	"sort"
	"sync"
)

type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Args        []Arg  `json:"args,omitempty"`
	Flags       []Flag `json:"flags,omitempty"`
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

var defaultRegistry = newRegistry()

func Register(c Command) {
	defaultRegistry.Register(c)
}

func All() []Command {
	return defaultRegistry.All()
}

type registry struct {
	mu       sync.Mutex
	commands []Command
}

func newRegistry() *registry {
	return &registry{}
}

func (r *registry) Register(c Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands = append(r.commands, cloneCommand(c))
}

func (r *registry) All() []Command {
	r.mu.Lock()
	out := make([]Command, len(r.commands))
	for i, cmd := range r.commands {
		out[i] = cloneCommand(cmd)
	}
	r.mu.Unlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func cloneCommand(c Command) Command {
	cloned := c
	if c.Args != nil {
		cloned.Args = append([]Arg(nil), c.Args...)
	}
	if c.Flags != nil {
		cloned.Flags = append([]Flag(nil), c.Flags...)
	}
	return cloned
}
