// Package media provides functionality for interacting with the SwitchTube API
package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const (
	SwitchTubeBaseURL           = "https://tube.switch.ch"
	DefaultDirectoryPermissions = 0o755
)

type DownloadConfig struct {
	AccessToken   string
	ChannelID     string
	VideoIDs      []string
	OutputDir     string `mapstructure:"output-dir"`
	Filename      string `mapstructure:"filename"`
	Overwrite     bool   `mapstructure:"overwrite"`
	Skip          bool   `mapstructure:"skip"`
	SelectVariant bool   `mapstructure:"select-variant"`
	All           bool   `mapstructure:"all"`
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

type ChannelDetails struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ChannelVideo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type VideoVariant struct {
	Path string `json:"path"`
	// NOTE: not used yet
	ExpiresAt string `json:"expires_at"`
	Name      string `json:"name"`       // Label to distinguish variants, not display title
	MediaType string `json:"media_type"` // Expected to be video/mp4 for video downloads
}

type VideoDetails struct {
	ID                     string `json:"id"`
	Title                  string `json:"title"`
	PublishedAt            string `json:"published_at"`             // Date and time at which the video was last published including time zone information formatted (returns string in this format: 2025-06-02T11:08:32.977+02:00)
	DurationInMilliseconds int    `json:"duration_in_milliseconds"` // Duration of the video expressed in milliseconds. The value can be slightly different from the duration in the actual media files
}

func (c *Client) DownloadVideo(
	ctx context.Context,
	cfg *DownloadConfig,
	variant *VideoVariant,
) error {
	videoID := cfg.VideoIDs[0]

	videoDetails, err := c.fetchVideoDetails(ctx, videoID)
	if err != nil {
		return fmt.Errorf("failed to fetch video details: %w", err)
	}

	if variant == nil {
		variants, err := c.fetchVideoVariants(ctx, videoID)
		if err != nil {
			return err
		}

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
	if outputFile == "" { // If skip was chosen in interactive mode (existing file)
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

	// store pre-selected variants (1. fetch all variants, 2. select variants, 3. download each)
	videoVariants := make(map[string]*VideoVariant)
	if cfg.SelectVariant && isInteractive() {
		individualSelection, err := c.promptForQualitySelection(ctx, cfg)
		if err != nil {
			fmt.Printf("Warning: failed to select quality: %v. Using best quality.\n", err)
			cfg.SelectVariant = false
		} else if individualSelection { // select variants individually
			for i, videoID := range cfg.VideoIDs {
				fmt.Printf("\nProcessing video %d/%d (ID: %s)\n", i+1, summary.Total, videoID)
				variants, err := c.fetchVideoVariants(ctx, videoID)
				if err != nil {
					fmt.Printf("Failed to fetch variants for video %s: %v\n", videoID, err)
					continue
				}
				variant, err := selectVariantInteractively(variants)
				if err != nil {
					fmt.Printf("Failed to select variant for video %s: %v\n", videoID, err)
					continue
				}
				videoVariants[videoID] = variant
			}
		}
	}

	fmt.Printf("Starting download of %d video(s)\n", summary.Total)

	for i, videoID := range cfg.VideoIDs {
		fmt.Printf("\nProcessing video %d/%d (ID: %s)\n", i+1, summary.Total, videoID)

		videoCfg := &DownloadConfig{
			AccessToken:   cfg.AccessToken,
			OutputDir:     cfg.OutputDir,
			Overwrite:     cfg.Overwrite,
			Skip:          cfg.Skip,
			SelectVariant: cfg.SelectVariant,
			VideoIDs:      []string{videoID},
			Filename:      cfg.Filename,
		}

		variant := videoVariants[videoID] // will be nil if flag not provided
		err := c.DownloadVideo(ctx, videoCfg, variant)
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

func (c *Client) DownloadChannel(ctx context.Context, cfg *DownloadConfig) error {
	channelDetails, err := c.fetchChannelDetails(ctx, cfg.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to fetch channel details: %w", err)
	}

	channelVideos, err := c.fetchChannelVideos(ctx, cfg.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to fetch channel videos: %w", err)
	}

	if len(channelVideos) == 0 {
		fmt.Println("No videos found in this channel.")
		return nil
	}

	fmt.Printf("Found %d videos in channel '%s'\n", len(channelVideos), channelDetails.Name)

	videos := make([]*VideoDetails, len(channelVideos))
	for i, v := range channelVideos {
		details, err := c.fetchVideoDetails(ctx, v.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch video details for %s: %w", v.ID, err)
		}
		videos[i] = details
	}

	var selectedVideos []*VideoDetails
	if cfg.All {
		selectedVideos = videos
	} else {
		selectedVideos, err = selectVideosInteractively(videos)
		if err != nil {
			return err
		}
	}

	if len(selectedVideos) == 0 {
		fmt.Println("No videos selected.")
		return nil
	}

	// create subdirectory for channel videos
	channelDir := filepath.Join(cfg.OutputDir, sanitizeFilename(channelDetails.Name))
	if err := os.MkdirAll(channelDir, DefaultDirectoryPermissions); err != nil {
		return fmt.Errorf("failed to create channel directory: %w", err)
	}
	fmt.Printf("Downloading %d video(s) to '%s'\n", len(selectedVideos), channelDir)

	videoIDs := make([]string, len(selectedVideos))
	for i, v := range selectedVideos {
		videoIDs[i] = v.ID
	}

	videoCfg := &DownloadConfig{
		AccessToken:   cfg.AccessToken,
		OutputDir:     channelDir,
		Overwrite:     cfg.Overwrite,
		Skip:          cfg.Skip,
		SelectVariant: cfg.SelectVariant,
		VideoIDs:      videoIDs,
	}

	c.DownloadVideos(ctx, videoCfg)

	return nil
}
