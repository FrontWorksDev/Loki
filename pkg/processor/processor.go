package processor

import (
	"context"
	"errors"
	"io"
)

// Errors returned by processor operations.
var (
	// ErrPreserveMetadataNotSupported is returned when PreserveMetadata is set to true.
	// Metadata preservation is not yet implemented.
	ErrPreserveMetadataNotSupported = errors.New("preserve metadata is not yet supported")

	// ErrFileTooLarge is returned when the input file exceeds the MaxFileSize limit.
	ErrFileTooLarge = errors.New("file size exceeds maximum allowed size")
)

// Processor defines the interface for image processing operations.
type Processor interface {
	// Compress compresses an image from the reader and writes to the writer.
	Compress(ctx context.Context, r io.Reader, w io.Writer, opts CompressOptions) (*Result, error)

	// Convert converts an image format from the reader and writes to the writer.
	Convert(ctx context.Context, r io.Reader, w io.Writer, opts ConvertOptions) (*Result, error)

	// SupportedFormats returns the list of supported image formats.
	SupportedFormats() []ImageFormat
}

// CompressOptions contains options for image compression.
type CompressOptions struct {
	// Quality specifies the JPEG quality (1-100). 0 means use default.
	// This setting is only applied to JPEG and is ignored for PNG,
	// which uses Level to control compression.
	Quality int

	// Level specifies the compression level.
	// For JPEG: used when Quality is 0 (Low=60, Medium=75, High=90).
	// For PNG: directly controls compression (BestSpeed, Default, BestCompression).
	Level CompressionLevel

	// PreserveMetadata indicates whether to preserve image metadata.
	// Not yet implemented. Setting this to true will cause Validate() to return an error.
	PreserveMetadata bool

	// MaxFileSize specifies the maximum allowed input file size in bytes.
	// 0 means no limit.
	MaxFileSize int64
}

// Validate validates the CompressOptions and returns an error if any option is unsupported.
func (o CompressOptions) Validate() error {
	if o.PreserveMetadata {
		return ErrPreserveMetadataNotSupported
	}
	if o.MaxFileSize < 0 {
		return errors.New("max file size must be non-negative")
	}
	return nil
}

// DefaultCompressOptions returns the default compression options.
func DefaultCompressOptions() CompressOptions {
	return CompressOptions{
		Quality:          0,
		Level:            CompressionMedium,
		PreserveMetadata: false,
	}
}

// ConvertOptions contains options for image format conversion.
type ConvertOptions struct {
	// Format specifies the target image format.
	Format ImageFormat

	// CompressOptions contains compression settings for the output.
	CompressOptions
}

// DefaultConvertOptions returns the default conversion options.
func DefaultConvertOptions(format ImageFormat) ConvertOptions {
	return ConvertOptions{
		Format:          format,
		CompressOptions: DefaultCompressOptions(),
	}
}

// Result contains the result of an image processing operation.
type Result struct {
	// OriginalSize is the size of the original image in bytes.
	OriginalSize int64

	// CompressedSize is the size of the processed image in bytes.
	CompressedSize int64

	// Format is the format of the output image.
	Format ImageFormat
}

// CompressionRatio returns the compression ratio as a percentage.
// Returns 0 if original size is 0.
func (r *Result) CompressionRatio() float64 {
	if r.OriginalSize == 0 {
		return 0
	}
	return float64(r.CompressedSize) / float64(r.OriginalSize) * 100
}

// SavedBytes returns the number of bytes saved by compression.
func (r *Result) SavedBytes() int64 {
	return r.OriginalSize - r.CompressedSize
}

// SavedPercentage returns the percentage of bytes saved.
// Returns 0 if original size is 0.
func (r *Result) SavedPercentage() float64 {
	if r.OriginalSize == 0 {
		return 0
	}
	return float64(r.SavedBytes()) / float64(r.OriginalSize) * 100
}

// BatchProcessor defines the interface for batch image processing operations.
// This interface is designed for FRO-61 batch processing implementation.
type BatchProcessor interface {
	// ProcessBatch processes multiple images in batch.
	ProcessBatch(ctx context.Context, items []BatchItem) ([]BatchResult, error)
}

// BatchItem represents a single item in batch processing.
type BatchItem struct {
	// InputPath is the path to the input image file.
	InputPath string

	// OutputPath is the path to the output image file.
	OutputPath string

	// Options contains the compression options for this item.
	Options CompressOptions
}

// BatchResult represents the result of processing a single batch item.
type BatchResult struct {
	// Item is the original batch item.
	Item BatchItem

	// Result contains the processing result, nil if error occurred.
	Result *Result

	// Error contains any error that occurred during processing.
	Error error
}

// IsSuccess returns true if the batch item was processed successfully.
func (br *BatchResult) IsSuccess() bool {
	return br.Error == nil && br.Result != nil
}

// BatchConvertItem represents a single item in batch format conversion.
type BatchConvertItem struct {
	// InputPath is the path to the input image file.
	InputPath string

	// OutputPath is the path to the output image file.
	OutputPath string

	// Options contains the conversion options for this item.
	Options ConvertOptions
}

// BatchConvertResult represents the result of processing a single batch convert item.
type BatchConvertResult struct {
	// Item is the original batch convert item.
	Item BatchConvertItem

	// Result contains the processing result, nil if error occurred.
	Result *Result

	// Error contains any error that occurred during processing.
	Error error
}

// IsSuccess returns true if the batch convert item was processed successfully.
func (br *BatchConvertResult) IsSuccess() bool {
	return br.Error == nil && br.Result != nil
}
