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

func (c *Client) ValidateToken(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/profiles/me", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send validation request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("access token is invalid or expired (HTTP %d)", resp.StatusCode)
	default:
		return fmt.Errorf("unexpected API response: HTTP %d", resp.StatusCode)
	}
}

func (c *Client) fetchVideoDetails(ctx context.Context, videoID string) (*VideoDetails, error) {
	url := fmt.Sprintf("%s/api/v1/browse/videos/%s", c.BaseURL, videoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

func (c *Client) fetchVideoVariants(
	ctx context.Context,
	videoID string,
) ([]VideoVariant, error) {
	url := fmt.Sprintf("%s/api/v1/browse/videos/%s/video_variants", c.BaseURL, videoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

func (c *Client) downloadVideoFile(
	ctx context.Context,
	downloadURL, outputFile string,
) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
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

func (c *Client) fetchChannelDetails(
	ctx context.Context,
	channelID string,
) (*ChannelDetails, error) {
	url := fmt.Sprintf("%s/api/v1/browse/channels/%s", c.BaseURL, channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for channel details: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel details: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for channel details: %d", resp.StatusCode)
	}

	var details ChannelDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("failed to parse channel details response: %w", err)
	}

	return &details, nil
}

func (c *Client) fetchChannelVideos(ctx context.Context, channelID string) ([]ChannelVideo, error) {
	url := fmt.Sprintf("%s/api/v1/browse/channels/%s/videos", c.BaseURL, channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for channel videos: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channel videos: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for channel videos: %d", resp.StatusCode)
	}

	var videos []ChannelVideo
	if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		return nil, fmt.Errorf("failed to parse channel videos response: %w", err)
	}

	return videos, nil
}
