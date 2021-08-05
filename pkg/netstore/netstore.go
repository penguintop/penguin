// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package netstore provides an abstraction layer over the
// Penguin local storage layer that leverages connectivity
// with other peers in order to retrieve chunks from the network that cannot
// be found locally.
package netstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/recovery"
	"github.com/penguintop/penguin/pkg/retrieval"
	"github.com/penguintop/penguin/pkg/sctx"
	"github.com/penguintop/penguin/pkg/storage"
    "github.com/penguintop/penguin/pkg/penguin"
)

type store struct {
	storage.Storer
	retrieval        retrieval.Interface
	logger           logging.Logger
	validStamp       func(penguin.Chunk, []byte) (penguin.Chunk, error)
	recoveryCallback recovery.Callback // this is the callback to be executed when a chunk fails to be retrieved
}

var (
	ErrRecoveryAttempt = errors.New("failed to retrieve chunk, recovery initiated")
)

// New returns a new NetStore that wraps a given Storer.
func New(s storage.Storer, validStamp func(penguin.Chunk, []byte) (penguin.Chunk, error), rcb recovery.Callback, r retrieval.Interface, logger logging.Logger) storage.Storer {
	return &store{Storer: s, validStamp: validStamp, recoveryCallback: rcb, retrieval: r, logger: logger}
}

// Get retrieves a given chunk address.
// It will request a chunk from the network whenever it cannot be found locally.
// If the network path is taken, the method also stores the found chunk into the
// local-store.
func (s *store) Get(ctx context.Context, mode storage.ModeGet, addr penguin.Address) (ch penguin.Chunk, err error) {
	ch, err = s.Storer.Get(ctx, mode, addr)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// request from network
			ch, err = s.retrieval.RetrieveChunk(ctx, addr)
			if err != nil {
				targets := sctx.GetTargets(ctx)
				if targets == nil || s.recoveryCallback == nil {
					return nil, err
				}
				go s.recoveryCallback(addr, targets)
				return nil, ErrRecoveryAttempt
			}
			stamp, err := ch.Stamp().MarshalBinary()
			if err != nil {
				return nil, err
			}

			putMode := storage.ModePutRequest
			if mode == storage.ModeGetRequestPin {
				putMode = storage.ModePutRequestPin
			}

			cch, err := s.validStamp(ch, stamp)
			if err != nil {
				// if a chunk with an invalid postage stamp was received
				// we force it into the cache.
				putMode = storage.ModePutRequestCache
				cch = ch
			}

			_, err = s.Storer.Put(ctx, putMode, cch)
			if err != nil {
				return nil, fmt.Errorf("netstore retrieve put: %w", err)
			}
			return ch, nil
		}
		return nil, fmt.Errorf("netstore get: %w", err)
	}
	return ch, nil
}
