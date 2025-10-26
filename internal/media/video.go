package media

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

type VideoDetails struct {
	ID                     string `json:"id"`
	Title                  string `json:"title"`
	PublishedAt            string `json:"published_at"`
	DurationInMilliseconds int    `json:"duration_in_milliseconds"`
}

type VideoVariant struct {
	Path      string `json:"path"`
	Name      string `json:"name"`       // Label to distinguish variants, not display title
	MediaType string `json:"media_type"` // Expected to be video/mp4 for video downloads
	ExpiresAt string `json:"expires_at"` // NOTE: not used
}

// DownloadVideos is the main entry point for downloading one or more videos
func (c *Client) DownloadVideos(ctx context.Context, cfg *DownloadConfig) *DownloadSummary {
	total := len(cfg.VideoIDs)

	// Phase 1: Preparation (interactive: all user prompts)
	fmt.Printf("Preparing download of %d video(s)...\n", total)

	prepared, err := c.prepareVideoDownloads(ctx, cfg)
	if err != nil {
		return &DownloadSummary{
			Total:   total,
			Failed:  total,
			Results: []DownloadResult{{VideoID: "preparation", Error: err}},
		}
	}

	if len(prepared) == 0 {
		fmt.Println("No videos to download.")
		return &DownloadSummary{Total: total}
	}

	// Phase 2: Execution (non-interactive)
	if len(prepared) > 1 {
		return c.ExecuteConcurrentDownloads(ctx, prepared)
	}

	return c.downloadSinglePreparedVideo(ctx, prepared[0])
}

func (c *Client) prepareVideoDownloads(
	ctx context.Context,
	cfg *DownloadConfig,
) ([]PreparedDownload, error) {
	total := len(cfg.VideoIDs)
	prepared := make([]PreparedDownload, 0, total)
	fetchResults := c.fetchAllVideoDetails(ctx, cfg.VideoIDs)
	variants := c.resolveAllVariants(ctx, fetchResults, cfg, total) // sequentially if selection needed

	// handle file naming and conflicts
	for i, result := range fetchResults {
		videoID := cfg.VideoIDs[i]

		if result.err != nil {
			fmt.Printf("Warning: Skipping video %s: %v\n", videoID, result.err)
			continue
		}

		// Skip if variant resolution failed
		variant := variants[videoID]
		if variant == nil {
			fmt.Printf("Warning: Skipping video %s: no variant available\n", videoID)
			continue
		}

		var outputFilename string
		switch {
		case cfg.Filename != "" && total == 1:
			outputFilename = ensureMp4Suffix(cfg.Filename)
		case result.details.Title != "":
			outputFilename = ensureMp4Suffix(sanitizeFilename(result.details.Title))
		default:
			outputFilename = fmt.Sprintf("video_%s.mp4", videoID)
		}

		outputFile := filepath.Join(cfg.OutputDir, outputFilename)

		resolvedFile, err := handleExistingOutputFile(outputFile, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to handle existing file for %s: %w", videoID, err)
		}
		if resolvedFile == "" {
			fmt.Printf("Skipping video %s\n", videoID)
			continue
		}

		prepared = append(prepared, PreparedDownload{
			VideoID:      videoID,
			VideoDetails: result.details,
			Variant:      variant,
			OutputFile:   resolvedFile,
			// Index and Total will be set after all downloads are prepared
		})
	}

	actualTotal := len(prepared)
	for i := range prepared {
		prepared[i].Index = i + 1
		prepared[i].Total = actualTotal
	}

	return prepared, nil
}

