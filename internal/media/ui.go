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

func handleExistingOutputFile(outputFile string, cfg *DownloadVideoConfig) (string, error) {
	if cfg.Overwrite {
		return outputFile, nil
	}
	if _, err := os.Stat(outputFile); err == nil {
		if isInteractive() {
			for {
				choice, err := promptUser(fmt.Sprintf("Output file %s already exists.\n[O]verwrite / [R]ename / [S]kip? (o/r/s): ", outputFile))
				if err != nil {
					return "", fmt.Errorf("failed to read user input: %w", err)
				}
				switch strings.ToLower(choice) {
				case "o", "overwrite":
					return outputFile, nil
				case "r", "rename":
					for {
						newName, err := promptUser("Enter new filename: ")
						if err != nil {
							return "", fmt.Errorf("failed to read new filename: %w", err)
						}
						newName = ensureMp4Suffix(newName)
						newPath := filepath.Join(cfg.OutputDir, newName)
						if _, err := os.Stat(newPath); os.IsNotExist(err) {
							return newPath, nil
						} else {
							fmt.Printf("File %s already exists. Please choose another name.\n", newName)
						}
					}
				case "s", "skip":
					fmt.Println("Skipping download.")
					return "", nil
				default:
					fmt.Println("Invalid choice. Please enter o, r, or s.")
				}
			}
		} else {
			return "", fmt.Errorf("output file %s already exists. Use -w / --overwrite to replace it", outputFile)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("error checking output file %s: %w", outputFile, err)
	}
	return outputFile, nil
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
