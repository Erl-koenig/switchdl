package media

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

func promptUser(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func handleExistingOutputFile(outputFile string, cfg *DownloadConfig) (string, error) {
	if cfg.Overwrite {
		fmt.Printf("File %s already exists. Overwriting it.\n", outputFile)
		return outputFile, nil
	}

	_, statErr := os.Stat(outputFile)
	if statErr == nil { // File exists
		if cfg.Skip {
			fmt.Printf("File %s already exists. Skipping download.\n", outputFile)
			return "", nil
		}

		if !isInteractive() {
			return "", fmt.Errorf(
				"output file %s already exists. Use -w / --overwrite to replace it or -s / --skip to skip",
				outputFile,
			)
		}

		return promptForFileAction(outputFile, cfg)
	} else if !os.IsNotExist(statErr) {
		return "", fmt.Errorf("error checking output file %s: %w", outputFile, statErr)
	}

	return outputFile, nil // file doesn't exist
}

func promptForFileAction(outputFile string, cfg *DownloadConfig) (string, error) {
	for {
		choice, inputErr := promptUser(
			fmt.Sprintf(
				"Output file %s already exists.\n[O]verwrite / [R]ename / [S]kip? (o/r/s): ",
				outputFile,
			),
		)
		if inputErr != nil {
			return "", fmt.Errorf("failed to read user input: %w", inputErr)
		}

		switch strings.ToLower(choice) {
		case "o", "overwrite":
			return outputFile, nil
		case "r", "rename":
			newPath, renameErr := promptForNewFilename(cfg)
			if renameErr != nil {
				return "", renameErr
			}
			return newPath, nil
		case "s", "skip":
			fmt.Println("Skipping download.")
			return "", nil
		default:
			fmt.Println("Invalid choice. Please enter o, r, or s.")
		}
	}
}

func promptForNewFilename(cfg *DownloadConfig) (string, error) {
	for {
		newName, inputErr := promptUser("Enter new filename: ")
		if inputErr != nil {
			return "", fmt.Errorf("failed to read new filename: %w", inputErr)
		}

		newName = ensureMp4Suffix(newName)
		newPath := filepath.Join(cfg.OutputDir, newName)

		if _, statErr := os.Stat(newPath); os.IsNotExist(statErr) {
			return newPath, nil
		}
		fmt.Printf("File %s already exists. Please choose another name.\n", newName)
	}
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
		progressBarWidth   = 64
	)

	contentLength := resp.Header.Get("Content-Length")
	var totalSize int64
	if contentLength != "" {
		var parseErr error
		totalSize, parseErr = strconv.ParseInt(contentLength, 10, 64)
		if parseErr != nil {
			// TODO: handle parseErr
			totalSize = 0
		}
	}

	p := mpb.NewWithContext(ctx, mpb.WithWidth(progressBarWidth))
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
	fmt.Printf("Video \"%s\" downloaded successfully \n", out.Name())
	return nil
}

func selectVariantInteractively(variants []VideoVariant) (*VideoVariant, error) {
	fmt.Println("\nAvailable video variants:")
	for i, v := range variants {
		fmt.Printf("[%d] %s (%s)\n", i+1, v.Name, v.MediaType)
	}

	for {
		choice, err := promptUser(fmt.Sprintf("\nSelect variant (1-%d): ", len(variants)))
		if err != nil {
			return nil, fmt.Errorf("failed to read user input: %w", err)
		}

		idx, err := strconv.Atoi(choice)
		if err != nil || idx < 1 || idx > len(variants) {
			fmt.Printf("Invalid choice. Please enter a number between 1 and %d.\n", len(variants))
			continue
		}

		return &variants[idx-1], nil
	}
}

func (c *Client) promptForQualitySelection(_ context.Context, cfg *DownloadConfig) (bool, error) {
	fmt.Println("\nMultiple videos detected. How would you like to handle video quality selection?")

	for {
		choice, err := promptUser(
			"Select quality [I]ndividually for each video / Use [B]est quality for all (i/b): ",
		)
		if err != nil {
			fmt.Println("Failed to read selection. Defaulting to best quality.")
			cfg.SelectVariant = false
			return false, err
		}

		switch strings.ToLower(choice) {
		case "i", "individual", "individually":
			return true, nil

		case "b", "best":
			fmt.Println("Using best quality for all videos.")
			cfg.SelectVariant = false
			return false, nil

		default:
			fmt.Println("Invalid choice. Please enter 'i' or 'b'.")
		}
	}
}

