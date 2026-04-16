package background

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

func ParseHexColor(raw string) (color.NRGBA, error) {
	value := strings.TrimPrefix(strings.TrimSpace(raw), "#")
	if len(value) != 6 {
		return color.NRGBA{}, fmt.Errorf("invalid --color value %q; want #RRGGBB", raw)
	}
	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid --color value %q; want #RRGGBB", raw)
	}
	return color.NRGBA{R: uint8(parsed >> 16), G: uint8((parsed >> 8) & 0xff), B: uint8(parsed & 0xff), A: 0xff}, nil
}

func WithinTolerance(a, b color.NRGBA, tolerance uint8) bool {
	return channelDiff(a.R, b.R) <= tolerance &&
		channelDiff(a.G, b.G) <= tolerance &&
		channelDiff(a.B, b.B) <= tolerance &&
		channelDiff(a.A, b.A) <= tolerance
}

func withinToleranceRGB(a, b color.NRGBA, tolerance uint8) bool {
	return channelDiff(a.R, b.R) <= tolerance &&
		channelDiff(a.G, b.G) <= tolerance &&
		channelDiff(a.B, b.B) <= tolerance
}

func channelDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
