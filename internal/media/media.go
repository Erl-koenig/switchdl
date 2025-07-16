package media

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const (
	SwitchTubeBaseURL = "https://tube.switch.ch"
)

type Client struct {
	BaseURL     string
	AccessToken string
	Client      *http.Client
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

func NewClient(accessToken string) *Client {
	return &Client{
		BaseURL:     SwitchTubeBaseURL,
		AccessToken: accessToken,
		Client:      &http.Client{},
	}
}

func (c *Client) DownloadVideo(ctx context.Context, videoID, outputDir, filename string) error {
	videoDetails, err := c.fetchVideoDetails(ctx, videoID)
	if err != nil {
		return fmt.Errorf("failed to fetch video details: %w", err)
	}

	variants, err := c.fetchVideoVariants(ctx, videoID)
	if err != nil {
		return err
	}

	variant := selectBestVariant(variants)
	if variant == nil {
		return fmt.Errorf("no video/mp4 variant found for video ID: %s", videoID)
	}

	outputFilename := filename
	if outputFilename == "" {
		if videoDetails.Title != "" {
			outputFilename = sanitizeFilename(videoDetails.Title) + ".mp4"
		} else {
			outputFilename = "video.mp4"
		}
	} else {
		if !strings.HasSuffix(outputFilename, ".mp4") {
			outputFilename += ".mp4"
		}
	}
	fmt.Printf("Downloading video \"%s\"\n", outputFilename)

	outputFile := filepath.Join(outputDir, outputFilename)
	downloadURL := c.BaseURL + variant.Path
	return c.downloadVideoFile(ctx, downloadURL, outputFile)
}

func (c *Client) fetchVideoDetails(ctx context.Context, videoID string) (*VideoDetails, error) {
	url := fmt.Sprintf("%s/api/v1/browse/videos/%s", c.BaseURL, videoID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for video details: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get video details: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for video details: %d", resp.StatusCode)
	}

	var details VideoDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to parse video details response: %w", err)
	}

	return &details, nil
}

func (c *Client) fetchVideoVariants(ctx context.Context, videoID string) ([]VideoVariant, error) {
	url := fmt.Sprintf("%s/api/v1/browse/videos/%s/video_variants", c.BaseURL, videoID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get video variants: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var variants []VideoVariant
	if err := json.NewDecoder(resp.Body).Decode(&variants); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return variants, nil
}

func selectBestVariant(variants []VideoVariant) *VideoVariant {
	for _, variant := range variants {
		if variant.MediaType == "video/mp4" {
			return &variant
		}
	}
	return nil
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

func (c *Client) downloadVideoFile(ctx context.Context, downloadURL, output string) (err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code for download: %d", resp.StatusCode)
	}

	out, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close output file: %w", cerr)
		}
	}()

	return copyWithProgress(ctx, resp, out)
}

func copyWithProgress(ctx context.Context, resp *http.Response, out *os.File) (err error) {
	const (
		barStyleLBound     = "["
		barStyleFiller     = "="
		barStyleTip        = ">"
		barStylePadding    = "-"
		barStyleRBound     = "]"
		decoratorSeparator = " | "
		downloadMessage    = "Downloading:"
		doneMessage        = "done"
		unknownSizeMessage = " (unknown size)"
	)

	contentLength := resp.Header.Get("Content-Length")
	var totalSize int64
	if contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			totalSize = size
		}
	}

	p := mpb.NewWithContext(ctx, mpb.WithWidth(64))
	barStyle := mpb.BarStyle().
		Lbound(barStyleLBound).
		Filler(barStyleFiller).
		Tip(barStyleTip).
		Padding(barStylePadding).
		Rbound(barStyleRBound)

	var bar *mpb.Bar
	if totalSize > 0 {
		bar = p.New(totalSize,
			barStyle,
			mpb.PrependDecorators(
				decor.Name(downloadMessage, decor.WC{C: decor.DindentRight | decor.DextraSpace}),
				decor.OnComplete(decor.CountersKibiByte("% .2f / % .2f"), doneMessage),
			),
			mpb.AppendDecorators(
				decor.Percentage(),
				decor.Name(decoratorSeparator),
				decor.OnComplete(decor.AverageETA(decor.ET_STYLE_GO), ""),
			),
		)
	} else {
		bar = p.New(0,
			barStyle,
			mpb.PrependDecorators(
				decor.Name(downloadMessage, decor.WC{C: decor.DindentRight | decor.DextraSpace}),
				decor.CountersKibiByte("% .2f"),
			),
			mpb.AppendDecorators(decor.Name(unknownSizeMessage)),
		)
	}

	reader := bar.ProxyReader(resp.Body)
	defer func() {
		if cerr := reader.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close reader: %w", cerr)
		}
	}()

	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("failed to write video to file: %w", err)
	}

	p.Wait()
	fmt.Println("Video downloaded successfully to", out.Name())
	return nil
}
