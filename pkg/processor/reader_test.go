package processor

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

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
