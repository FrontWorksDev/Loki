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

	// Create encoder with compression level
	encoder := &png.Encoder{
		CompressionLevel: opts.Level.ToPNGCompressionLevel(),
	}

	// Encode to buffer to get compressed size
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	// Write to output
	compressedSize := int64(buf.Len())
	if _, err := io.Copy(w, &buf); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &Result{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
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

	// Create encoder with compression level
	encoder := &png.Encoder{
		CompressionLevel: opts.Level.ToPNGCompressionLevel(),
	}

	// Encode to buffer
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	// Write to output
	compressedSize := int64(buf.Len())
	if _, err := io.Copy(w, &buf); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &Result{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Format:         FormatPNG,
	}, nil
}

// SupportedFormats returns the formats supported by this processor.
func (p *PNGProcessor) SupportedFormats() []ImageFormat {
	return []ImageFormat{FormatPNG}
}
