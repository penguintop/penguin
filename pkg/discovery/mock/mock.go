// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"context"
	"sync"

    "github.com/penguintop/penguin/pkg/penguin"
)

type Discovery struct {
	mtx     sync.Mutex
	ctr     int //how many ops
	records map[string][]penguin.Address
}

func NewDiscovery() *Discovery {
	return &Discovery{
		records: make(map[string][]penguin.Address),
	}
}

func (d *Discovery) BroadcastPeers(ctx context.Context, addressee penguin.Address, peers ...penguin.Address) error {
	for _, peer := range peers {
		d.mtx.Lock()
		d.records[addressee.String()] = append(d.records[addressee.String()], peer)
		d.mtx.Unlock()
	}
	d.mtx.Lock()
	d.ctr++
	d.mtx.Unlock()
	return nil
}

func (d *Discovery) Broadcasts() int {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return d.ctr
}

func (d *Discovery) AddresseeRecords(addressee penguin.Address) (peers []penguin.Address, exists bool) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	peers, exists = d.records[addressee.String()]
	return
}

func (d *Discovery) Reset() {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.ctr = 0
	d.records = make(map[string][]penguin.Address)
}
