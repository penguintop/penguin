// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package debugapi

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/sctx"
	"github.com/penguintop/penguin/pkg/settlement/swap/chequebook"

	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/gorilla/mux"
)

var (
	errChequebookBalance           = "cannot get chequebook balance"
	errChequebookNoAmount          = "did not specify amount"
	errChequebookNoWithdraw        = "cannot withdraw"
	errChequebookNoDeposit         = "cannot deposit"
	errChequebookInsufficientFunds = "insufficient funds"
	errCantLastChequePeer          = "cannot get last cheque for peer"
	errCantLastCheque              = "cannot get last cheque for all peers"
	errCannotCash                  = "cannot cash cheque"
	errCannotCashStatus            = "cannot get cashout status"
	errNoCashout                   = "no prior cashout"
	errNoCheque                    = "no prior cheque"
	errBadGasPrice                 = "bad gas price"
	errBadGasLimit                 = "bad gas limit"

	gasPriceHeader = "Gas-Price"
	gasLimitHeader = "Gas-Limit"
)

type chequebookBalanceResponse struct {
	TotalBalance     *big.Int `json:"totalBalance"`
	AvailableBalance *big.Int `json:"availableBalance"`
}

type chequebookAddressResponse struct {
	Address string `json:"chequebookAddress"`
}

type chequebookLastChequePeerResponse struct {
	Beneficiary string   `json:"beneficiary"`
	Chequebook  string   `json:"chequebook"`
	Payout      *big.Int `json:"payout"`
}

type chequebookLastChequesPeerResponse struct {
	Peer         string                            `json:"peer"`
	LastReceived *chequebookLastChequePeerResponse `json:"lastreceived"`
	LastSent     *chequebookLastChequePeerResponse `json:"lastsent"`
}

type chequebookLastChequesResponse struct {
	LastCheques []chequebookLastChequesPeerResponse `json:"lastcheques"`
}

func (s *Service) chequebookBalanceHandler(w http.ResponseWriter, r *http.Request) {
	balance, err := s.chequebook.Balance(r.Context())
	if err != nil {
		jsonhttp.InternalServerError(w, errChequebookBalance)
		s.logger.Debugf("Debug api: chequebook balance: %v", err)
		s.logger.Error("Debug api: cannot get chequebook balance")
		return
	}

	availableBalance, err := s.chequebook.AvailableBalance(r.Context())
	if err != nil {
		jsonhttp.InternalServerError(w, errChequebookBalance)
		s.logger.Debugf("Debug api: chequebook availableBalance: %v", err)
		s.logger.Error("Debug api: cannot get chequebook availableBalance")
		return
	}

	jsonhttp.OK(w, chequebookBalanceResponse{TotalBalance: balance, AvailableBalance: availableBalance})
}

func (s *Service) chequebookAddressHandler(w http.ResponseWriter, r *http.Request) {
	address := s.chequebook.Address()
	conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(address[:]))
	jsonhttp.OK(w, chequebookAddressResponse{Address: conAddr})
}

