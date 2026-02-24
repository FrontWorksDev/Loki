package processor

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"

	// Register JPEG decoder for Convert function
	_ "image/jpeg"
)

// PNGProcessor implements the Processor interface for PNG images.
type PNGProcessor struct{}

// NewPNGProcessor creates a new PNGProcessor.
func NewPNGProcessor() *PNGProcessor {
	return &PNGProcessor{}
}

// Compress compresses a PNG image.
func (p *PNGProcessor) Compress(ctx context.Context, r io.Reader, w io.Writer, opts CompressOptions) (*Result, error) {
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

	// Create encoder with compression level
	encoder := &png.Encoder{
		CompressionLevel: opts.Level.ToPNGCompressionLevel(),
	}

	// Encode directly to output via countingWriter
	cw := &countingWriter{w: w}
	if err := encoder.Encode(cw, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return &Result{
		OriginalSize:   originalSize,
		CompressedSize: cw.n,
		Format:         FormatPNG,
	}, nil
}

// Convert converts an image to PNG format.
func (p *PNGProcessor) Convert(ctx context.Context, r io.Reader, w io.Writer, opts ConvertOptions) (*Result, error) {
	// Validate target format
	if opts.Format != FormatPNG {
		return nil, fmt.Errorf("PNGProcessor only supports conversion to PNG, got %s", opts.Format)
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

	// Create encoder with compression level
	encoder := &png.Encoder{
		CompressionLevel: opts.Level.ToPNGCompressionLevel(),
	}

	// Encode directly to output via countingWriter
	cw := &countingWriter{w: w}
	if err := encoder.Encode(cw, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return &Result{
		OriginalSize:   originalSize,
		CompressedSize: cw.n,
		Format:         FormatPNG,
	}, nil
}

// SupportedFormats returns the formats supported by this processor.
func (p *PNGProcessor) SupportedFormats() []ImageFormat {
	return []ImageFormat{FormatPNG}
}
