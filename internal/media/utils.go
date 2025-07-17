package media

import (
	"fmt"
	"strings"
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
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
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

func printDownloadSummary(summary *DownloadSummary) {
	fmt.Printf("\nDownload Summary:\n")
	fmt.Printf("Total videos: %d\n", summary.Total)
	fmt.Printf("Successfully downloaded: %d\n", summary.Succeeded)
	fmt.Printf("Failed: %d\n", summary.Failed)

	if summary.Failed > 0 {
		fmt.Println("\nFailed downloads:")
		for _, result := range summary.Results {
			if result.Error != nil {
				fmt.Printf("- Video %s: %v\n", result.VideoID, result.Error)
			}
		}
	}
}
