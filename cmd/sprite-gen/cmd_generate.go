package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kkjang/sprite-gen/internal/jsonout"
	"github.com/kkjang/sprite-gen/internal/provider"
	_ "github.com/kkjang/sprite-gen/internal/provider/openai"
	"github.com/kkjang/sprite-gen/internal/secrets"
	"github.com/kkjang/sprite-gen/internal/specreg"
)

var (
	loadSecret               = secrets.Load
	getProvider              = provider.Get
	commandStdin   io.Reader = os.Stdin
	writeFile                = os.WriteFile
	mkdirAll                 = os.MkdirAll
	readPromptFile           = os.ReadFile
)

func init() {
	registerHandler("generate", runGenerate)
	specreg.Register(specreg.Command{
		Name:        "generate image",
		Description: "Generate one or more PNGs from a prompt",
		Args: []specreg.Arg{
			{Name: "prompt", Required: false, Description: "Prompt text when --prompt-file is not used"},
		},
		Flags: []specreg.Flag{
			{Name: "provider", Default: "openai", Description: "Image provider name"},
			{Name: "model", Default: "gpt-image-1", Description: "Provider model name"},
			{Name: "size", Default: "1024x1024", Description: "Requested image size"},
			{Name: "n", Default: "1", Description: "Number of images to generate"},
			{Name: "out", Description: "Output file for one image or directory for many images"},
			{Name: "prompt-file", Description: "Read the prompt from a file path or - for stdin"},
			{Name: "dry-run", Default: "false", Description: "Validate and print the request without calling the provider"},
		},
	})
}

func runGenerate(args []string, stdout, stderr io.Writer, asJSON bool) error {
	if len(args) == 0 {
		return fmt.Errorf("missing generate subcommand; try: sprite-gen spec")
	}
	if args[0] != "image" {
		return fmt.Errorf("unknown generate subcommand %q; try: sprite-gen spec", args[0])
	}
	return runGenerateImage(args[1:], stdout, stderr, asJSON)
}

