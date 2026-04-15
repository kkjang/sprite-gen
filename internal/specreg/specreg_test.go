package specreg

import (
	"fmt"
	"sync"
	"testing"
)

func TestRegistryAllSorted(t *testing.T) {
	r := newRegistry()
	r.Register(Command{Name: "version", Description: "show version"})
	r.Register(Command{Name: "spec", Description: "show spec"})

	got := r.All()
	if len(got) != 2 {
		t.Fatalf("len(All()) = %d, want 2", len(got))
	}
	if got[0].Name != "spec" || got[1].Name != "version" {
		t.Fatalf("names = %#v, want [spec version]", []string{got[0].Name, got[1].Name})
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
			r.Register(Command{Name: fmt.Sprintf("cmd-%02d", i), Description: "test"})
		}(i)
	}
	wg.Wait()

	got := r.All()
	if len(got) != total {
		t.Fatalf("len(All()) = %d, want %d", len(got), total)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Name > got[i].Name {
			t.Fatalf("commands out of order: %q > %q", got[i-1].Name, got[i].Name)
		}
	}
}
