// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testing

import (
	"encoding/binary"
	"math/rand"

    "github.com/penguintop/penguin/pkg/penguin"
)

// GenerateTestRandomFileChunk generates one single chunk with arbitrary content and address
func GenerateTestRandomFileChunk(address penguin.Address, spanLength, dataSize int) penguin.Chunk {
	data := make([]byte, dataSize+8)
	binary.LittleEndian.PutUint64(data, uint64(spanLength))
	_, _ = rand.Read(data[8:]) // # skipcq: GSC-G404
	key := make([]byte, penguin.SectionSize)
	if address.IsZero() {
		_, _ = rand.Read(key) // # skipcq: GSC-G404
	} else {
		copy(key, address.Bytes())
	}
	return penguin.NewChunk(penguin.NewAddress(key), data)
}
