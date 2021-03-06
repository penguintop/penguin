// Copyright 2021 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package erc20

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/transaction"
	"github.com/ethersphere/go-sw3-abi/sw3abi"
)

var (
	erc20ABI     = transaction.ParseABIUnchecked(sw3abi.ERC20ABIv0_3_1)
	errDecodeABI = errors.New("could not decode abi data")
)

type Service interface {
	BalanceOf(ctx context.Context, address common.Address) (*big.Int, error)
	Transfer(ctx context.Context, address common.Address, value *big.Int) (common.Hash, error)
	TransferToContract(ctx context.Context, address common.Address, value *big.Int) (common.Hash, error)
}

type erc20Service struct {
	backend            transaction.Backend
	transactionService transaction.Service
	address            common.Address
}

func New(backend transaction.Backend, transactionService transaction.Service, address common.Address) Service {
	return &erc20Service{
		backend:            backend,
		transactionService: transactionService,
		address:            address,
	}
}

func (c *erc20Service) BalanceOf(ctx context.Context, address common.Address) (*big.Int, error) {
	//callData, err := erc20ABI.Pack("balanceOf", address)
	//if err != nil {
	//	return nil, err
	//}
	//
	//output, err := c.transactionService.Call(ctx, &transaction.TxRequest{
	//	To:   &c.address,
	//	Data: callData,
	//})
	//if err != nil {
	//	return nil, err
	//}
	//
	//results, err := erc20ABI.Unpack("balanceOf", output)
	//if err != nil {
	//	return nil, err
	//}
	//
	//if len(results) != 1 {
	//	return nil, errDecodeABI
	//}
	//
	//balance, ok := abi.ConvertType(results[0], new(big.Int)).(*big.Int)
	//if !ok || balance == nil {
	//	return nil, errDecodeABI
	//}
	//return balance, nil

	xwcUserAddr, err := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(address[:]))
	if err != nil {
		return nil, err
	}

	balanceStr, err := c.backend.InvokeContractOffline(ctx, c.address, "balanceOf", xwcUserAddr)
	if err != nil {
		return nil, err
	}

	v, ok := big.NewInt(0).SetString(balanceStr, 10)
	if !ok {
		return big.NewInt(0), nil
	}

	return v, nil
}

func (c *erc20Service) Transfer(ctx context.Context, address common.Address, value *big.Int) (common.Hash, error) {
	//callData, err := erc20ABI.Pack("transfer", address, value)
	//if err != nil {
	//	return common.Hash{}, err
	//}
	//
	//request := &transaction.TxRequest{
	//	To:       &c.address,
	//	Data:     callData,
	//	GasPrice: sctx.GetGasPrice(ctx),
	//	GasLimit: 0,
	//	Value:    big.NewInt(0),
	//}
	//
	//txHash, err := c.transactionService.Send(ctx, request)
	//if err != nil {
	//	return common.Hash{}, err
	//}
	//
	//return txHash, nil

	erc20To, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(address[:]))

	conApi := "transfer"
	conArg := fmt.Sprintf("%s,%d", erc20To, value.Uint64())

	request := &transaction.TxRequest{
		To:       &c.address,
		GasPrice: big.NewInt(10),
		GasLimit: 100000,

		TxType:     transaction.TxTypeInvokeContract,
		InvokeApi:  conApi,
		InvokeArgs: conArg,
	}

	txHash, err := c.transactionService.Send(ctx, request)
	if err != nil {
		return common.Hash{}, err
	}

	return txHash, nil
}

func (c *erc20Service) TransferToContract(ctx context.Context, address common.Address, value *big.Int) (common.Hash, error) {
	//callData, err := erc20ABI.Pack("transfer", address, value)
	//if err != nil {
	//	return common.Hash{}, err
	//}
	//
	//request := &transaction.TxRequest{
	//	To:       &c.address,
	//	Data:     callData,
	//	GasPrice: sctx.GetGasPrice(ctx),
	//	GasLimit: 0,
	//	Value:    big.NewInt(0),
	//}
	//
	//txHash, err := c.transactionService.Send(ctx, request)
	//if err != nil {
	//	return common.Hash{}, err
	//}
	//
	//return txHash, nil

	erc20To, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(address[:]))

	conApi := "transfer"
	conArg := fmt.Sprintf("%s,%d", erc20To, value.Uint64())

	request := &transaction.TxRequest{
		To:       &c.address,
		GasPrice: big.NewInt(10),
		GasLimit: 100000,

		TxType:     transaction.TxTypeInvokeContract,
		InvokeApi:  conApi,
		InvokeArgs: conArg,
	}

	txHash, err := c.transactionService.Send(ctx, request)
	if err != nil {
		return common.Hash{}, err
	}

	return txHash, nil
}