// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package postage

import "math/big"

// ChainState contains data the batch service reads from the chain.
type ChainState struct {
	Block        uint64   `json:"block"`        // The block number of the last postage event.
	TotalAmount  *big.Int `json:"totalAmount"`  // Cumulative amount paid per stamp.
	CurrentPrice *big.Int `json:"currentPrice"` // Pen/chunk/block normalised price.
}
