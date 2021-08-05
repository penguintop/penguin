package cac

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/penguintop/penguin/pkg/bmtpool"
    "github.com/penguintop/penguin/pkg/penguin"
)

var (
	errTooShortChunkData = errors.New("short chunk data")
	errTooLargeChunkData = errors.New("data too large")
)

// New creates a new content address chunk by initializing a span and appending the data to it.
func New(data []byte) (penguin.Chunk, error) {
	dataLength := len(data)
	if dataLength > penguin.ChunkSize {
		return nil, errTooLargeChunkData
	}

	if dataLength == 0 {
		return nil, errTooShortChunkData
	}

	span := make([]byte, penguin.SpanSize)
	binary.LittleEndian.PutUint64(span, uint64(dataLength))
	return newWithSpan(data, span)
}

// NewWithDataSpan creates a new chunk assuming that the span precedes the actual data.
func NewWithDataSpan(data []byte) (penguin.Chunk, error) {
	dataLength := len(data)
	if dataLength > penguin.ChunkSize+penguin.SpanSize {
		return nil, errTooLargeChunkData
	}

	if dataLength < penguin.SpanSize {
		return nil, errTooShortChunkData
	}
	return newWithSpan(data[penguin.SpanSize:], data[:penguin.SpanSize])
}

// newWithSpan creates a new chunk prepending the given span to the data.
func newWithSpan(data, span []byte) (penguin.Chunk, error) {
	h := hasher(data)
	hash, err := h(span)
	if err != nil {
		return nil, err
	}

	cdata := make([]byte, len(data)+len(span))
	copy(cdata[:penguin.SpanSize], span)
	copy(cdata[penguin.SpanSize:], data)
	return penguin.NewChunk(penguin.NewAddress(hash), cdata), nil
}

// hasher is a helper function to hash a given data based on the given span.
func hasher(data []byte) func([]byte) ([]byte, error) {
	return func(span []byte) ([]byte, error) {
		hasher := bmtpool.Get()
		defer bmtpool.Put(hasher)

		hasher.SetHeader(span)
		if _, err := hasher.Write(data); err != nil {
			return nil, err
		}
		return hasher.Hash(nil)
	}
}

// Valid checks whether the given chunk is a valid content-addressed chunk.
func Valid(c penguin.Chunk) bool {
	data := c.Data()
	if len(data) < penguin.SpanSize {
		return false
	}

	if len(data) > penguin.ChunkSize+penguin.SpanSize {
		return false
	}

	h := hasher(data[penguin.SpanSize:])
	hash, _ := h(data[:penguin.SpanSize])
	return bytes.Equal(hash, c.Address().Bytes())
}
