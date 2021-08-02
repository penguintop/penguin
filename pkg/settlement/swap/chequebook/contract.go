// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chequebook

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/transaction"
)

type chequebookContract struct {
	address            common.Address
	transactionService transaction.Service
}

func newChequebookContract(address common.Address, transactionService transaction.Service) *chequebookContract {
	return &chequebookContract{
		address:            address,
		transactionService: transactionService,
	}
}

func (c *chequebookContract) Issuer(ctx context.Context) (common.Address, error) {
	//callData, err := chequebookABI.Pack("issuer")
	//if err != nil {
	//	return common.Address{}, err
	//}

	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "issuer"
	callData.CallArgs = ""

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return common.Address{}, err
	}

	output, err := c.transactionService.Call(ctx, &transaction.TxRequest{
		To:   &c.address,
		Data: callDataBytes,
	})
	if err != nil {
		return common.Address{}, err
	}

	issuerAddrHex, _ := xwcfmt.XwcAddrToHexAddr(string(output))
	issuerAddrBytes, _ := hex.DecodeString(issuerAddrHex)

	var addr common.Address
	addr.SetBytes(issuerAddrBytes)

	return addr, nil
}

// Balance returns the token balance of the chequebook.
func (c *chequebookContract) Balance(ctx context.Context) (*big.Int, error) {
	//callData, err := chequebookABI.Pack("balance")
	//if err != nil {
	//	return nil, err
	//}

	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "balance"
	callData.CallArgs = ""

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return nil, err
	}

	output, err := c.transactionService.Call(ctx, &transaction.TxRequest{
		To:   &c.address,
		Data: callDataBytes,
	})
	if err != nil {
		return nil, err
	}

	balance, ok := big.NewInt(0).SetString(string(output), 10)
	if !ok {
		return nil, errors.New("invalid balance value")
	}

	return balance, nil
}

func (c *chequebookContract) PaidOut(ctx context.Context, address common.Address) (*big.Int, error) {
	//callData, err := chequebookABI.Pack("paidOut", address)
	//if err != nil {
	//	return nil, err
	//}

	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "paidOut"
	callData.CallArgs, _ = xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(address[:]))

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return nil, err
	}

	output, err := c.transactionService.Call(ctx, &transaction.TxRequest{
		To:   &c.address,
		Data: callDataBytes,
	})
	if err != nil {
		return nil, err
	}

	paidOut, ok := big.NewInt(0).SetString(string(output), 10)
	if !ok {
		return nil, errors.New("invalid paidOut value")
	}

	return paidOut, nil
}

func (c *chequebookContract) TotalPaidOut(ctx context.Context) (*big.Int, error) {
	//callData, err := chequebookABI.Pack("totalPaidOut")
	//if err != nil {
	//	return nil, err
	//}

	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "totalPaidOut"
	callData.CallArgs = ""

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return nil, err
	}

	output, err := c.transactionService.Call(ctx, &transaction.TxRequest{
		To:   &c.address,
		Data: callDataBytes,
	})
	if err != nil {
		return nil, err
	}

	totalPaidOut, ok := big.NewInt(0).SetString(string(output), 10)
	if !ok {
		return nil, errors.New("invalid totalPaidOut value")
	}

	return totalPaidOut, nil
}
