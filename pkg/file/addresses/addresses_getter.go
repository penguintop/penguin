// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package addresses

import (
	"context"

	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/penguin"
)

type addressesGetterStore struct {
	getter storage.Getter
	fn     penguin.AddressIterFunc
}

// NewGetter creates a new proxy storage.Getter which calls provided function
// for each chunk address processed.
func NewGetter(getter storage.Getter, fn penguin.AddressIterFunc) storage.Getter {
	return &addressesGetterStore{getter, fn}
}

func (s *addressesGetterStore) Get(ctx context.Context, mode storage.ModeGet, addr penguin.Address) (penguin.Chunk, error) {
	ch, err := s.getter.Get(ctx, mode, addr)
	if err != nil {
		return nil, err
	}

	return ch, s.fn(ch.Address())
}
