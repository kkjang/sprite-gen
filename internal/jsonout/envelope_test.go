package jsonout

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestWriteText(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, false, "hello\n", map[string]string{"ignored": "value"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got := buf.String(); got != "hello\n" {
		t.Fatalf("buf = %q, want %q", got, "hello\n")
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, true, "ignored", map[string]string{"version": "dev"}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	var got Envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !got.OK {
		t.Fatalf("ok = false, want true")
	}
	data, ok := got.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T, want map[string]any", got.Data)
	}
	if data["version"] != "dev" {
		t.Fatalf("data.version = %v, want dev", data["version"])
	}
}

func TestWriteErrJSON(t *testing.T) {
	var buf bytes.Buffer
	wantErr := errors.New("boom")
	err := WriteErr(&buf, true, wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("WriteErr() error = %v, want %v", err, wantErr)
	}

	var got Envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.OK {
		t.Fatalf("ok = true, want false")
	}
	if got.Error != "boom" {
		t.Fatalf("error = %q, want %q", got.Error, "boom")
	}
}
