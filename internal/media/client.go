package media

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	BaseURL     string
	AccessToken string
	Client      *http.Client
}

func NewClient(accessToken string) *Client {
	return &Client{
		BaseURL:     SwitchTubeBaseURL,
		AccessToken: accessToken,
		Client:      &http.Client{},
	}
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

func (c *Client) downloadVideoFile(ctx context.Context, downloadURL, outputFile string) (err error) {
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

	out, err := os.Create(outputFile)
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
