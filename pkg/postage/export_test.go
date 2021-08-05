// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package postage

import (
    "github.com/penguintop/penguin/pkg/penguin"
)

func (st *StampIssuer) Inc(a penguin.Address) error {
	return st.inc(a)
}
