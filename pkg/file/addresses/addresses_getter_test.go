// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package addresses_test

import (
	"context"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/penguintop/penguin/pkg/file"
	"github.com/penguintop/penguin/pkg/file/addresses"
	"github.com/penguintop/penguin/pkg/file/joiner"
	filetest "github.com/penguintop/penguin/pkg/file/testing"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/storage/mock"
    "github.com/penguintop/penguin/pkg/penguin"
)

func TestAddressesGetterIterateChunkAddresses(t *testing.T) {
	store := mock.NewStorer()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// create root chunk with 2 references and the referenced data chunks
	rootChunk := filetest.GenerateTestRandomFileChunk(penguin.ZeroAddress, penguin.ChunkSize*2, penguin.SectionSize*2)
	_, err := store.Put(ctx, storage.ModePutUpload, rootChunk)
	if err != nil {
		t.Fatal(err)
	}

	firstAddress := penguin.NewAddress(rootChunk.Data()[8 : penguin.SectionSize+8])
	firstChunk := filetest.GenerateTestRandomFileChunk(firstAddress, penguin.ChunkSize, penguin.ChunkSize)
	_, err = store.Put(ctx, storage.ModePutUpload, firstChunk)
	if err != nil {
		t.Fatal(err)
	}

	secondAddress := penguin.NewAddress(rootChunk.Data()[penguin.SectionSize+8:])
	secondChunk := filetest.GenerateTestRandomFileChunk(secondAddress, penguin.ChunkSize, penguin.ChunkSize)
	_, err = store.Put(ctx, storage.ModePutUpload, secondChunk)
	if err != nil {
		t.Fatal(err)
	}

	createdAddresses := []penguin.Address{rootChunk.Address(), firstAddress, secondAddress}

	foundAddresses := make(map[string]struct{})
	var foundAddressesMu sync.Mutex

	addressIterFunc := func(addr penguin.Address) error {
		foundAddressesMu.Lock()
		defer foundAddressesMu.Unlock()

		foundAddresses[addr.String()] = struct{}{}
		return nil
	}

	addressesGetter := addresses.NewGetter(store, addressIterFunc)

	j, _, err := joiner.New(ctx, addressesGetter, rootChunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	_, err = file.JoinReadAll(ctx, j, ioutil.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if len(createdAddresses) != len(foundAddresses) {
		t.Fatalf("expected to find %d addresses, got %d", len(createdAddresses), len(foundAddresses))
	}

	checkAddressFound := func(t *testing.T, foundAddresses map[string]struct{}, address penguin.Address) {
		t.Helper()

		if _, ok := foundAddresses[address.String()]; !ok {
			t.Fatalf("expected address %s not found", address.String())
		}
	}

	for _, createdAddress := range createdAddresses {
		checkAddressFound(t, foundAddresses, createdAddress)
	}
}
