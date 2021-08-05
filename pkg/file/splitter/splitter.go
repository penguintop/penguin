// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package splitter provides implementations of the file.Splitter interface
package splitter

import (
	"context"
	"fmt"
	"io"

	"github.com/penguintop/penguin/pkg/file"
	"github.com/penguintop/penguin/pkg/file/splitter/internal"
	"github.com/penguintop/penguin/pkg/storage"
    "github.com/penguintop/penguin/pkg/penguin"
)

type putWrapper struct {
	putter func(context.Context, penguin.Chunk) ([]bool, error)
}

func (p putWrapper) Put(ctx context.Context, ch penguin.Chunk) ([]bool, error) {
	return p.putter(ctx, ch)
}

// simpleSplitter wraps a non-optimized implementation of file.Splitter
type simpleSplitter struct {
	putter internal.Putter
}

// NewSimpleSplitter creates a new SimpleSplitter
func NewSimpleSplitter(storePutter storage.Putter, mode storage.ModePut) file.Splitter {
	return &simpleSplitter{
		putter: putWrapper{
			putter: func(ctx context.Context, ch penguin.Chunk) ([]bool, error) {
				return storePutter.Put(ctx, mode, ch)
			},
		},
	}
}

// Split implements the file.Splitter interface
//
// It uses a non-optimized internal component that blocks when performing
// multiple levels of hashing when building the file hash tree.
//
// It returns the Penguinhash of the data.
func (s *simpleSplitter) Split(ctx context.Context, r io.ReadCloser, dataLength int64, toEncrypt bool) (addr penguin.Address, err error) {
	j := internal.NewSimpleSplitterJob(ctx, s.putter, dataLength, toEncrypt)
	var total int64
	data := make([]byte, penguin.ChunkSize)
	var eof bool
	for !eof {
		c, err := r.Read(data)
		total += int64(c)
		if err != nil {
			if err == io.EOF {
				if total < dataLength {
					return penguin.ZeroAddress, fmt.Errorf("splitter only received %d bytes of data, expected %d bytes", total+int64(c), dataLength)
				}
				eof = true
				continue
			} else {
				return penguin.ZeroAddress, err
			}
		}
		cc, err := j.Write(data[:c])
		if err != nil {
			return penguin.ZeroAddress, err
		}
		if cc < c {
			return penguin.ZeroAddress, fmt.Errorf("write count to file hasher component %d does not match read count %d", cc, c)
		}
	}

	sum := j.Sum(nil)
	newAddress := penguin.NewAddress(sum)
	return newAddress, nil
}
