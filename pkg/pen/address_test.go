// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pen_test

import (
	"testing"

	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/pen"

	ma "github.com/multiformats/go-multiaddr"
)

func TestPenAddress(t *testing.T) {
	node1ma, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/1634/p2p/16Uiu2HAkx8ULY8cTXhdVAcMmLcH9AsTKz6uBQ7DPLKRjMLgBVYkA")
	if err != nil {
		t.Fatal(err)
	}

	privateKey1, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		t.Fatal(err)
	}

	overlay, err := crypto.NewOverlayAddress(privateKey1.PublicKey, 3)
	if err != nil {
		t.Fatal(err)
	}
	signer1 := crypto.NewDefaultSigner(privateKey1)

	penAddress, err := pen.NewAddress(signer1, node1ma, overlay, 3)
	if err != nil {
		t.Fatal(err)
	}

	penAddress2, err := pen.ParseAddress(node1ma.Bytes(), overlay.Bytes(), penAddress.Signature, 3)
	if err != nil {
		t.Fatal(err)
	}

	if !penAddress.Equal(penAddress2) {
		t.Fatalf("got %s expected %s", penAddress, penAddress)
	}

	bytes, err := penAddress.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	var newpen pen.Address
	if err := newpen.UnmarshalJSON(bytes); err != nil {
		t.Fatal(err)
	}

	if !newpen.Equal(penAddress) {
		t.Fatalf("got %s expected %s", newpen, penAddress)
	}
}
