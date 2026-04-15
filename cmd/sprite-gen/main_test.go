package main

import (
	"bytes"
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func TestRunVersion(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})
	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"version"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if got := stdout.String(); !strings.Contains(got, version) {
		t.Fatalf("stdout = %q, want version %q", got, version)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersionJSON(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})
	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"version", "--json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}

	var got jsonout.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !got.OK {
		t.Fatalf("envelope ok = false, want true")
	}
	data, ok := got.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T, want map[string]any", got.Data)
	}
	if data["version"] == "" {
		t.Fatalf("data.version = %v, want non-empty", data["version"])
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersionUsesBuildInfoVersion(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})
	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v0.1.1"}}, true
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"version"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if got := stdout.String(); !strings.Contains(got, "v0.1.1") {
		t.Fatalf("stdout = %q, want build info version", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersionPrefersInjectedVersion(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})
	version = "v9.9.9"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v0.1.1"}}, true
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"version"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if got := stdout.String(); !strings.Contains(got, "v9.9.9") {
		t.Fatalf("stdout = %q, want injected version", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunSpec(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"spec"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}

	var got []specreg.Command
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("spec command count = %d, want 3", len(got))
	}
	if got[0].Name != "generate image" || got[1].Name != "spec" || got[2].Name != "version" {
		t.Fatalf("spec command names = %#v, want [generate image spec version]", []string{got[0].Name, got[1].Name, got[2].Name})
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"bogus"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("run() exit code = %d, want non-zero", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); !strings.Contains(got, `unknown command "bogus"; try: sprite-gen spec`) {
		t.Fatalf("stderr = %q, want actionable unknown command message", got)
	}
}
