package processor

import (
	"io"
)

// readAllWithLimit reads all data from r, enforcing an optional size limit.
// If maxSize <= 0, it behaves like io.ReadAll with no limit.
// If maxSize > 0, it returns ErrFileTooLarge if the data exceeds maxSize bytes.
func readAllWithLimit(r io.Reader, maxSize int64) ([]byte, error) {
	if maxSize <= 0 {
		return io.ReadAll(r)
	}

	// Read up to maxSize+1 bytes to detect overflow
	limited := io.LimitReader(r, maxSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > maxSize {
		return nil, ErrFileTooLarge
	}

	return data, nil
}
