package processor

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"

	// Register JPEG decoder for Convert function
	_ "image/jpeg"
	// Register PNG decoder for Convert function
	_ "image/png"

	"github.com/chai2010/webp"
)

// WEBPProcessor implements the Processor interface for WebP images.
type WEBPProcessor struct{}

// NewWEBPProcessor creates a new WEBPProcessor.
func NewWEBPProcessor() *WEBPProcessor {
	return &WEBPProcessor{}
}

// Compress compresses a WebP image.
func (p *WEBPProcessor) Compress(ctx context.Context, r io.Reader, w io.Writer, opts CompressOptions) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	// Read all input data to calculate original size
	inputData, err := readAllWithLimit(r, opts.MaxFileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	originalSize := int64(len(inputData))

	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(inputData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	// Determine quality
	quality := float32(opts.Quality)
	if opts.Quality == 0 {
		quality = opts.Level.ToWebPQuality()
	} else {
		if quality < 1 {
			quality = 1
		}
		if quality > 100 {
			quality = 100
		}
	}

	// Encode directly to output via countingWriter
	cw := &countingWriter{w: w}
	if err := webp.Encode(cw, img, &webp.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}

	return &Result{
		OriginalSize:   originalSize,
		CompressedSize: cw.n,
		Format:         FormatWEBP,
	}, nil
}

// Convert converts an image to WebP format.
func (p *WEBPProcessor) Convert(ctx context.Context, r io.Reader, w io.Writer, opts ConvertOptions) (*Result, error) {
	// Validate target format
	if opts.Format != FormatWEBP {
		return nil, fmt.Errorf("WEBPProcessor only supports conversion to WebP, got %s", opts.Format)
	}

	if err := opts.Validate(); err != nil {
		return nil, err
	}

	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	// Read all input data to calculate original size
	inputData, err := readAllWithLimit(r, opts.MaxFileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	originalSize := int64(len(inputData))

	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	// Decode the image (supports multiple formats via registered decoders)
	img, _, err := image.Decode(bytes.NewReader(inputData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	// Determine quality
	quality := float32(opts.Quality)
	if opts.Quality <= 0 {
		quality = opts.Level.ToWebPQuality()
	}
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}

	// Encode directly to output via countingWriter
	cw := &countingWriter{w: w}
	if err := webp.Encode(cw, img, &webp.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}

	return &Result{
		OriginalSize:   originalSize,
		CompressedSize: cw.n,
		Format:         FormatWEBP,
	}, nil
}

// SupportedFormats returns the formats supported by this processor.
func (p *WEBPProcessor) SupportedFormats() []ImageFormat {
	return []ImageFormat{FormatWEBP}
}
