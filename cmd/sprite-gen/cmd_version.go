package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("version", runVersion)
	specreg.Register(specreg.Command{
		Name:        "version",
		Description: "Print the sprite-gen version",
	})
}

func runVersion(args []string, stdout, _ io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("version takes no positional arguments")
	}

	text := fmt.Sprintf("sprite-gen %s\n", version)
	data := map[string]string{"version": version}
	return jsonout.Write(stdout, asJSON, text, data)
}
