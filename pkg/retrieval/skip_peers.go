// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package retrieval

import (
	"sync"

    "github.com/penguintop/penguin/pkg/penguin"
)

type skipPeers struct {
	overdraftAddresses []penguin.Address
	addresses          []penguin.Address
	mu                 sync.Mutex
}

func newSkipPeers() *skipPeers {
	return &skipPeers{}
}

func (s *skipPeers) All() []penguin.Address {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append(append(s.addresses[:0:0], s.addresses...), s.overdraftAddresses...)
}

func (s *skipPeers) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overdraftAddresses = []penguin.Address{}
}

func (s *skipPeers) Add(address penguin.Address) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range s.addresses {
		if a.Equal(address) {
			return
		}
	}

	s.addresses = append(s.addresses, address)
}

func (s *skipPeers) AddOverdraft(address penguin.Address) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range s.overdraftAddresses {
		if a.Equal(address) {
			return
		}
	}

	s.overdraftAddresses = append(s.overdraftAddresses, address)
}
