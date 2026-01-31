// Package processor provides image processing interfaces and types.
package processor

import (
	"image/png"
)

// ImageFormat represents supported image formats.
type ImageFormat int

const (
	// FormatJPEG represents JPEG image format.
	FormatJPEG ImageFormat = iota
	// FormatPNG represents PNG image format.
	FormatPNG
)

// String returns the string representation of the ImageFormat.
func (f ImageFormat) String() string {
	switch f {
	case FormatJPEG:
		return "jpeg"
	case FormatPNG:
		return "png"
	default:
		return "unknown"
	}
}

// IsValid returns true if the ImageFormat is a valid value.
func (f ImageFormat) IsValid() bool {
	switch f {
	case FormatJPEG, FormatPNG:
		return true
	default:
		return false
	}
}

// Extension returns the file extension for the ImageFormat.
func (f ImageFormat) Extension() string {
	switch f {
	case FormatJPEG:
		return ".jpg"
	case FormatPNG:
		return ".png"
	default:
		return ""
	}
}

// MIMEType returns the MIME type for the ImageFormat.
func (f ImageFormat) MIMEType() string {
	switch f {
	case FormatJPEG:
		return "image/jpeg"
	case FormatPNG:
		return "image/png"
	default:
		return ""
	}
}

// CompressionLevel represents the compression level.
type CompressionLevel int

const (
	// CompressionLow provides fast compression with larger file size.
	CompressionLow CompressionLevel = iota
	// CompressionMedium provides balanced compression.
	CompressionMedium
	// CompressionHigh provides slower compression with smaller file size.
	CompressionHigh
)

// String returns the string representation of the CompressionLevel.
func (c CompressionLevel) String() string {
	switch c {
	case CompressionLow:
		return "low"
	case CompressionMedium:
		return "medium"
	case CompressionHigh:
		return "high"
	default:
		return "unknown"
	}
}

// IsValid returns true if the CompressionLevel is a valid value.
func (c CompressionLevel) IsValid() bool {
	switch c {
	case CompressionLow, CompressionMedium, CompressionHigh:
		return true
	default:
		return false
	}
}

// ToPNGCompressionLevel converts CompressionLevel to png.CompressionLevel.
func (c CompressionLevel) ToPNGCompressionLevel() png.CompressionLevel {
	switch c {
	case CompressionLow:
		return png.BestSpeed
	case CompressionMedium:
		return png.DefaultCompression
	case CompressionHigh:
		return png.BestCompression
	default:
		return png.DefaultCompression
	}
}

// ToJPEGQuality converts CompressionLevel to JPEG quality value (1-100).
// Low=60, Medium=75, High=90.
func (c CompressionLevel) ToJPEGQuality() int {
	switch c {
	case CompressionLow:
		return 60
	case CompressionMedium:
		return 75
	case CompressionHigh:
		return 90
	default:
		return 75
	}
}
