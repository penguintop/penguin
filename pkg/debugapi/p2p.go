// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package debugapi

import (
	"encoding/hex"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"net/http"

	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/multiformats/go-multiaddr"
)

type addressesResponse struct {
	Overlay  penguin.Address       `json:"overlay"`
	Underlay []multiaddr.Multiaddr `json:"underlay"`
	//Ethereum     common.Address        `json:"ethereum"`
	Xwc          xwcfmt.Address `json:"xwc"`
	PublicKey    string         `json:"publicKey"`
	PSSPublicKey string         `json:"pssPublicKey"`
}

func (s *Service) addressesHandler(w http.ResponseWriter, r *http.Request) {
	// Initialize variable to json encode as [] instead null if p2p is nil
	underlay := make([]multiaddr.Multiaddr, 0)
	// Addresses endpoint is exposed before p2p service is configured
	// to provide information about other addresses.
	if s.p2p != nil {
		u, err := s.p2p.Addresses()
		if err != nil {
			s.logger.Debugf("Debug api: p2p addresses: %v", err)
			jsonhttp.InternalServerError(w, err)
			return
		}
		underlay = u
	}
	var addr xwcfmt.Address
	addr.SetBytes(s.xwcAddress[:])
	jsonhttp.OK(w, addressesResponse{
		Overlay:  s.overlay,
		Underlay: underlay,
		//Ethereum:     s.ethereumAddress,
		Xwc:          addr,
		PublicKey:    hex.EncodeToString(crypto.EncodeSecp256k1PublicKey(&s.publicKey)),
		PSSPublicKey: hex.EncodeToString(crypto.EncodeSecp256k1PublicKey(&s.pssPublicKey)),
	})
}