func runGenerateImage(args []string, stdout, _ io.Writer, asJSON bool) error {
	fs := flag.NewFlagSet("generate image", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	providerName := fs.String("provider", "openai", "image provider name")
	model := fs.String("model", "gpt-image-1", "provider model name")
	size := fs.String("size", "1024x1024", "requested image size")
	n := fs.Int("n", 1, "number of images to generate")
	out := fs.String("out", "", "output file path for one image or directory for many images")
	promptFile := fs.String("prompt-file", "", "read the prompt from a file path or - for stdin")
	dryRun := fs.Bool("dry-run", false, "validate and print the request without calling the provider")
	flagArgs, positionalArgs, err := splitGenerateArgs(args)
	if err != nil {
		return err
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	prompt, err := resolvePrompt(positionalArgs, *promptFile)
	if err != nil {
		return err
	}
	if *n < 1 {
		return fmt.Errorf("--n must be at least 1")
	}

	p, err := getProvider(*providerName)
	if err != nil {
		return err
	}
	if !slices.Contains(p.Models(), *model) {
		return fmt.Errorf("invalid model %q for provider %q; try one of: %s", *model, *providerName, strings.Join(p.Models(), ", "))
	}
	if !slices.Contains(p.Sizes(*model), *size) {
		return fmt.Errorf("invalid --size %q for provider %q model %q; try one of: %s", *size, *providerName, *model, strings.Join(p.Sizes(*model), ", "))
	}

	req := provider.Request{Prompt: prompt, Model: *model, Size: *size, N: *n}
	if *dryRun {
		return writeGenerateResult(stdout, asJSON, generateResultFromRequest(*providerName, req, *out))
	}

	apiKey, err := loadSecret("SPRITE_GEN_OPENAI_API_KEY", "OPENAI_API_KEY")
	if err != nil {
		return fmt.Errorf("no OpenAI credentials; set OPENAI_API_KEY in your env or ./.env (see README)")
	}

	resp, err := p.WithAPIKey(apiKey).Generate(context.Background(), req)
	if err != nil {
		return fmt.Errorf("%s", secrets.Redact(err.Error(), apiKey))
	}

	result := generateResultFromResponse(*providerName, req, *out, resp)
	for _, image := range result.Images {
		if err := mkdirAll(filepath.Dir(image.Path), 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}
	for idx, image := range resp.Images {
		if err := writeFile(result.Images[idx].Path, image.Bytes, 0o644); err != nil {
			return fmt.Errorf("write image %q: %w", result.Images[idx].Path, err)
		}
	}

	return writeGenerateResult(stdout, asJSON, result)
}

type generateResult struct {
	Provider string                `json:"provider"`
	Model    string                `json:"model"`
	Size     string                `json:"size"`
	Prompt   string                `json:"prompt"`
	DryRun   bool                  `json:"dry_run,omitempty"`
	Images   []generateResultImage `json:"images"`
}

type generateResultImage struct {
	Path          string `json:"path"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	Seed          string `json:"seed,omitempty"`
}

func resolvePrompt(args []string, promptFile string) (string, error) {
	if promptFile != "" && len(args) != 0 {
		return "", fmt.Errorf("prompt must come from either a positional argument or --prompt-file, not both")
	}
	if promptFile == "" {
		if len(args) != 1 {
			return "", fmt.Errorf("generate image requires exactly one prompt or --prompt-file")
		}
		prompt := strings.TrimSpace(args[0])
		if prompt == "" {
			return "", fmt.Errorf("prompt must not be empty")
		}
		return prompt, nil
	}
	if promptFile == "-" {
		payload, err := io.ReadAll(commandStdin)
		if err != nil {
			return "", fmt.Errorf("read prompt from stdin: %w", err)
		}
		prompt := strings.TrimSpace(string(payload))
		if prompt == "" {
			return "", fmt.Errorf("prompt file was empty")
		}
		return prompt, nil
	}
	payload, err := readPromptFile(promptFile)
	if err != nil {
		return "", fmt.Errorf("read prompt file %q: %w", promptFile, err)
	}
	prompt := strings.TrimSpace(string(payload))
	if prompt == "" {
		return "", fmt.Errorf("prompt file was empty")
	}
	return prompt, nil
}

func generateResultFromRequest(providerName string, req provider.Request, out string) generateResult {
	result := generateResult{Provider: providerName, Model: req.Model, Size: req.Size, Prompt: req.Prompt, DryRun: true}
	for i := 0; i < req.N; i++ {
		result.Images = append(result.Images, generateResultImage{Path: outputPath(req.Prompt, req.N, i, out)})
	}
	return result
}

func generateResultFromResponse(providerName string, req provider.Request, out string, resp *provider.Response) generateResult {
	result := generateResult{Provider: providerName, Model: resp.Model, Size: req.Size, Prompt: req.Prompt}
	for i, image := range resp.Images {
		result.Images = append(result.Images, generateResultImage{
			Path:          outputPath(req.Prompt, req.N, i, out),
			RevisedPrompt: image.Revised,
			Seed:          image.Seed,
		})
	}
	return result
}

func outputPath(prompt string, total, idx int, out string) string {
	if out != "" {
		if total == 1 {
			return out
		}
		return filepath.Join(out, fmt.Sprintf("image-%03d.png", idx))
	}
	stem := promptStem(prompt)
	return filepath.Join("out", "generate", stem, fmt.Sprintf("image-%03d.png", idx))
}

func promptStem(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])[:12]
}

func writeGenerateResult(stdout io.Writer, asJSON bool, result generateResult) error {
	if asJSON {
		return jsonout.Write(stdout, true, "", result)
	}
	var b strings.Builder
	for _, image := range result.Images {
		b.WriteString(image.Path)
		b.WriteByte('\n')
	}
	return jsonout.Write(stdout, false, b.String(), result)
}

func splitGenerateArgs(args []string) ([]string, []string, error) {
	valueFlags := map[string]bool{
		"--provider":    true,
		"--model":       true,
		"--size":        true,
		"--n":           true,
		"--out":         true,
		"--prompt-file": true,
	}

	var flagArgs []string
	var positionalArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case valueFlags[arg]:
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag needs an argument: %s", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case hasValueFlag(arg, valueFlags), arg == "--dry-run":
			flagArgs = append(flagArgs, arg)
		case strings.HasPrefix(arg, "-"):
			flagArgs = append(flagArgs, arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return flagArgs, positionalArgs, nil
}

func hasValueFlag(arg string, valueFlags map[string]bool) bool {
	name, _, ok := strings.Cut(arg, "=")
	return ok && valueFlags[name]
}
