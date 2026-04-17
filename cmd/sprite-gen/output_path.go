package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func defaultStageOutPath(inPath, stage, name string) string {
	return filepath.Join("out", outputSubject(inPath), stage, name)
}

func defaultPaletteExtractOutPath(inPath, format string, maxColors int) string {
	return defaultStageOutPath(inPath, "palette", fmt.Sprintf("extracted-%d.%s", maxColors, strings.ToLower(format)))
}

func defaultExportOut(inPath, formatName string) string {
	subject := outputSubject(inPath)
	switch formatName {
	case "gif":
		return filepath.Join("out", subject, "export")
	case "sheet":
		return filepath.Join("out", subject, "export")
	}
	return filepath.Join("out", subject, "export")
}

func outputSubject(inPath string) string {
	parts := pathParts(inPath)
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "out" {
			continue
		}
		if i+2 < len(parts) && isOutputStage(parts[i+2]) {
			return parts[i+1]
		}
		if i+2 < len(parts) && isOutputStage(parts[i+1]) {
			return parts[i+2]
		}
	}

	stem := strings.TrimSuffix(filepath.Base(filepath.Clean(inPath)), filepath.Ext(inPath))
	if stem == "" || stem == "." {
		return "artifact"
	}
	return stem
}

func pathParts(path string) []string {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	clean = strings.TrimPrefix(clean, volume)
	clean = strings.TrimPrefix(clean, string(filepath.Separator))
	if clean == "" || clean == "." {
		return nil
	}
	return strings.Split(clean, string(filepath.Separator))
}

func isOutputStage(name string) bool {
	switch name {
	case "align", "diff", "export", "generate", "palette", "segment", "slice", "snap":
		return true
	case "normalize":
		return true
	case "prep":
		return true
	case "resize":
		return true
	case "rows":
		return true
	default:
		return false
	}
}
