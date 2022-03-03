// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package discovery exposes the discovery driver interface
// which is implemented by discovery protocols.
package discovery

import (
	"context"

	"github.com/penguintop/penguin/pkg/penguin"
)

type Driver interface {
	BroadcastPeers(ctx context.Context, addressee penguin.Address, peers ...penguin.Address) error
}
