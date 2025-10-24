package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type ChannelDetails struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ChannelVideo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (c *Client) DownloadChannel(ctx context.Context, cfg *DownloadConfig) error {
	// Phase 1: Preparation (interactive)
	prepared, err := c.prepareChannelDownloads(ctx, cfg)
	if err != nil {
		return err
	}

	if len(prepared) == 0 {
		return nil
	}

	// Phase 2: Execution (non-interactive)
	fmt.Printf("\nDownloading %d video(s) to '%s'\n", len(prepared), filepath.Dir(prepared[0].OutputFile))

	var summary *DownloadSummary
	if len(prepared) > 1 {
		summary = c.ExecuteConcurrentDownloads(ctx, prepared)
	} else {
		summary = c.downloadSinglePreparedVideo(ctx, prepared[0])
	}

	if summary.Succeeded == 0 {
		return errors.New("failed to download any videos")
	}

	return nil
}

func (c *Client) DownloadMultipleChannels(
	ctx context.Context,
	channelIDs []string,
	cfg *DownloadConfig,
) error {
	var allPrepared []PreparedDownload

	// Phase 1: Prepare ALL channels (interactive: collect all user input upfront)
	for i, channelID := range channelIDs {
		fmt.Printf("\n=== Preparing Channel %d/%d (ID: %s) ===\n", i+1, len(channelIDs), channelID)
		cfg.ChannelID = channelID

		prepared, err := c.prepareChannelDownloads(ctx, cfg)
		if err != nil {
			fmt.Printf("Warning: Failed to prepare channel %s: %v\n", channelID, err)
			continue
		}

		allPrepared = append(allPrepared, prepared...)
	}

	if len(allPrepared) == 0 {
		fmt.Println("No videos to download from any channel.")
		return nil
	}

	// Renumber all downloads with correct total
	totalVideos := len(allPrepared)
	for i := range allPrepared {
		allPrepared[i].Index = i + 1
		allPrepared[i].Total = totalVideos
	}

	// Phase 2: Execute all downloads concurrently (non-interactive)
	fmt.Printf("\n=== Starting download of %d video(s) from %d channel(s) ===\n",
		totalVideos, len(channelIDs))

	var summary *DownloadSummary
	if len(allPrepared) > 1 {
		summary = c.ExecuteConcurrentDownloads(ctx, allPrepared)
	} else {
		summary = c.downloadSinglePreparedVideo(ctx, allPrepared[0])
	}

	if summary.Succeeded == 0 {
		return errors.New("failed to download any videos")
	}

	return nil
}

func (c *Client) prepareChannelDownloads(
	ctx context.Context,
	cfg *DownloadConfig,
) ([]PreparedDownload, error) {
	channelDetails, err := c.fetchChannelDetails(ctx, cfg.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channel details: %w", err)
	}

	channelVideos, err := c.fetchChannelVideos(ctx, cfg.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channel videos: %w", err)
	}

	if len(channelVideos) == 0 {
		fmt.Println("No videos found in this channel.")
		return nil, nil
	}

	fmt.Printf("Found %d videos in channel '%s'\n", len(channelVideos), channelDetails.Name)

	// Fetch details for all videos
	videos := make([]*VideoDetails, len(channelVideos))
	for i, v := range channelVideos {
		details, fetchErr := c.fetchVideoDetails(ctx, v.ID)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch video details for %s: %w", v.ID, fetchErr)
		}
		videos[i] = details
	}

	// Select videos (interactive if not --all)
	var selectedVideos []*VideoDetails
	if cfg.All {
		selectedVideos = videos
	} else {
		selectedVideos, err = selectVideosInteractively(videos)
		if err != nil {
			return nil, err
		}
	}

	if len(selectedVideos) == 0 {
		fmt.Println("No videos selected.")
		return nil, nil
	}

	// Create subdirectory for channel videos
	channelDir := filepath.Join(cfg.OutputDir, sanitizeFilename(channelDetails.Name))
	if mkdirErr := os.MkdirAll(channelDir, DefaultDirectoryPermissions); mkdirErr != nil {
		return nil, fmt.Errorf("failed to create channel directory: %w", mkdirErr)
	}

	// Prepare downloads using the channel subdirectory
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

	return c.prepareVideoDownloads(ctx, videoCfg)
}