func (s *Service) chequebookLastPeerHandler(w http.ResponseWriter, r *http.Request) {
	addr := mux.Vars(r)["peer"]
	peer, err := penguin.ParseHexAddress(addr)
	if err != nil {
		s.logger.Debugf("Debug api: chequebook cheque peer: invalid peer address %s: %v", addr, err)
		s.logger.Errorf("Debug api: chequebook cheque peer: invalid peer address %s", addr)
		jsonhttp.NotFound(w, errInvalidAddress)
		return
	}

	var lastSentResponse *chequebookLastChequePeerResponse
	lastSent, err := s.swap.LastSentCheque(peer)
	if err != nil && err != chequebook.ErrNoCheque {
		s.logger.Debugf("Debug api: chequebook cheque peer: get peer %s last cheque: %v", peer.String(), err)
		s.logger.Errorf("Debug api: chequebook cheque peer: can't get peer %s last cheque", peer.String())
		jsonhttp.InternalServerError(w, errCantLastChequePeer)
		return
	}
	if err == nil {
		xwcAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(lastSent.Cheque.Beneficiary[:]))
		conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(lastSent.Cheque.Chequebook[:]))
		lastSentResponse = &chequebookLastChequePeerResponse{
			Beneficiary: xwcAddr,
			Chequebook:  conAddr,
			Payout:      lastSent.Cheque.CumulativePayout,
		}
	}

	var lastReceivedResponse *chequebookLastChequePeerResponse
	lastReceived, err := s.swap.LastReceivedCheque(peer)
	if err != nil && err != chequebook.ErrNoCheque {
		s.logger.Debugf("Debug api: chequebook cheque peer: get peer %s last cheque: %v", peer.String(), err)
		s.logger.Errorf("Debug api: chequebook cheque peer: can't get peer %s last cheque", peer.String())
		jsonhttp.InternalServerError(w, errCantLastChequePeer)
		return
	}
	if err == nil {
		xwcAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(lastReceived.Cheque.Beneficiary[:]))
		conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(lastReceived.Cheque.Chequebook[:]))
		lastReceivedResponse = &chequebookLastChequePeerResponse{
			Beneficiary: xwcAddr,
			Chequebook:  conAddr,
			Payout:      lastReceived.Cheque.CumulativePayout,
		}
	}

	jsonhttp.OK(w, chequebookLastChequesPeerResponse{
		Peer:         addr,
		LastReceived: lastReceivedResponse,
		LastSent:     lastSentResponse,
	})
}

func (s *Service) chequebookAllLastHandler(w http.ResponseWriter, r *http.Request) {
	lastchequessent, err := s.swap.LastSentCheques()
	if err != nil {
		s.logger.Debugf("Debug api: chequebook cheque all: get all last cheques: %v", err)
		s.logger.Errorf("Debug api: chequebook cheque all: can't get all last cheques")
		jsonhttp.InternalServerError(w, errCantLastCheque)
		return
	}
	lastchequesreceived, err := s.swap.LastReceivedCheques()
	if err != nil {
		s.logger.Debugf("Debug api: chequebook cheque all: get all last cheques: %v", err)
		s.logger.Errorf("Debug api: chequebook cheque all: can't get all last cheques")
		jsonhttp.InternalServerError(w, errCantLastCheque)
		return
	}

	lcr := make(map[string]chequebookLastChequesPeerResponse)
	for i, j := range lastchequessent {
		xwcAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(j.Cheque.Beneficiary[:]))
		conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(j.Cheque.Chequebook[:]))
		lcr[i] = chequebookLastChequesPeerResponse{
			Peer: i,
			LastSent: &chequebookLastChequePeerResponse{
				Beneficiary: xwcAddr,
				Chequebook:  conAddr,
				Payout:      j.Cheque.CumulativePayout,
			},
			LastReceived: nil,
		}
	}
	for i, j := range lastchequesreceived {
		xwcAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(j.Cheque.Beneficiary[:]))
		conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(j.Cheque.Chequebook[:]))
		if _, ok := lcr[i]; ok {
			t := lcr[i]
			t.LastReceived = &chequebookLastChequePeerResponse{
				Beneficiary: xwcAddr,
				Chequebook:  conAddr,
				Payout:      j.Cheque.CumulativePayout,
			}
			lcr[i] = t
		} else {
			lcr[i] = chequebookLastChequesPeerResponse{
				Peer:     i,
				LastSent: nil,
				LastReceived: &chequebookLastChequePeerResponse{
					Beneficiary: xwcAddr,
					Chequebook:  conAddr,
					Payout:      j.Cheque.CumulativePayout,
				},
			}
		}
	}

	lcresponses := make([]chequebookLastChequesPeerResponse, len(lcr))
	i := 0
	for k := range lcr {
		lcresponses[i] = lcr[k]
		i++
	}

	jsonhttp.OK(w, chequebookLastChequesResponse{LastCheques: lcresponses})
}

type swapCashoutResponse struct {
	TransactionHash string `json:"transactionHash"`
}

