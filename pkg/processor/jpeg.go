package processor

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"

	// Register PNG decoder for Convert function
	_ "image/png"
)

// JPEGProcessor implements the Processor interface for JPEG images.
type JPEGProcessor struct{}

// NewJPEGProcessor creates a new JPEGProcessor.
func NewJPEGProcessor() *JPEGProcessor {
	return &JPEGProcessor{}
}

// Compress compresses a JPEG image.
func (p *JPEGProcessor) Compress(ctx context.Context, r io.Reader, w io.Writer, opts CompressOptions) (*Result, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read all input data to calculate original size
	inputData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	originalSize := int64(len(inputData))

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(inputData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Determine quality
	quality := opts.Quality
	if quality <= 0 {
		quality = opts.Level.ToJPEGQuality()
	}
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}

	// Encode to buffer to get compressed size
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	// Write to output
	compressedSize := int64(buf.Len())
	if _, err := io.Copy(w, &buf); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &Result{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Format:         FormatJPEG,
	}, nil
}

// Convert converts an image to JPEG format.
func (p *JPEGProcessor) Convert(ctx context.Context, r io.Reader, w io.Writer, opts ConvertOptions) (*Result, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read all input data to calculate original size
	inputData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	originalSize := int64(len(inputData))

	// Decode the image (supports multiple formats via registered decoders)
	img, _, err := image.Decode(bytes.NewReader(inputData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Determine quality
	quality := opts.Quality
	if quality <= 0 {
		quality = opts.Level.ToJPEGQuality()
	}
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}

	// Encode to buffer
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	// Write to output
	compressedSize := int64(buf.Len())
	if _, err := io.Copy(w, &buf); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &Result{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Format:         FormatJPEG,
	}, nil
}

// SupportedFormats returns the formats supported by this processor.
func (p *JPEGProcessor) SupportedFormats() []ImageFormat {
	return []ImageFormat{FormatJPEG}
}
