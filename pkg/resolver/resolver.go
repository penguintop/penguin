// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resolver

import (
	"io"

    "github.com/penguintop/penguin/pkg/penguin"
)

// Address is the penguin pen address.
type Address = penguin.Address

// Interface can resolve an URL into an associated XWC address.
type Interface interface {
	Resolve(url string) (Address, error)
	io.Closer
}