func (s *Service) swapCashoutHandler(w http.ResponseWriter, r *http.Request) {
	addr := mux.Vars(r)["peer"]
	peer, err := penguin.ParseHexAddress(addr)
	if err != nil {
		s.logger.Debugf("Debug api: cashout peer: invalid peer address %s: %v", addr, err)
		s.logger.Errorf("Debug api: cashout peer: invalid peer address %s", addr)
		jsonhttp.NotFound(w, errInvalidAddress)
		return
	}

	ctx := r.Context()
	if price, ok := r.Header[gasPriceHeader]; ok {
		p, ok := big.NewInt(0).SetString(price[0], 10)
		if !ok {
			s.logger.Error("Debug api: cashout peer: bad gas price")
			jsonhttp.BadRequest(w, errBadGasPrice)
			return
		}
		ctx = sctx.SetGasPrice(ctx, p)
	}

	if limit, ok := r.Header[gasLimitHeader]; ok {
		l, err := strconv.ParseUint(limit[0], 10, 64)
		if err != nil {
			s.logger.Debugf("Debug api: cashout peer: bad gas limit: %v", err)
			s.logger.Error("Debug api: cashout peer: bad gas limit")
			jsonhttp.BadRequest(w, errBadGasLimit)
			return
		}
		ctx = sctx.SetGasLimit(ctx, l)
	}

	txHash, err := s.swap.CashCheque(ctx, peer)
	if err != nil {
		s.logger.Debugf("Debug api: cashout peer: cannot cash %s: %v", addr, err)
		s.logger.Errorf("Debug api: cashout peer: cannot cash %s", addr)
		jsonhttp.InternalServerError(w, errCannotCash)
		return
	}

	jsonhttp.OK(w, swapCashoutResponse{TransactionHash: txHash.String()})
}

type swapCashoutStatusResult struct {
	Recipient  common.Address `json:"recipient"`
	LastPayout *big.Int       `json:"lastPayout"`
	Bounced    bool           `json:"bounced"`
}

type swapCashoutStatusResponse struct {
	Peer            penguin.Address                   `json:"peer"`
	Cheque          *chequebookLastChequePeerResponse `json:"lastCashedCheque"`
	TransactionHash *common.Hash                      `json:"transactionHash"`
	Result          *swapCashoutStatusResult          `json:"result"`
	UncashedAmount  *big.Int                          `json:"uncashedAmount"`
}

func (s *Service) swapCashoutStatusHandler(w http.ResponseWriter, r *http.Request) {
	addr := mux.Vars(r)["peer"]
	peer, err := penguin.ParseHexAddress(addr)
	if err != nil {
		s.logger.Debugf("Debug api: cashout status peer: invalid peer address %s: %v", addr, err)
		s.logger.Errorf("Debug api: cashout status peer: invalid peer address %s", addr)
		jsonhttp.NotFound(w, errInvalidAddress)
		return
	}

	status, err := s.swap.CashoutStatus(r.Context(), peer)
	if err != nil {
		if errors.Is(err, chequebook.ErrNoCheque) {
			s.logger.Debugf("Debug api: cashout status peer: %v, err: %v", addr, err)
			s.logger.Errorf("Debug api: cashout status peer: %s", addr)
			jsonhttp.NotFound(w, errNoCheque)
			return
		}
		if errors.Is(err, chequebook.ErrNoCashout) {
			s.logger.Debugf("Debug api: cashout status peer: %v, err: %v", addr, err)
			s.logger.Errorf("Debug api: cashout status peer: %s", addr)
			jsonhttp.NotFound(w, errNoCashout)
			return
		}
		s.logger.Debugf("Debug api: cashout status peer: cannot get status %s: %v", addr, err)
		s.logger.Errorf("Debug api: cashout status peer: cannot get status %s", addr)
		jsonhttp.InternalServerError(w, errCannotCashStatus)
		return
	}

	var result *swapCashoutStatusResult
	var txHash *common.Hash
	var chequeResponse *chequebookLastChequePeerResponse
	if status.Last != nil {
		if status.Last.Result != nil {
			result = &swapCashoutStatusResult{
				Recipient:  status.Last.Result.Recipient,
				LastPayout: status.Last.Result.TotalPayout,
				Bounced:    status.Last.Result.Bounced,
			}
		}

		xwcAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(status.Last.Cheque.Beneficiary[:]))
		conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(status.Last.Cheque.Chequebook[:]))
		chequeResponse = &chequebookLastChequePeerResponse{
			Chequebook:  conAddr,
			Payout:      status.Last.Cheque.CumulativePayout,
			Beneficiary: xwcAddr,
		}
		txHash = &status.Last.TxHash
	}

	jsonhttp.OK(w, swapCashoutStatusResponse{
		Peer:            peer,
		TransactionHash: txHash,
		Cheque:          chequeResponse,
		Result:          result,
		UncashedAmount:  status.UncashedAmount,
	})
}

