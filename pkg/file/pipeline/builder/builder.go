// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builder

import (
	"context"
	"fmt"
	"io"

	"github.com/penguintop/penguin/pkg/encryption"
	"github.com/penguintop/penguin/pkg/file/pipeline"
	"github.com/penguintop/penguin/pkg/file/pipeline/bmt"
	enc "github.com/penguintop/penguin/pkg/file/pipeline/encryption"
	"github.com/penguintop/penguin/pkg/file/pipeline/feeder"
	"github.com/penguintop/penguin/pkg/file/pipeline/hashtrie"
	"github.com/penguintop/penguin/pkg/file/pipeline/store"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/penguin"
)

// NewPipelineBuilder returns the appropriate pipeline according to the specified parameters
func NewPipelineBuilder(ctx context.Context, s storage.Putter, mode storage.ModePut, encrypt bool) pipeline.Interface {
	if encrypt {
		return newEncryptionPipeline(ctx, s, mode)
	}
	return newPipeline(ctx, s, mode)
}

// newPipeline creates a standard pipeline that only hashes content with BMT to create
// a merkle-tree of hashes that represent the given arbitrary size byte stream. Partial
// writes are supported. The pipeline flow is: Data -> Feeder -> BMT -> Storage -> HashTrie.
func newPipeline(ctx context.Context, s storage.Putter, mode storage.ModePut) pipeline.Interface {
	tw := hashtrie.NewHashTrieWriter(penguin.ChunkSize, penguin.Branches, penguin.HashSize, newShortPipelineFunc(ctx, s, mode))
	lsw := store.NewStoreWriter(ctx, s, mode, tw)
	b := bmt.NewBmtWriter(lsw)
	return feeder.NewChunkFeederWriter(penguin.ChunkSize, b)
}

// newShortPipelineFunc returns a constructor function for an ephemeral hashing pipeline
// needed by the hashTrieWriter.
func newShortPipelineFunc(ctx context.Context, s storage.Putter, mode storage.ModePut) func() pipeline.ChainWriter {
	return func() pipeline.ChainWriter {
		lsw := store.NewStoreWriter(ctx, s, mode, nil)
		return bmt.NewBmtWriter(lsw)
	}
}

// newEncryptionPipeline creates an encryption pipeline that encrypts using CTR, hashes content with BMT to create
// a merkle-tree of hashes that represent the given arbitrary size byte stream. Partial
// writes are supported. The pipeline flow is: Data -> Feeder -> Encryption -> BMT -> Storage -> HashTrie.
// Note that the encryption writer will mutate the data to contain the encrypted span, but the span field
// with the unencrypted span is preserved.
func newEncryptionPipeline(ctx context.Context, s storage.Putter, mode storage.ModePut) pipeline.Interface {
	tw := hashtrie.NewHashTrieWriter(penguin.ChunkSize, 64, penguin.HashSize+encryption.KeyLength, newShortEncryptionPipelineFunc(ctx, s, mode))
	lsw := store.NewStoreWriter(ctx, s, mode, tw)
	b := bmt.NewBmtWriter(lsw)
	enc := enc.NewEncryptionWriter(encryption.NewChunkEncrypter(), b)
	return feeder.NewChunkFeederWriter(penguin.ChunkSize, enc)
}

// newShortEncryptionPipelineFunc returns a constructor function for an ephemeral hashing pipeline
// needed by the hashTrieWriter.
func newShortEncryptionPipelineFunc(ctx context.Context, s storage.Putter, mode storage.ModePut) func() pipeline.ChainWriter {
	return func() pipeline.ChainWriter {
		lsw := store.NewStoreWriter(ctx, s, mode, nil)
		b := bmt.NewBmtWriter(lsw)
		return enc.NewEncryptionWriter(encryption.NewChunkEncrypter(), b)
	}
}

// FeedPipeline feeds the pipeline with the given reader until EOF is reached.
// It returns the cryptographic root hash of the content.
func FeedPipeline(ctx context.Context, pipeline pipeline.Interface, r io.Reader) (addr penguin.Address, err error) {
	data := make([]byte, penguin.ChunkSize)
	for {
		c, err := r.Read(data)
		if err != nil {
			if err == io.EOF {
				if c > 0 {
					cc, err := pipeline.Write(data[:c])
					if err != nil {
						return penguin.ZeroAddress, err
					}
					if cc < c {
						return penguin.ZeroAddress, fmt.Errorf("pipeline short write: %d mismatches %d", cc, c)
					}
				}
				break
			} else {
				return penguin.ZeroAddress, err
			}
		}
		cc, err := pipeline.Write(data[:c])
		if err != nil {
			return penguin.ZeroAddress, err
		}
		if cc < c {
			return penguin.ZeroAddress, fmt.Errorf("pipeline short write: %d mismatches %d", cc, c)
		}
		select {
		case <-ctx.Done():
			return penguin.ZeroAddress, ctx.Err()
		default:
		}
	}
	select {
	case <-ctx.Done():
		return penguin.ZeroAddress, ctx.Err()
	default:
	}

	sum, err := pipeline.Sum()
	if err != nil {
		return penguin.ZeroAddress, err
	}

	newAddress := penguin.NewAddress(sum)
	return newAddress, nil
}
