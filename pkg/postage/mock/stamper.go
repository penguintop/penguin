// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"github.com/penguintop/penguin/pkg/postage"
    "github.com/penguintop/penguin/pkg/penguin"
)

type mockStamper struct{}

// NewStamper returns anew new mock stamper.
func NewStamper() postage.Stamper {
	return &mockStamper{}
}

// Stamp implements the Stamper interface. It returns an empty postage stamp.
func (mockStamper) Stamp(_ penguin.Address) (*postage.Stamp, error) {
	return &postage.Stamp{}, nil
}