func selectVideosInteractively(videos []*VideoDetails) ([]*VideoDetails, error) {
	if err := displayVideosInTable(videos); err != nil {
		return nil, fmt.Errorf("failed to display videos: %w", err)
	}
	return promptForVideoSelection(videos)
}

func displayVideosInTable(videos []*VideoDetails) error {
	fmt.Println("\nAvailable videos:")

	const (
		minWidth      = 0
		tabWidth      = 0
		padding       = 3
		padChar       = ' '
		flags         = 0
		indexWidth    = 6
		titleWidth    = 15
		durationWidth = 10
		dateWidth     = 12
	)
	writer := tabwriter.NewWriter(os.Stdout, minWidth, tabWidth, padding, padChar, flags)

	if _, err := fmt.Fprintln(writer, "Index \t Title \t Duration \t Date"); err != nil {
		return fmt.Errorf("failed to write table header: %w", err)
	}
	if _, err := fmt.Fprintln(writer, strings.Repeat("─", indexWidth)+"\t"+strings.Repeat("─", titleWidth)+"\t"+strings.Repeat("─", durationWidth)+"\t"+strings.Repeat("─", durationWidth)); err != nil {
		return fmt.Errorf("failed to write table separator: %w", err)
	}

	for i, v := range videos {
		formattedDuration, formattedDate := formatVideoDetails(v)
		indexStr := strconv.Itoa(i + 1)

		if _, err := fmt.Fprintf(writer, "%s \t %-s \t %s \t %s\n", indexStr, v.Title, formattedDuration, formattedDate); err != nil {
			return fmt.Errorf("failed to write video row %d: %w", i+1, err)
		}
	}
	return writer.Flush()
}

func formatVideoDetails(v *VideoDetails) (string, string) {
	d := time.Duration(v.DurationInMilliseconds) * time.Millisecond
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60 //nolint:mnd // obvious, wrap minutes in an hour
	seconds := int(d.Seconds()) % 60 //nolint:mnd // obvious, wrap seconds in a minute

	formattedDuration := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds) // HH:MM:SS

	parsedTime, err := time.Parse(time.RFC3339, v.PublishedAt)
	if err != nil {
		return formattedDuration, "N/A"
	}
	return formattedDuration, parsedTime.Format(time.DateOnly)
}

func promptForVideoSelection(videos []*VideoDetails) ([]*VideoDetails, error) {
	for {
		selection, err := promptUser("\nSelect videos (1,3-5,8,...) or 'a'/'all' for all: ")
		if err != nil {
			return nil, fmt.Errorf("failed to read user input: %w", err)
		}

		trimmedSelection := strings.TrimSpace(selection)
		if trimmedSelection == "" {
			fmt.Println("Input cannot be empty. Please enter 'a'/'all' or a valid selection.")
			continue
		}

		selectionLower := strings.ToLower(trimmedSelection)
		if selectionLower == "a" || selectionLower == "all" {
			return videos, nil
		}

		selectedIndices, err := parseVideoSelection(trimmedSelection, len(videos))
		if err != nil {
			fmt.Printf("Invalid selection: %v. Try again.\n", err)
			continue
		}

		selectedVideos := make([]*VideoDetails, 0, len(selectedIndices))
		for _, idx := range selectedIndices {
			selectedVideos = append(selectedVideos, videos[idx])
		}
		return selectedVideos, nil
	}
}

func parseVideoSelection(selection string, lenVideos int) ([]int, error) {
	var finalIndices []int
	seen := make(map[int]bool)
	parts := strings.Split(selection, ",")
	for _, part := range parts {
		indices, err := parseSelectionPart(strings.TrimSpace(part), lenVideos)
		if err != nil {
			return nil, err
		}

		for _, idx := range indices {
			if !seen[idx] {
				finalIndices = append(finalIndices, idx)
				seen[idx] = true
			}
		}
	}
	return finalIndices, nil
}

func parseSelectionPart(part string, lenVideos int) ([]int, error) {
	const expectedRangeParts = 2
	if strings.Contains(part, "-") {
		rangeParts := strings.Split(part, "-")
		if len(rangeParts) != expectedRangeParts {
			return nil, fmt.Errorf("invalid range format: %s", part)
		}
		start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))

		if err1 != nil || err2 != nil || start < 1 || end > lenVideos || start > end {
			return nil, fmt.Errorf("invalid range: %s", part)
		}

		var indices []int
		for i := start; i <= end; i++ {
			indices = append(indices, i-1)
		}
		return indices, nil
	}

	idx, err := strconv.Atoi(part)
	if err != nil || idx < 1 || idx > lenVideos {
		return nil, fmt.Errorf("invalid video number: %s", part)
	}
	return []int{idx - 1}, nil
}
