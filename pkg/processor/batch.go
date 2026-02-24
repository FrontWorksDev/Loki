package processor

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Progress represents the progress of batch processing.
type Progress struct {
	// Total is the total number of items to process.
	Total int
	// Completed is the number of items that have been processed successfully.
	Completed int
	// Failed is the number of items that have failed.
	Failed int
	// Current is the path of the item currently being processed.
	Current string
}

// BatchProcessorOption is a functional option for DefaultBatchProcessor.
type BatchProcessorOption func(*DefaultBatchProcessor)

// DefaultBatchProcessor implements the BatchProcessor interface with parallel processing.
type DefaultBatchProcessor struct {
	jpegProc         Processor
	pngProc          Processor
	maxWorkers       int
	progressCallback func(Progress)
}

// NewDefaultBatchProcessor creates a new DefaultBatchProcessor with the given options.
func NewDefaultBatchProcessor(opts ...BatchProcessorOption) *DefaultBatchProcessor {
	bp := &DefaultBatchProcessor{
		jpegProc:   NewJPEGProcessor(),
		pngProc:    NewPNGProcessor(),
		maxWorkers: runtime.NumCPU(),
	}
	for _, opt := range opts {
		opt(bp)
	}
	if bp.maxWorkers < 1 {
		bp.maxWorkers = 1
	}
	return bp
}

// WithMaxWorkers sets the maximum number of concurrent workers.
func WithMaxWorkers(n int) BatchProcessorOption {
	return func(bp *DefaultBatchProcessor) {
		bp.maxWorkers = n
	}
}

// WithProgressCallback sets a callback function that is called on progress updates.
func WithProgressCallback(cb func(Progress)) BatchProcessorOption {
	return func(bp *DefaultBatchProcessor) {
		bp.progressCallback = cb
	}
}

// ProcessBatch processes multiple images in batch with parallel workers.
func (bp *DefaultBatchProcessor) ProcessBatch(ctx context.Context, items []BatchItem) ([]BatchResult, error) {
	if len(items) == 0 {
		return []BatchResult{}, nil
	}

	results := make([]BatchResult, len(items))
	itemCh := make(chan int, len(items))

	var mu sync.Mutex
	progress := Progress{Total: len(items)}

	// Send item indices to the channel.
	for i := range items {
		itemCh <- i
	}
	close(itemCh)

	// Determine worker count.
	workers := min(bp.maxWorkers, len(items))

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for idx := range itemCh {
				var result BatchResult

				select {
				case <-ctx.Done():
					// Context cancelled: mark item as failed but continue draining.
					result = BatchResult{
						Item:  items[idx],
						Error: ctx.Err(),
					}
				default:
					result = bp.processItem(ctx, items[idx])
				}

				results[idx] = result

				mu.Lock()
				if result.Error != nil {
					progress.Failed++
				} else {
					progress.Completed++
				}
				p := Progress{
					Total:     progress.Total,
					Completed: progress.Completed,
					Failed:    progress.Failed,
					Current:   items[idx].InputPath,
				}
				cb := bp.progressCallback
				mu.Unlock()

				if cb != nil {
					cb(p)
				}
			}
		})
	}

	wg.Wait()

	return results, nil
}

// processItem processes a single batch item.
func (bp *DefaultBatchProcessor) processItem(ctx context.Context, item BatchItem) BatchResult {
	format, err := detectFormatFromPath(item.InputPath)
	if err != nil {
		return BatchResult{Item: item, Error: err}
	}

	inFile, err := os.Open(item.InputPath)
	if err != nil {
		return BatchResult{Item: item, Error: fmt.Errorf("failed to open input file: %w", err)}
	}
	defer func() { _ = inFile.Close() }()

	// Ensure output directory exists.
	outDir := filepath.Dir(item.OutputPath)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return BatchResult{Item: item, Error: fmt.Errorf("failed to create output directory: %w", err)}
	}

	outFile, err := os.Create(item.OutputPath)
	if err != nil {
		return BatchResult{Item: item, Error: fmt.Errorf("failed to create output file: %w", err)}
	}
	defer func() { _ = outFile.Close() }()

	var proc Processor
	switch format {
	case FormatJPEG:
		proc = bp.jpegProc
	case FormatPNG:
		proc = bp.pngProc
	default:
		return BatchResult{Item: item, Error: fmt.Errorf("unsupported format: %s", format)}
	}

	result, err := proc.Compress(ctx, inFile, outFile, item.Options)
	if err != nil {
		_ = outFile.Close()
		_ = os.Remove(item.OutputPath)
		return BatchResult{Item: item, Error: err}
	}

	return BatchResult{Item: item, Result: result}
}

// detectFormatFromPath detects the image format from the file extension.
func detectFormatFromPath(path string) (ImageFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return FormatJPEG, nil
	case ".png":
		return FormatPNG, nil
	default:
		return -1, fmt.Errorf("unsupported image format: %s", ext)
	}
}

// ScanDirectoryOption is a functional option for ScanDirectory.
type ScanDirectoryOption func(*scanConfig)

type scanConfig struct {
	opts CompressOptions
}

// WithCompressOptions sets the compression options for scanned items.
func WithCompressOptions(opts CompressOptions) ScanDirectoryOption {
	return func(cfg *scanConfig) {
		cfg.opts = opts
	}
}

// ScanDirectory scans a directory for supported image files and returns BatchItems.
func ScanDirectory(inputDir, outputDir string, opts ...ScanDirectoryOption) ([]BatchItem, error) {
	cfg := &scanConfig{
		opts: DefaultCompressOptions(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	info, err := os.Stat(inputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to access input directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("input path is not a directory: %s", inputDir)
	}

	var items []BatchItem
	err = filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		_, fmtErr := detectFormatFromPath(path)
		if fmtErr != nil {
			return nil // Skip unsupported files.
		}

		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		outPath := filepath.Join(outputDir, relPath)

		items = append(items, BatchItem{
			InputPath:  path,
			OutputPath: outPath,
			Options:    cfg.opts,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	return items, nil
}
