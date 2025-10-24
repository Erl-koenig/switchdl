package media

import (
	"strings"
)

const (
	maxFilenameLength = 255
)

func ensureMp4Suffix(name string) string {
	if !strings.HasSuffix(name, ".mp4") {
		return name + ".mp4"
	}
	return name
}

func sanitizeFilename(name string) string {
	invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	sanitized := name
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}
	sanitized = strings.TrimSpace(sanitized)
	if len(sanitized) > maxFilenameLength {
		sanitized = sanitized[:maxFilenameLength]
	}
	return sanitized
}

// NOTE: from the api docs "Video variants are ordered on their quality level with the highest quality variant first."
func selectBestVariant(variants []VideoVariant) *VideoVariant {
	for _, variant := range variants {
		if variant.MediaType == "video/mp4" {
			return &variant
		}
	}
	return nil
}
