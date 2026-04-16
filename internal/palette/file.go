package palette

import (
	"bufio"
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"
)

func ReadHex(r io.Reader) ([]color.NRGBA, error) {
	s := bufio.NewScanner(r)
	var pal []color.NRGBA
	for lineNo := 1; s.Scan(); lineNo++ {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}

		token := strings.Fields(line)[0]
		c, ok, err := parseHexColor(token)
		if err != nil {
			return nil, fmt.Errorf("read .hex line %d: %w", lineNo, err)
		}
		if ok {
			pal = append(pal, c)
			continue
		}
		if strings.HasPrefix(token, "#") {
			continue
		}
		return nil, fmt.Errorf("read .hex line %d: want #rrggbb", lineNo)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read .hex palette: %w", err)
	}
	if len(pal) == 0 {
		return nil, fmt.Errorf("read .hex palette: no colors found")
	}
	return pal, nil
}

func WriteHex(w io.Writer, pal []color.NRGBA) error {
	for _, c := range pal {
		if _, err := fmt.Fprintf(w, "#%02x%02x%02x\n", c.R, c.G, c.B); err != nil {
			return fmt.Errorf("write .hex palette: %w", err)
		}
	}
	return nil
}

func ReadGPL(r io.Reader) ([]color.NRGBA, error) {
	s := bufio.NewScanner(r)
	var pal []color.NRGBA
	for lineNo := 1; s.Scan(); lineNo++ {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "GIMP Palette") || strings.HasPrefix(line, "Name:") || strings.HasPrefix(line, "Columns:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("read .gpl line %d: want at least three RGB fields", lineNo)
		}

		r, err := parseGPLComponent(fields[0])
		if err != nil {
			return nil, fmt.Errorf("read .gpl line %d: %w", lineNo, err)
		}
		g, err := parseGPLComponent(fields[1])
		if err != nil {
			return nil, fmt.Errorf("read .gpl line %d: %w", lineNo, err)
		}
		b, err := parseGPLComponent(fields[2])
		if err != nil {
			return nil, fmt.Errorf("read .gpl line %d: %w", lineNo, err)
		}
		pal = append(pal, color.NRGBA{R: r, G: g, B: b, A: 255})
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read .gpl palette: %w", err)
	}
	if len(pal) == 0 {
		return nil, fmt.Errorf("read .gpl palette: no colors found")
	}
	return pal, nil
}

func WriteGPL(w io.Writer, name string, pal []color.NRGBA) error {
	if name == "" {
		name = "sprite-gen"
	}
	if _, err := fmt.Fprintf(w, "GIMP Palette\nName: %s\nColumns: 8\n#\n", name); err != nil {
		return fmt.Errorf("write .gpl palette header: %w", err)
	}
	for _, c := range pal {
		if _, err := fmt.Fprintf(w, "%3d %3d %3d\t#%02x%02x%02x\n", c.R, c.G, c.B, c.R, c.G, c.B); err != nil {
			return fmt.Errorf("write .gpl palette: %w", err)
		}
	}
	return nil
}

func parseHexColor(token string) (color.NRGBA, bool, error) {
	if !strings.HasPrefix(token, "#") {
		return color.NRGBA{}, false, nil
	}
	if len(token) != 7 {
		return color.NRGBA{}, false, nil
	}
	v, err := strconv.ParseUint(token[1:], 16, 32)
	if err != nil {
		return color.NRGBA{}, false, fmt.Errorf("invalid hex color %q", token)
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}, true, nil
}

func parseGPLComponent(token string) (uint8, error) {
	v, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", token)
	}
	if v < 0 || v > 255 {
		return 0, fmt.Errorf("RGB component %d out of range", v)
	}
	return uint8(v), nil
}