// fetchAllVideoDetails fetches video details for all videos concurrently
func (c *Client) fetchAllVideoDetails(ctx context.Context, videoIDs []string) []fetchResult {
	total := len(videoIDs)
	jobs := make(chan fetchJob, total)
	resultsChan := make(chan fetchResult, total)

	var wg sync.WaitGroup
	for range NumWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result := fetchResult{index: job.index, videoID: job.videoID}

				details, err := c.fetchVideoDetails(ctx, job.videoID)
				if err != nil {
					result.err = fmt.Errorf("failed to fetch video details: %w", err)
				} else {
					result.details = details
				}

				resultsChan <- result
			}
		}()
	}

	for i, videoID := range videoIDs {
		jobs <- fetchJob{index: i, videoID: videoID}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results in order
	fetchResults := make([]fetchResult, total)
	for result := range resultsChan {
		fetchResults[result.index] = result
	}

	return fetchResults
}

func (c *Client) downloadSinglePreparedVideo(
	ctx context.Context,
	prepared PreparedDownload,
) *DownloadSummary {
	summary := &DownloadSummary{
		Total: 1,
	}

	fmt.Printf("Downloading video \"%s\"\n", filepath.Base(prepared.OutputFile))

	err := c.downloadPreparedVideo(ctx, prepared, nil, nil)
	if err != nil {
		fmt.Printf("Failed to download video %s: %v\n", prepared.VideoID, err)
		summary.Failed = 1
		summary.Results = append(summary.Results, DownloadResult{VideoID: prepared.VideoID, Error: err})
	} else {
		summary.Succeeded = 1
	}

	return summary
}

// resolveAllVariants handles variant selection for all videos
func (c *Client) resolveAllVariants(
	ctx context.Context,
	fetchResults []fetchResult,
	cfg *DownloadConfig,
	total int,
) map[string]*VideoVariant {
	// Check if we need interactive variant selection
	needsInteractiveSelection := cfg.SelectVariant && isInteractive() && total > 1
	if !needsInteractiveSelection {
		return c.fetchBestVariantsForAll(ctx, fetchResults)
	}

	individualSelection, err := c.promptForQualitySelection(ctx, cfg)
	if err != nil {
		fmt.Printf("Warning: failed to select quality: %v. Using best quality.\n", err)
		return c.fetchBestVariantsForAll(ctx, fetchResults)
	}

	if !individualSelection {
		return c.fetchBestVariantsForAll(ctx, fetchResults)
	}

	// User choose individual selection, prompt sequentially
	return c.selectVariantsInteractivelyForAll(ctx, fetchResults, total)
}

func (c *Client) fetchBestVariantsForAll(
	ctx context.Context,
	fetchResults []fetchResult,
) map[string]*VideoVariant {
	variants := make(map[string]*VideoVariant)

	for _, result := range fetchResults {
		if result.err != nil {
			continue
		}

		variantList, err := c.fetchVideoVariants(ctx, result.videoID)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch variants for %s: %v\n", result.videoID, err)
			continue
		}

		if len(variantList) == 0 {
			fmt.Printf("Warning: No variants available for %s\n", result.videoID)
			continue
		}

		variants[result.videoID] = selectBestVariant(variantList)
	}

	return variants
}

func (c *Client) selectVariantsInteractivelyForAll(
	ctx context.Context,
	fetchResults []fetchResult,
	total int,
) map[string]*VideoVariant {
	variants := make(map[string]*VideoVariant)

	for i, result := range fetchResults {
		if result.err != nil {
			continue
		}

		fmt.Printf("\nSelecting variant for video %d/%d (ID: %s)\n", i+1, total, result.videoID)

		variantList, err := c.fetchVideoVariants(ctx, result.videoID)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch variants for %s: %v. Skipping.\n", result.videoID, err)
			continue
		}

		if len(variantList) == 0 {
			fmt.Printf("Warning: No variants available for %s\n", result.videoID)
			continue
		}

		// Let user select interactively
		selectedVariant, err := selectVariantInteractively(variantList)
		if err != nil {
			fmt.Printf("Warning: Failed to select variant for %s: %v. Using best quality.\n", result.videoID, err)
			variants[result.videoID] = selectBestVariant(variantList)
			continue
		}

		variants[result.videoID] = selectedVariant
	}

	return variants
}
