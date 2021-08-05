// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pen exposes the data structure and operations
// necessary on the pen.Address type which used in the handshake
// protocol, address-book and hive protocol.
package pen

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/penguintop/penguin/pkg/crypto"
    "github.com/penguintop/penguin/pkg/penguin"

	ma "github.com/multiformats/go-multiaddr"
)

var ErrInvalidAddress = errors.New("invalid address")

// Address represents the pen address in penguin.
// It consists of a peers underlay (physical) address, overlay (topology) address and signature.
// Signature is used to verify the `Overlay/Underlay` pair, as it is based on `underlay|networkID`, signed with the public key of Overlay address
type Address struct {
	Underlay  ma.Multiaddr
	Overlay   penguin.Address
	Signature []byte
}

type addressJSON struct {
	Overlay   string `json:"overlay"`
	Underlay  string `json:"underlay"`
	Signature string `json:"signature"`
}

func NewAddress(signer crypto.Signer, underlay ma.Multiaddr, overlay penguin.Address, networkID uint64) (*Address, error) {
	underlayBinary, err := underlay.MarshalBinary()
	if err != nil {
		return nil, err
	}

	signature, err := signer.Sign(generateSignData(underlayBinary, overlay.Bytes(), networkID))
	if err != nil {
		return nil, err
	}

	return &Address{
		Underlay:  underlay,
		Overlay:   overlay,
		Signature: signature,
	}, nil
}

func ParseAddress(underlay, overlay, signature []byte, networkID uint64) (*Address, error) {
	recoveredPK, err := crypto.Recover(signature, generateSignData(underlay, overlay, networkID))
	if err != nil {
		return nil, ErrInvalidAddress
	}

	recoveredOverlay, err := crypto.NewOverlayAddress(*recoveredPK, networkID)
	if err != nil {
		return nil, ErrInvalidAddress
	}
	if !bytes.Equal(recoveredOverlay.Bytes(), overlay) {
		return nil, ErrInvalidAddress
	}

	multiUnderlay, err := ma.NewMultiaddrBytes(underlay)
	if err != nil {
		return nil, ErrInvalidAddress
	}

	return &Address{
		Underlay:  multiUnderlay,
		Overlay:   penguin.NewAddress(overlay),
		Signature: signature,
	}, nil
}

func generateSignData(underlay, overlay []byte, networkID uint64) []byte {
	networkIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(networkIDBytes, networkID)
	signData := append([]byte("pen-handshake-"), underlay...)
	signData = append(signData, overlay...)
	return append(signData, networkIDBytes...)
}

func (a *Address) Equal(b *Address) bool {
	return a.Overlay.Equal(b.Overlay) && a.Underlay.Equal(b.Underlay) && bytes.Equal(a.Signature, b.Signature)
}

func (a *Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(&addressJSON{
		Overlay:   a.Overlay.String(),
		Underlay:  a.Underlay.String(),
		Signature: base64.StdEncoding.EncodeToString(a.Signature),
	})
}

func (a *Address) UnmarshalJSON(b []byte) error {
	v := &addressJSON{}
	err := json.Unmarshal(b, v)
	if err != nil {
		return err
	}

	addr, err := penguin.ParseHexAddress(v.Overlay)
	if err != nil {
		return err
	}

	a.Overlay = addr

	m, err := ma.NewMultiaddr(v.Underlay)
	if err != nil {
		return err
	}

	a.Underlay = m
	a.Signature, err = base64.StdEncoding.DecodeString(v.Signature)
	return err
}

func (a *Address) String() string {
	return fmt.Sprintf("[Underlay: %v, Overlay %v, Signature %x]", a.Underlay, a.Overlay, a.Signature)
}

// ShortString returns shortened versions of pen address in a format: [Overlay, Underlay]
// It can be used for logging
func (a *Address) ShortString() string {
	return fmt.Sprintf("[Overlay: %s, Underlay: %s]", a.Overlay.String(), a.Underlay.String())
}
