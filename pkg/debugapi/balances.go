// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package debugapi

import (
	"errors"
	"math/big"
	"net/http"

	"github.com/penguintop/penguin/pkg/accounting"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/gorilla/mux"
)

var (
	errCantBalances   = "Cannot get balances"
	errCantBalance    = "Cannot get balance"
	errNoBalance      = "No balance for peer"
	errInvalidAddress = "Invalid address"
)

type balanceResponse struct {
	Peer    string   `json:"peer"`
	Balance *big.Int `json:"balance"`
}

type balancesResponse struct {
	Balances []balanceResponse `json:"balances"`
}

func (s *Service) balancesHandler(w http.ResponseWriter, r *http.Request) {
	balances, err := s.accounting.Balances()
	if err != nil {
		jsonhttp.InternalServerError(w, errCantBalances)
		s.logger.Debugf("Debug api: balances: %v", err)
		s.logger.Error("Debug api: can not get balances")
		return
	}

	balResponses := make([]balanceResponse, len(balances))
	i := 0
	for k := range balances {
		balResponses[i] = balanceResponse{
			Peer:    k,
			Balance: balances[k],
		}
		i++
	}

	jsonhttp.OK(w, balancesResponse{Balances: balResponses})
}

func (s *Service) peerBalanceHandler(w http.ResponseWriter, r *http.Request) {
	addr := mux.Vars(r)["peer"]
	peer, err := penguin.ParseHexAddress(addr)
	if err != nil {
		s.logger.Debugf("Debug api: balances peer: invalid peer address %s: %v", addr, err)
		s.logger.Errorf("Debug api: balances peer: invalid peer address %s", addr)
		jsonhttp.NotFound(w, errInvalidAddress)
		return
	}

	balance, err := s.accounting.Balance(peer)
	if err != nil {
		if errors.Is(err, accounting.ErrPeerNoBalance) {
			jsonhttp.NotFound(w, errNoBalance)
			return
		}
		s.logger.Debugf("Debug api: balances peer: get peer %s balance: %v", peer.String(), err)
		s.logger.Errorf("Debug api: balances peer: can't get peer %s balance", peer.String())
		jsonhttp.InternalServerError(w, errCantBalance)
		return
	}

	jsonhttp.OK(w, balanceResponse{
		Peer:    peer.String(),
		Balance: balance,
	})
}

func (s *Service) compensatedBalancesHandler(w http.ResponseWriter, r *http.Request) {
	balances, err := s.accounting.CompensatedBalances()
	if err != nil {
		jsonhttp.InternalServerError(w, errCantBalances)
		s.logger.Debugf("Debug api: compensated balances: %v", err)
		s.logger.Error("Debug api: can not get compensated balances")
		return
	}

	balResponses := make([]balanceResponse, len(balances))
	i := 0
	for k := range balances {
		balResponses[i] = balanceResponse{
			Peer:    k,
			Balance: balances[k],
		}
		i++
	}

	jsonhttp.OK(w, balancesResponse{Balances: balResponses})
}

func (s *Service) compensatedPeerBalanceHandler(w http.ResponseWriter, r *http.Request) {
	addr := mux.Vars(r)["peer"]
	peer, err := penguin.ParseHexAddress(addr)
	if err != nil {
		s.logger.Debugf("Debug api: compensated balances peer: invalid peer address %s: %v", addr, err)
		s.logger.Errorf("Debug api: compensated balances peer: invalid peer address %s", addr)
		jsonhttp.NotFound(w, errInvalidAddress)
		return
	}

	balance, err := s.accounting.CompensatedBalance(peer)
	if err != nil {
		if errors.Is(err, accounting.ErrPeerNoBalance) {
			jsonhttp.NotFound(w, errNoBalance)
			return
		}
		s.logger.Debugf("Debug api: compensated balances peer: get peer %s balance: %v", peer.String(), err)
		s.logger.Errorf("Debug api: compensated balances peer: can't get peer %s balance", peer.String())
		jsonhttp.InternalServerError(w, errCantBalance)
		return
	}

	jsonhttp.OK(w, balanceResponse{
		Peer:    peer.String(),
		Balance: balance,
	})
}
