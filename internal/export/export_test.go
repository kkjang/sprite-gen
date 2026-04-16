package export

import (
	"fmt"
	"testing"
)

type testFormat struct {
	name        string
	description string
}

func (f testFormat) Name() string {
	return f.name
}

func (f testFormat) Description() string {
	return f.description
}

func (f testFormat) Export(*Context) (*Result, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestRegistryRegisterGetRoundTrip(t *testing.T) {
	r := newRegistry()
	format := testFormat{name: "gif", description: "Animated GIF"}
	r.Register(format)

	got, err := r.Get("gif")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != format.Name() {
		t.Fatalf("Get().Name() = %q, want %q", got.Name(), format.Name())
	}
}

func TestRegistryRegisterDuplicatePanics(t *testing.T) {
	r := newRegistry()
	r.Register(testFormat{name: "gif"})

	defer func() {
		if recover() == nil {
			t.Fatal("Register() panic = nil, want duplicate panic")
		}
	}()
	r.Register(testFormat{name: "gif"})
}

func TestRegistryAllSorted(t *testing.T) {
	r := newRegistry()
	r.Register(testFormat{name: "sheet-png"})
	r.Register(testFormat{name: "gif"})

	got := r.All()
	if len(got) != 2 {
		t.Fatalf("len(All()) = %d, want 2", len(got))
	}
	if got[0].Name() != "gif" || got[1].Name() != "sheet-png" {
		t.Fatalf("All() names = [%q, %q], want ["+"gif"+", "+"sheet-png"+"]", got[0].Name(), got[1].Name())
	}
}

func TestRegistryGetUnknownListsAvailableFormats(t *testing.T) {
	r := newRegistry()
	r.Register(testFormat{name: "gif"})
	r.Register(testFormat{name: "sheet-png"})

	_, err := r.Get("bogus")
	if err == nil {
		t.Fatal("Get() error = nil, want unknown format error")
	}
	if got := err.Error(); got != `unknown export format "bogus"; available formats: gif, sheet-png` {
		t.Fatalf("Get() error = %q, want actionable available format list", got)
	}
}
