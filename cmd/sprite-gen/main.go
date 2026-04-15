package main

import (
	"fmt"
	"io"
	"os"

	"github.com/kkjang/sprite-gen/internal/jsonout"
)

const version = "dev"

type handlerFunc func(args []string, stdout, stderr io.Writer, asJSON bool) error

var handlers = map[string]handlerFunc{}

func registerHandler(name string, h handlerFunc) {
	handlers[name] = h
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	args, asJSON := extractGlobalJSONFlag(args)
	if len(args) == 0 {
		_ = jsonout.WriteErr(stderr, asJSON, fmt.Errorf("missing command; try: sprite-gen spec"))
		return 1
	}

	h, ok := handlers[args[0]]
	if !ok {
		_ = jsonout.WriteErr(stderr, asJSON, fmt.Errorf("unknown command %q; try: sprite-gen spec", args[0]))
		return 1
	}

	if err := h(args[1:], stdout, stderr, asJSON); err != nil {
		_ = jsonout.WriteErr(stderr, asJSON, err)
		return 1
	}

	return 0
}

func extractGlobalJSONFlag(args []string) ([]string, bool) {
	filtered := make([]string, 0, len(args))
	asJSON := false
	for _, arg := range args {
		if arg == "--json" {
			asJSON = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, asJSON
}
