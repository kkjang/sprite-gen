package gif

import (
	"fmt"
	"image"
	"image/color"
	stdgif "image/gif"
	"math"
	"os"
	"path/filepath"
	"strconv"

	internalexport "github.com/kkjang/sprite-gen/internal/export"
	internalpalette "github.com/kkjang/sprite-gen/internal/palette"
	internalresize "github.com/kkjang/sprite-gen/internal/resize"
)

type GIF struct{}

func (GIF) Name() string {
	return "gif"
}

func (GIF) Description() string {
	return "Animated GIF preview (for visual verification)"
}

func (GIF) Export(ctx *internalexport.Context) (*internalexport.Result, error) {
	if len(ctx.Frames) == 0 {
		return nil, fmt.Errorf("export gif: no frames to export")
	}

	fps, err := positiveIntOption(ctx.Options, "fps", 8)
	if err != nil {
		return nil, err
	}
	scale, err := rangedIntOption(ctx.Options, "scale", 1, 1, 8)
	if err != nil {
		return nil, err
	}
	loop, err := boolOption(ctx.Options, "loop", true)
	if err != nil {
		return nil, err
	}

	delay := delayForFPS(fps)
	animation := &stdgif.GIF{
		Image:           make([]*image.Paletted, len(ctx.Frames)),
		Delay:           make([]int, len(ctx.Frames)),
		Disposal:        make([]byte, len(ctx.Frames)),
		BackgroundIndex: 0,
		LoopCount:       -1,
	}
	if loop {
		animation.LoopCount = 0
	}

	frameW := 0
	frameH := 0
	for i, frame := range ctx.Frames {
		img := frame.Image
		if scale > 1 {
			img, err = internalresize.Image(img, internalresize.Options{Direction: internalresize.Up, Factor: scale})
			if err != nil {
				return nil, err
			}
		}
		animation.Image[i] = paletted(img)
		animation.Delay[i] = delay
		// Clear transparent pixels from earlier frames in animated previews.
		animation.Disposal[i] = stdgif.DisposalBackground
		if w := img.Bounds().Dx(); w > frameW {
			frameW = w
		}
		if h := img.Bounds().Dy(); h > frameH {
			frameH = h
		}
	}
	animation.Config.Width = frameW
	animation.Config.Height = frameH

	verb := "wrote"
	if ctx.DryRun {
		verb = "would write"
	} else if err := writeGIF(ctx.OutPath, animation); err != nil {
		return nil, err
	}

	durationMS := len(ctx.Frames) * delay * 10
	data := map[string]any{
		"format":         ctx.Format,
		"out":            ctx.OutPath,
		"frames":         len(ctx.Frames),
		"fps":            fps,
		"frame_delay_cs": delay,
		"duration_ms":    durationMS,
		"loop":           loop,
		"scale":          scale,
		"frame_w":        frameW,
		"frame_h":        frameH,
		"dry_run":        ctx.DryRun,
	}
	text := fmt.Sprintf("%s: %s (%d frames, %d fps target, %dms total)\n", verb, ctx.OutPath, len(ctx.Frames), fps, durationMS)
	return &internalexport.Result{Text: text, Data: data}, nil
}

func init() {
	internalexport.Register(GIF{})
}

func positiveIntOption(options map[string]string, key string, defaultValue int) (int, error) {
	raw := options[key]
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid --%s value %q; want a positive integer", key, raw)
	}
	return value, nil
}

func rangedIntOption(options map[string]string, key string, defaultValue, minValue, maxValue int) (int, error) {
	value, err := positiveIntOption(options, key, defaultValue)
	if err != nil {
		return 0, err
	}
	if value < minValue || value > maxValue {
		return 0, fmt.Errorf("invalid --%s value %d; want %d-%d", key, value, minValue, maxValue)
	}
	return value, nil
}

func boolOption(options map[string]string, key string, defaultValue bool) (bool, error) {
	raw := options[key]
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid --%s value %q; want true or false", key, raw)
	}
	return value, nil
}

func delayForFPS(fps int) int {
	delay := int(math.Round(100.0 / float64(fps)))
	if delay < 1 {
		return 1
	}
	return delay
}

func writeGIF(path string, animation *stdgif.GIF) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory for %q: %w", path, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create GIF %q: %w", path, err)
	}
	defer f.Close()

	if err := stdgif.EncodeAll(f, animation); err != nil {
		return fmt.Errorf("encode GIF %q: %w", path, err)
	}
	return nil
}

func paletted(img *image.NRGBA) *image.Paletted {
	bounds := img.Bounds()
	visiblePalette := internalpalette.Extract(img, 255)
	stdPalette := make(color.Palette, 1, len(visiblePalette)+1)
	stdPalette[0] = color.NRGBA{}
	indexByRGB := map[[3]uint8]uint8{}
	for _, c := range visiblePalette {
		key := [3]uint8{c.R, c.G, c.B}
		if _, exists := indexByRGB[key]; exists {
			continue
		}
		indexByRGB[key] = uint8(len(stdPalette))
		c.A = 255
		stdPalette = append(stdPalette, c)
	}

	out := image.NewPaletted(image.Rect(0, 0, bounds.Dx(), bounds.Dy()), stdPalette)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.A == 0 {
				out.SetColorIndex(x-bounds.Min.X, y-bounds.Min.Y, 0)
				continue
			}
			if len(visiblePalette) == 0 {
				out.SetColorIndex(x-bounds.Min.X, y-bounds.Min.Y, 0)
				continue
			}
			snapped := internalpalette.Snap(c, visiblePalette)
			idx := indexByRGB[[3]uint8{snapped.R, snapped.G, snapped.B}]
			out.SetColorIndex(x-bounds.Min.X, y-bounds.Min.Y, idx)
		}
	}
	return out
}
