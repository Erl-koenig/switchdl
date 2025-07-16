package media

import (
	"context"
	"fmt"
	"path/filepath"
)

const (
	SwitchTubeBaseURL = "https://tube.switch.ch"
)

type DownloadVideoConfig struct {
	OutputDir   string
	Filename    string
	VideoID     string
	Overwrite   bool
	AccessToken string
}

type VideoVariant struct {
	Path string `json:"path"`
	// NOTE: not used yet
	ExpiresAt string `json:"expires_at"`
	Name      string `json:"name"`       // Label to distinguish variants, not display title
	MediaType string `json:"media_type"` // Expected to be video/mp4 for video downloads
}

type VideoDetails struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (c *Client) DownloadVideo(ctx context.Context, cfg *DownloadVideoConfig) error {
	videoDetails, err := c.fetchVideoDetails(ctx, cfg.VideoID)
	if err != nil {
		return fmt.Errorf("failed to fetch video details: %w", err)
	}

	variants, err := c.fetchVideoVariants(ctx, cfg.VideoID)
	if err != nil {
		return err
	}

	variant := selectBestVariant(variants)
	if variant == nil {
		return fmt.Errorf("no video/mp4 variant found for video ID: %s", cfg.VideoID)
	}

	outputFilename := cfg.Filename
	if outputFilename == "" {
		if videoDetails.Title != "" {
			outputFilename = ensureMp4Suffix(sanitizeFilename(videoDetails.Title))
		} else {
			outputFilename = "video.mp4"
		}
	} else {
		outputFilename = ensureMp4Suffix(outputFilename)
	}
	fmt.Printf("Downloading video \"%s\"\n", outputFilename)

	outputFile := filepath.Join(cfg.OutputDir, outputFilename)

	outputFile, err = handleExistingOutputFile(outputFile, cfg)
	if err != nil {
		return err
	}
	if outputFile == "" { // If skip was chosen in interactive mode
		return nil
	}

	downloadURL := c.BaseURL + variant.Path
	return c.downloadVideoFile(ctx, downloadURL, outputFile)
}
