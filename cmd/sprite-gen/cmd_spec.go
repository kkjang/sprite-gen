package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

func init() {
	registerHandler("spec", runSpec)
	specreg.Register(specreg.Command{
		Name:        "spec",
		Description: "Print the command registry",
		Flags: []specreg.Flag{
			{Name: "markdown", Description: "Print the registry as markdown"},
		},
	})
}

func runSpec(args []string, stdout, _ io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("spec", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	markdown := fs.Bool("markdown", false, "print markdown output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("spec takes no positional arguments")
	}

	commands := specreg.All()
	if asJSON {
		if *markdown {
			return jsonout.Write(stdout, true, "", map[string]string{"markdown": renderSpecMarkdown(commands)})
		}
		return jsonout.Write(stdout, true, "", commands)
	}

	if *markdown {
		_, err := io.WriteString(stdout, renderSpecMarkdown(commands))
		return err
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(commands)
}

func renderSpecMarkdown(commands []specreg.Command) string {
	var b strings.Builder
	b.WriteString("| Command | Description |\n")
	b.WriteString("|---|---|\n")
	for _, cmd := range commands {
		fmt.Fprintf(&b, "| `%s` | %s |\n", cmd.Name, cmd.Description)
	}
	return b.String()
}
