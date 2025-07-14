package cmd

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

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"github.com/zalando/go-keyring"
)

const (
	switchTubeBaseURL = "https://tube.switch.ch"
)

type VideoVariant struct {
	Path string `json:"path"`
	// NOTE: not used yet
	ExpiresAt string `json:"expires_at"`
	Name      string `json:"name"`       // Label to distinguish variants, not display title
	MediaType string `json:"media_type"` // Expected to be video/mp4 for video downloads
}

type VideoDetails struct {
	Title string `json:"title"`
}

type downloadConfig struct {
	OutputDir   string
	Filename    string
	AccessToken string
	VideoID     string
}

var videoCfg downloadConfig

var videoCmd = &cobra.Command{
	Use:   "video <video_id>",
	Short: "Download a video specified by its id",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf(
				`no video ID provided. Usage: switchdl video <ID>
				 Use 'switchdl --help' for more information`,
			)
		}
		videoCfg.VideoID = args[0]

		if videoCfg.AccessToken == "" {
			token, err := keyring.Get(keyringService, keyringUser)
			if err == nil {
				videoCfg.AccessToken = token
			} else {
				return fmt.Errorf("access token not found. Run 'switchdl configure' or provide it with the --token flag")
			}
		}

		fmt.Println("Downloading video with ID:", videoCfg.VideoID)

		if err := os.MkdirAll(videoCfg.OutputDir, 0755); err != nil {
			return fmt.Errorf("error creating output directory: %w", err)
		}

		if err := downloadVideo(cmd.Context(), &videoCfg); err != nil {
			return fmt.Errorf("error downloading video: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.Flags().StringVarP(&videoCfg.OutputDir, "output-dir", "o", ".", "Output directory path")
	videoCmd.Flags().StringVarP(&videoCfg.Filename, "filename", "f", "", "Output filename (defaults to video title)")
	videoCmd.Flags().
		StringVarP(&videoCfg.AccessToken, "token", "t", "", "Access token for API authentication (overrides configured token)")
}

func downloadVideo(ctx context.Context, cfg *downloadConfig) error {
	client := &http.Client{}

	videoDetails, err := fetchVideoDetails(ctx, client, cfg)
	if err != nil {
		return fmt.Errorf("failed to fetch video details: %w", err)
	}

	variants, err := fetchVideoVariants(ctx, client, cfg)
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
			outputFilename = sanitizeFilename(videoDetails.Title) + ".mp4"
		} else {
			outputFilename = "video.mp4"
		}
	} else {
		if !strings.HasSuffix(outputFilename, ".mp4") {
			outputFilename += ".mp4"
		}
	}

	outputFile := filepath.Join(cfg.OutputDir, outputFilename)
	downloadURL := switchTubeBaseURL + variant.Path
	return downloadVideoFile(ctx, client, downloadURL, outputFile)
}

func fetchVideoDetails(ctx context.Context, client *http.Client, cfg *downloadConfig) (*VideoDetails, error) {
	url := fmt.Sprintf("%s/api/v1/browse/videos/%s", switchTubeBaseURL, cfg.VideoID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for video details: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", cfg.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
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

func fetchVideoVariants(ctx context.Context, client *http.Client, cfg *downloadConfig) ([]VideoVariant, error) {
	url := fmt.Sprintf("%s/api/v1/browse/videos/%s/video_variants", switchTubeBaseURL, cfg.VideoID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", cfg.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
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
	sanitized = strings.TrimSpace(sanitized) // Remove leading/trailing spaces
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}
	return sanitized
}

func downloadVideoFile(ctx context.Context, client *http.Client, downloadURL, output string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

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

func copyWithProgress(ctx context.Context, resp *http.Response, out *os.File) error {
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
	barStyle := mpb.BarStyle().Lbound(barStyleLBound).Filler(barStyleFiller).Tip(barStyleTip).Padding(barStylePadding).Rbound(barStyleRBound)

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

	_, err := io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("failed to write video to file: %w", err)
	}

	p.Wait()
	fmt.Println("Video downloaded successfully to", out.Name())
	return nil
}
