package pixel

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

// SavePNG encodes img to path, creating parent directories as needed.
func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory for %q: %w", path, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create PNG %q: %w", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encode PNG %q: %w", path, err)
	}
	return nil
}
