package jsonout

import (
	"encoding/json"
	"io"
)

type Envelope struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func Write(w io.Writer, asJSON bool, text string, data any) error {
	if !asJSON {
		_, err := io.WriteString(w, text)
		return err
	}
	return json.NewEncoder(w).Encode(Envelope{OK: true, Data: data})
}

func WriteErr(w io.Writer, asJSON bool, err error) error {
	if !asJSON {
		_, writeErr := io.WriteString(w, err.Error()+"\n")
		if writeErr != nil {
			return writeErr
		}
		return err
	}
	encodeErr := json.NewEncoder(w).Encode(Envelope{OK: false, Error: err.Error()})
	if encodeErr != nil {
		return encodeErr
	}
	return err
}
