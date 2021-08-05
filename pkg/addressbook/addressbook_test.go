// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package addressbook_test

import (
	"testing"

	"github.com/penguintop/penguin/pkg/addressbook"
	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/pen"
	"github.com/penguintop/penguin/pkg/statestore/mock"
    "github.com/penguintop/penguin/pkg/penguin"

	ma "github.com/multiformats/go-multiaddr"
)

type bookFunc func(t *testing.T) (book addressbook.Interface)

func TestInMem(t *testing.T) {
	run(t, func(t *testing.T) addressbook.Interface {
		store := mock.NewStateStore()
		book := addressbook.New(store)
		return book
	})
}

func run(t *testing.T, f bookFunc) {
	store := f(t)
	addr1 := penguin.NewAddress([]byte{0, 1, 2, 3})
	addr2 := penguin.NewAddress([]byte{0, 1, 2, 4})
	multiaddr, err := ma.NewMultiaddr("/ip4/1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}

	pk, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		t.Fatal(err)
	}

	penAddr, err := pen.NewAddress(crypto.NewDefaultSigner(pk), multiaddr, addr1, 1)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Put(addr1, *penAddr)
	if err != nil {
		t.Fatal(err)
	}

	v, err := store.Get(addr1)
	if err != nil {
		t.Fatal(err)
	}

	if !penAddr.Equal(v) {
		t.Fatalf("expectted: %s, want %s", v, multiaddr)
	}

	notFound, err := store.Get(addr2)
	if err != addressbook.ErrNotFound {
		t.Fatal(err)
	}

	if notFound != nil {
		t.Fatalf("expected nil got %s", v)
	}

	overlays, err := store.Overlays()
	if err != nil {
		t.Fatal(err)
	}

	if len(overlays) != 1 {
		t.Fatalf("expected overlay len %v, got %v", 1, len(overlays))
	}

	addresses, err := store.Addresses()
	if err != nil {
		t.Fatal(err)
	}

	if len(addresses) != 1 {
		t.Fatalf("expected addresses len %v, got %v", 1, len(addresses))
	}
}
