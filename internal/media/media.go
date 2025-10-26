// Package media provides functionality for interacting with the SwitchTube API
package media

import "fmt"

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

type PreparedDownload struct {
	VideoID      string
	VideoDetails *VideoDetails
	Variant      *VideoVariant
	OutputFile   string
	Index        int // For progress tracking ("Video 2/10")
	Total        int
}

type fetchJob struct {
	index   int
	videoID string
}

type fetchResult struct {
	index   int
	videoID string
	details *VideoDetails
	err     error
}

func printDownloadSummary(summary *DownloadSummary) {
	fmt.Printf("\nDownload Summary:\n")
	fmt.Printf("Total videos: %d\n", summary.Total)
	fmt.Printf("Successfully downloaded: %d\n", summary.Succeeded)
	fmt.Printf("Failed: %d\n", summary.Failed)

	if summary.Failed > 0 {
		fmt.Println("\nFailed downloads:")
		for _, result := range summary.Results {
			fmt.Printf("- Video %s: %v\n", result.VideoID, result.Error)
		}
	}
}
