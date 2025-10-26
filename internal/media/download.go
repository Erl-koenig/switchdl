package media

import (
	"context"
	"fmt"
	"sync"

	"github.com/vbauerster/mpb/v8"
)

const (
	NumWorkers = 3 // NOTE: set to 3, as the API documentation recommends
)

type DownloadWorkerPool struct {
	ctx      context.Context
	client   *Client
	jobs     chan PreparedDownload
	results  chan DownloadResult
	progress *mpb.Progress
	bars     map[string]*mpb.Bar // Map of videoID to pre-created progress bar
	wg       sync.WaitGroup
}

func NewDownloadWorkerPool(
	ctx context.Context,
	client *Client,
	progress *mpb.Progress,
	bars map[string]*mpb.Bar,
) *DownloadWorkerPool {
	return &DownloadWorkerPool{
		ctx:      ctx,
		client:   client,
		jobs:     make(chan PreparedDownload, NumWorkers),
		results:  make(chan DownloadResult, NumWorkers),
		progress: progress,
		bars:     bars,
	}
}

// start launches the worker goroutines
func (p *DownloadWorkerPool) start() {
	for range NumWorkers {
		p.wg.Add(1)
		go p.worker()
	}
}

// close signals that no more jobs will be submitted and waits for completion
func (p *DownloadWorkerPool) close() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// worker processes download jobs from the jobs channel
func (p *DownloadWorkerPool) worker() {
	defer p.wg.Done()

	for job := range p.jobs {
		select {
		case <-p.ctx.Done():
			p.results <- DownloadResult{
				VideoID: job.VideoID,
				Error:   p.ctx.Err(),
			}
			return
		default:
			result := p.downloadJob(job)
			p.results <- result
		}
	}
}

// downloadJob executes a single download job
func (p *DownloadWorkerPool) downloadJob(job PreparedDownload) DownloadResult {
	bar := p.bars[job.VideoID]
	err := p.client.downloadPreparedVideo(p.ctx, job, p.progress, bar)
	return DownloadResult{
		VideoID: job.VideoID,
		Error:   err,
	}
}

func (c *Client) ExecuteConcurrentDownloads(
	ctx context.Context,
	prepared []PreparedDownload,
) *DownloadSummary {
	total := prepared[0].Total
	fmt.Printf("\nStarting concurrent download of %d video(s)\n", len(prepared))

	progress := mpb.NewWithContext(ctx, mpb.WithWidth(progressBarWidth))

	// Pre-create all progress bars in order
	bars := createProgressBars(progress, prepared)

	pool := NewDownloadWorkerPool(ctx, c, progress, bars)
	pool.start()

	// Submit all jobs
	go func() {
		for _, job := range prepared {
			pool.jobs <- job
		}
		pool.close()
	}()

	// Collect results
	summary := &DownloadSummary{
		Total: total,
	}

	for result := range pool.results {
		if result.Error != nil {
			summary.Failed++
			summary.Results = append(summary.Results, result)
		} else {
			summary.Succeeded++
		}
	}

	// Wait for all progress bars to complete
	progress.Wait()

	if len(prepared) > 1 {
		printDownloadSummary(summary)
	}

	return summary
}
