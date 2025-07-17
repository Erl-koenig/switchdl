// Package media provides functionality for interacting with the SwitchTube API
package media

import (
	"context"
	"fmt"
	"path/filepath"
)

const (
	SwitchTubeBaseURL = "https://tube.switch.ch"
)

type DownloadConfig struct {
	AccessToken   string
	OutputDir     string
	Overwrite     bool
	SelectVariant bool
	VideoIDs      []string
	Filename      string
}

type DownloadSummary struct {
	Total     int
	Succeeded int
	Failed    int
	Results   []DownloadResult
}

type DownloadResult struct {
	VideoID string
	Error   error
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

func (c *Client) DownloadVideo(ctx context.Context, cfg *DownloadConfig) error {
	// TODO: consider using two structs for single and multiple video downloads
	videoID := cfg.VideoIDs[0]

	videoDetails, err := c.fetchVideoDetails(ctx, videoID)
	if err != nil {
		return fmt.Errorf("failed to fetch video details: %w", err)
	}

	variants, err := c.fetchVideoVariants(ctx, videoID)
	if err != nil {
		return err
	}

	var variant *VideoVariant
	if cfg.SelectVariant && isInteractive() && len(variants) > 1 {
		variant, err = selectVariantInteractively(variants)
		if err != nil {
			return err
		}
	} else {
		variant = selectBestVariant(variants)
	}

	if variant == nil {
		return fmt.Errorf("no video/mp4 variant found for video ID: %s", videoID)
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

func (c *Client) DownloadVideos(ctx context.Context, cfg *DownloadConfig) *DownloadSummary {
	summary := &DownloadSummary{
		Total:   len(cfg.VideoIDs),
		Results: make([]DownloadResult, 0, len(cfg.VideoIDs)),
	}

	var variant *VideoVariant
	if cfg.SelectVariant && isInteractive() && len(cfg.VideoIDs) > 1 {
		var err error
		variant, err = c.promptForQualitySelection(ctx, cfg)
		if err != nil {
			fmt.Printf("Warning: failed to select quality: %v. Using best quality.\n", err)
			cfg.SelectVariant = false
		}
	}
	fmt.Printf("Starting download of %d video(s)\n", summary.Total)

	for i, videoID := range cfg.VideoIDs {
		fmt.Printf("\nProcessing video %d/%d (ID: %s)\n", i+1, summary.Total, videoID)

		// TODO: better way to handle variant selection and new configs, make cfg.SelectVariant immutable
		videoCfg := &DownloadConfig{
			AccessToken:   cfg.AccessToken,
			OutputDir:     cfg.OutputDir,
			Overwrite:     cfg.Overwrite,
			SelectVariant: cfg.SelectVariant && variant == nil,
			VideoIDs:      []string{videoID},
			Filename:      cfg.Filename,
		}
		err := c.DownloadVideo(ctx, videoCfg)
		result := DownloadResult{VideoID: videoID, Error: err}

		if err != nil {
			summary.Failed++
			fmt.Printf("Failed to download video %s: %v\n", videoID, err)
		} else {
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	if summary.Total > 1 {
		printDownloadSummary(summary)
	}

	return summary
}
