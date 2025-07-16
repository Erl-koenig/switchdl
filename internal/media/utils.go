package media

import "strings"

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
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}
	return sanitized
}

func selectBestVariant(variants []VideoVariant) *VideoVariant {
	for _, variant := range variants {
		if variant.MediaType == "video/mp4" {
			return &variant
		}
	}
	return nil
}
