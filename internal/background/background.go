package background

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

type Method string

const (
	MethodAuto Method = "auto"
	MethodKey  Method = "key"
	MethodEdge Method = "edge"
)

type Options struct {
	Method       Method
	KeyColor     color.NRGBA
	HasKeyColor  bool
	Tolerance    uint8
	Connectivity int
}

type Result struct {
	Image         *image.NRGBA
	Method        Method
	RemovedPixels int
	ChangedPixels int
}

type cleanerFunc func(img *image.NRGBA, opts Options) (Result, error)

var cleaners = map[Method]cleanerFunc{
	MethodKey:  removeKey,
	MethodEdge: removeEdge,
}

func ParseMethod(raw string) (Method, error) {
	switch Method(strings.ToLower(raw)) {
	case MethodAuto:
		return MethodAuto, nil
	case MethodKey:
		return MethodKey, nil
	case MethodEdge:
		return MethodEdge, nil
	default:
		return "", fmt.Errorf("invalid --method value %q; want auto, key, or edge", raw)
	}
}

func Remove(img *image.NRGBA, opts Options) (Result, error) {
	if img == nil {
		return Result{}, fmt.Errorf("remove background: image is nil")
	}
	if opts.Connectivity == 0 {
		opts.Connectivity = 4
	}
	if opts.Connectivity != 4 && opts.Connectivity != 8 {
		return Result{}, fmt.Errorf("invalid connectivity %d; want 4 or 8", opts.Connectivity)
	}

	method := opts.Method
	if method == "" {
		method = MethodAuto
	}
	if method == MethodAuto {
		if opts.HasKeyColor {
			method = MethodKey
		} else {
			method = MethodEdge
		}
	}

	cleaner, ok := cleaners[method]
	if !ok {
		return Result{}, fmt.Errorf("unsupported background removal method %q", method)
	}
	result, err := cleaner(img, opts)
	if err != nil {
		return Result{}, err
	}
	result.Method = method
	return result, nil
}

func cloneNRGBA(src *image.NRGBA) *image.NRGBA {
	if src == nil {
		return nil
	}
	out := image.NewNRGBA(src.Bounds())
	copy(out.Pix, src.Pix)
	return out
}
