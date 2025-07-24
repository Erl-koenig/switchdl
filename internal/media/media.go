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
	Path      string `json:"path"`
	Name      string `json:"name"`       // Label to distinguish variants, not display title
	MediaType string `json:"media_type"` // Expected to be video/mp4 for video downloads
	ExpiresAt string `json:"expires_at"` // NOTE: not used
}

type VideoDetails struct {
	ID                     string `json:"id"`
	Title                  string `json:"title"`
	PublishedAt            string `json:"published_at"`             // Date and time at which the video was last published including time zone information formatted (returns string in this format: 2025-06-02T11:08:32.977+02:00)
	DurationInMilliseconds int    `json:"duration_in_milliseconds"` // Duration of the video expressed in milliseconds. The value can be slightly different from the duration in the actual media files
}

func (c *Client) downloadSingleVideo(
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
		variant, err = c.resolveVideoVariant(ctx, videoID, cfg)
		if err != nil {
			return err
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
	return c.downloadFileFromURL(ctx, downloadURL, outputFile)
}

func (c *Client) resolveVideoVariant(
	ctx context.Context,
	videoID string,
	cfg *DownloadConfig,
) (*VideoVariant, error) {
	variants, err := c.fetchVideoVariants(ctx, videoID)
	if err != nil {
		return nil, err
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("no video/mp4 variant found for video ID: %s", videoID)
	}

	if cfg.SelectVariant && isInteractive() && len(variants) > 1 {
		return selectVariantInteractively(variants)
	}
	return selectBestVariant(variants), nil
}

func (c *Client) DownloadVideos(ctx context.Context, cfg *DownloadConfig) *DownloadSummary {
	summary := &DownloadSummary{
		Total:   len(cfg.VideoIDs),
		Results: make([]DownloadResult, 0, len(cfg.VideoIDs)),
	}

	fmt.Printf("Starting download of %d video(s)\n", summary.Total)

	videoVariants := c.prepareVariants(ctx, cfg, summary)

	for i, videoID := range cfg.VideoIDs {
		result := c.processVideoDownload(
			ctx,
			videoID,
			i,
			summary.Total,
			cfg,
			videoVariants[videoID],
		)
		if result.Error != nil {
			summary.Failed++
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
		details, fetchErr := c.fetchVideoDetails(ctx, v.ID)
		if fetchErr != nil {
			return fmt.Errorf("failed to fetch video details for %s: %w", v.ID, fetchErr)
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
	if mkdirErr := os.MkdirAll(channelDir, DefaultDirectoryPermissions); mkdirErr != nil {
		return fmt.Errorf("failed to create channel directory: %w", mkdirErr)
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

func (c *Client) prepareVariants(
	ctx context.Context,
	cfg *DownloadConfig,
	summary *DownloadSummary,
) map[string]*VideoVariant {
	videoVariants := make(map[string]*VideoVariant)

	if !cfg.SelectVariant || !isInteractive() {
		return videoVariants
	}

	individualSelection, selectionErr := c.promptForQualitySelection(ctx, cfg)
	if selectionErr != nil {
		fmt.Printf("Warning: failed to select quality: %v. Using best quality.\n", selectionErr)
		cfg.SelectVariant = false
		return videoVariants
	}

	if !individualSelection {
		return videoVariants
	}

	for i, videoID := range cfg.VideoIDs {
		fmt.Printf("\nProcessing video %d/%d (ID: %s)\n", i+1, summary.Total, videoID)

		variants, variantErr := c.fetchVideoVariants(ctx, videoID)
		if variantErr != nil {
			fmt.Printf("Failed to fetch variants for video %s: %v\n", videoID, variantErr)
			continue
		}

		variant, selectErr := selectVariantInteractively(variants)
		if selectErr != nil {
			fmt.Printf("Failed to select variant for video %s: %v\n", videoID, selectErr)
			continue
		}

		videoVariants[videoID] = variant
	}

	return videoVariants
}

func (c *Client) processVideoDownload(
	ctx context.Context,
	videoID string,
	index int,
	total int,
	cfg *DownloadConfig,
	variant *VideoVariant,
) DownloadResult {
	fmt.Printf("\nProcessing video %d/%d (ID: %s)\n", index+1, total, videoID)

	videoCfg := &DownloadConfig{
		AccessToken:   cfg.AccessToken,
		OutputDir:     cfg.OutputDir,
		Overwrite:     cfg.Overwrite,
		Skip:          cfg.Skip,
		SelectVariant: cfg.SelectVariant,
		VideoIDs:      []string{videoID},
		Filename:      cfg.Filename,
	}

	err := c.downloadSingleVideo(ctx, videoCfg, variant)
	if err != nil {
		fmt.Printf("Failed to download video %s: %v\n", videoID, err)
	}

	return DownloadResult{VideoID: videoID, Error: err}
}
