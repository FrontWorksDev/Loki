package processor

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
)

func TestReadAllWithLimit_MaxInt64(t *testing.T) {
	data := []byte("hello world")
	result, err := readAllWithLimit(bytes.NewReader(data), math.MaxInt64)
	if err != nil {
		t.Fatalf("readAllWithLimit() error = %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Errorf("readAllWithLimit() = %v, want %v", result, data)
	}
}

func TestReadAllWithLimit_NoLimit(t *testing.T) {
	data := []byte("hello world")
	result, err := readAllWithLimit(bytes.NewReader(data), 0)
	if err != nil {
		t.Fatalf("readAllWithLimit() error = %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Errorf("readAllWithLimit() = %v, want %v", result, data)
	}
}

func TestReadAllWithLimit_NegativeLimit(t *testing.T) {
	data := []byte("hello world")
	result, err := readAllWithLimit(bytes.NewReader(data), -1)
	if err != nil {
		t.Fatalf("readAllWithLimit() error = %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Errorf("readAllWithLimit() = %v, want %v", result, data)
	}
}

func TestReadAllWithLimit_WithinLimit(t *testing.T) {
	data := []byte("hello world")
	result, err := readAllWithLimit(bytes.NewReader(data), 100)
	if err != nil {
		t.Fatalf("readAllWithLimit() error = %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Errorf("readAllWithLimit() = %v, want %v", result, data)
	}
}

func TestReadAllWithLimit_ExactLimit(t *testing.T) {
	data := []byte("hello world")
	result, err := readAllWithLimit(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("readAllWithLimit() error = %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Errorf("readAllWithLimit() = %v, want %v", result, data)
	}
}

func TestReadAllWithLimit_ExceedsLimit(t *testing.T) {
	data := []byte("hello world")
	_, err := readAllWithLimit(bytes.NewReader(data), int64(len(data)-1))
	if err == nil {
		t.Fatal("readAllWithLimit() should return error when data exceeds limit")
	}
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("readAllWithLimit() error = %v, want %v", err, ErrFileTooLarge)
	}
}

func TestReadAllWithLimit_ReadError(t *testing.T) {
	_, err := readAllWithLimit(&errReader{err: io.ErrUnexpectedEOF}, 100)
	if err == nil {
		t.Fatal("readAllWithLimit() should return error on read failure")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("readAllWithLimit() error = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestReadAllWithLimit_ReadErrorNoLimit(t *testing.T) {
	_, err := readAllWithLimit(&errReader{err: io.ErrUnexpectedEOF}, 0)
	if err == nil {
		t.Fatal("readAllWithLimit() should return error on read failure")
	}
}

func TestCountingWriter_Write(t *testing.T) {
	var buf bytes.Buffer
	cw := &countingWriter{w: &buf}

	data := []byte("hello world")
	n, err := cw.Write(data)
	if err != nil {
		t.Fatalf("countingWriter.Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("countingWriter.Write() n = %d, want %d", n, len(data))
	}
	if cw.n != int64(len(data)) {
		t.Errorf("countingWriter.n = %d, want %d", cw.n, len(data))
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("countingWriter wrote %v, want %v", buf.Bytes(), data)
	}
}

func TestCountingWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	cw := &countingWriter{w: &buf}

	data1 := []byte("hello ")
	data2 := []byte("world")

	if _, err := cw.Write(data1); err != nil {
		t.Fatalf("Write(data1) error = %v", err)
	}
	if _, err := cw.Write(data2); err != nil {
		t.Fatalf("Write(data2) error = %v", err)
	}

	expectedTotal := int64(len(data1) + len(data2))
	if cw.n != expectedTotal {
		t.Errorf("countingWriter.n = %d, want %d", cw.n, expectedTotal)
	}
	if buf.String() != "hello world" {
		t.Errorf("countingWriter wrote %q, want %q", buf.String(), "hello world")
	}
}

func TestCountingWriter_WriteError(t *testing.T) {
	ew := &errWriter{err: io.ErrClosedPipe}
	cw := &countingWriter{w: ew}

	_, err := cw.Write([]byte("test"))
	if err == nil {
		t.Fatal("countingWriter.Write() should propagate write error")
	}
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("countingWriter.Write() error = %v, want %v", err, io.ErrClosedPipe)
	}
}