//type chequebookTxResponse struct {
//	TransactionHash common.Hash `json:"transactionHash"`
//}

type chequebookTxResponse struct {
	TransactionHash string `json:"transactionHash"`
}

func (s *Service) chequebookWithdrawHandler(w http.ResponseWriter, r *http.Request) {
	amountStr := r.URL.Query().Get("amount")
	if amountStr == "" {
		jsonhttp.BadRequest(w, errChequebookNoAmount)
		s.logger.Error("Debug api: no withdraw amount")
		return
	}

	amount, ok := big.NewInt(0).SetString(amountStr, 10)
	if !ok {
		jsonhttp.BadRequest(w, errChequebookNoAmount)
		s.logger.Error("Debug api: invalid withdraw amount")
		return
	}

	ctx := r.Context()
	if price, ok := r.Header[gasPriceHeader]; ok {
		p, ok := big.NewInt(0).SetString(price[0], 10)
		if !ok {
			s.logger.Error("Debug api: withdraw: bad gas price")
			jsonhttp.BadRequest(w, errBadGasPrice)
			return
		}
		ctx = sctx.SetGasPrice(ctx, p)
	}

	txHash, err := s.chequebook.Withdraw(ctx, amount)
	if errors.Is(err, chequebook.ErrInsufficientFunds) {
		jsonhttp.BadRequest(w, errChequebookInsufficientFunds)
		s.logger.Debugf("Debug api: chequebook withdraw: %v", err)
		s.logger.Error("Debug api: cannot withdraw from chequebook")
		return
	}
	if err != nil {
		jsonhttp.InternalServerError(w, errChequebookNoWithdraw)
		s.logger.Debugf("Debug api: chequebook withdraw: %v", err)
		s.logger.Error("Debug api: cannot withdraw from chequebook")
		return
	}

	txHashStr := fmt.Sprintf("%x", txHash[12:])
	jsonhttp.OK(w, chequebookTxResponse{TransactionHash: txHashStr})
}

func (s *Service) chequebookDepositHandler(w http.ResponseWriter, r *http.Request) {
	amountStr := r.URL.Query().Get("amount")
	if amountStr == "" {
		jsonhttp.BadRequest(w, errChequebookNoAmount)
		s.logger.Error("Debug api: no deposit amount")
		return
	}

	amount, ok := big.NewInt(0).SetString(amountStr, 10)
	if !ok {
		jsonhttp.BadRequest(w, errChequebookNoAmount)
		s.logger.Error("Debug api: invalid deposit amount")
		return
	}

	ctx := r.Context()
	if price, ok := r.Header[gasPriceHeader]; ok {
		p, ok := big.NewInt(0).SetString(price[0], 10)
		if !ok {
			s.logger.Error("Debug api: deposit: bad gas price")
			jsonhttp.BadRequest(w, errBadGasPrice)
			return
		}
		ctx = sctx.SetGasPrice(ctx, p)
	}

	txHash, err := s.chequebook.Deposit(ctx, amount)
	if errors.Is(err, chequebook.ErrInsufficientFunds) {
		jsonhttp.BadRequest(w, errChequebookInsufficientFunds)
		s.logger.Debugf("Debug api: chequebook deposit: %v", err)
		s.logger.Error("Debug api: cannot deposit from chequebook")
		return
	}
	if err != nil {
		jsonhttp.InternalServerError(w, errChequebookNoDeposit)
		s.logger.Debugf("Debug api: chequebook deposit: %v", err)
		s.logger.Error("Debug api: cannot deposit from chequebook")
		return
	}

	txHashStr := fmt.Sprintf("%x", txHash[12:])
	jsonhttp.OK(w, chequebookTxResponse{TransactionHash: txHashStr})
}
