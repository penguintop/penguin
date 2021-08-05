// Copyright 2021 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package postagecontract

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcclient"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"github.com/penguintop/penguin/pkg/xwctypes"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/postage"
	"github.com/penguintop/penguin/pkg/transaction"
	"github.com/ethersphere/go-storage-incentives-abi/postageabi"
	"github.com/ethersphere/go-sw3-abi/sw3abi"
)

var (
	BucketDepth = uint8(16)

	postageStampABI   = parseABI(postageabi.PostageStampABIv0_1_0)
	erc20ABI          = parseABI(sw3abi.ERC20ABIv0_3_1)
	batchCreatedTopic = postageStampABI.Events["BatchCreated"].ID

	batchCreatedTopicXwc = "BatchCreated"

	ErrBatchCreate       = errors.New("batch creation failed")
	ErrInsufficientFunds = errors.New("insufficient token balance")
	ErrInvalidDepth      = errors.New("invalid depth")
)

type Interface interface {
	CreateBatch(ctx context.Context, initialBalance *big.Int, depth uint8, label string) ([]byte, error)
}

type postageContract struct {
	owner                  common.Address
	postageContractAddress common.Address
	penTokenAddress        common.Address
	transactionService     transaction.Service
	postageService         postage.Service
}

func New(
	owner,
	postageContractAddress,
	penTokenAddress common.Address,
	transactionService transaction.Service,
	postageService postage.Service,
) Interface {
	return &postageContract{
		owner:                  owner,
		postageContractAddress: postageContractAddress,
		penTokenAddress:        penTokenAddress,
		transactionService:     transactionService,
		postageService:         postageService,
	}
}

func (c *postageContract) sendApproveTransaction(ctx context.Context, amount *big.Int) (*xwctypes.RpcTransactionReceipt, error) {
	postageAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(c.postageContractAddress[:]))
	amountStr := amount.String()

	txHash, err := c.transactionService.Send(ctx, &transaction.TxRequest{
		To: &c.penTokenAddress,
		//Data:     callData,
		GasPrice: big.NewInt(10),
		GasLimit: 100000,
		Value:    big.NewInt(0),

		TxType:     transaction.TxTypeInvokeContract,
		InvokeApi:  "approve",
		InvokeArgs: strings.Join([]string{postageAddr, amountStr}, ","),
	})
	if err != nil {
		return nil, err
	}

	receipt, err := c.transactionService.WaitForReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}

	if receipt.ExecSucceed == false {
		return nil, transaction.ErrTransactionReverted
	}

	return receipt, nil
}

func (c *postageContract) sendCreateBatchTransaction(ctx context.Context, owner common.Address, initialBalance *big.Int, depth uint8, nonce common.Hash) (*xwctypes.RpcTransactionReceipt, error) {
	ownerAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(owner[:]))
	balancePerChunkStr := initialBalance.String()
	depthStr := strconv.Itoa(int(depth))
	nonceStr := hex.EncodeToString(nonce[:])

	request := &transaction.TxRequest{
		To:       &c.postageContractAddress,
		GasPrice: big.NewInt(10),
		GasLimit: 100000,
		Value:    big.NewInt(0),

		TxType:     transaction.TxTypeInvokeContract,
		InvokeApi:  "createBatch",
		InvokeArgs: strings.Join([]string{ownerAddr, balancePerChunkStr, depthStr, nonceStr}, ","),
	}

	txHash, err := c.transactionService.Send(ctx, request)
	if err != nil {
		return nil, err
	}

	receipt, err := c.transactionService.WaitForReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}

	if receipt.ExecSucceed == false {
		return nil, transaction.ErrTransactionReverted
	}

	return receipt, nil
}

func (c *postageContract) getBalance(ctx context.Context) (*big.Int, error) {
	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "balanceOf"
	callData.CallArgs, _ = xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(c.owner[:]))

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return nil, err
	}

	request := &transaction.TxRequest{
		To:       &c.penTokenAddress,
		Data:     callDataBytes,
		GasPrice: nil,
		GasLimit: 0,
		Value:    big.NewInt(0),
	}

	data, err := c.transactionService.Call(ctx, request)
	if err != nil {
		return nil, err
	}

	balance, ok := big.NewInt(0).SetString(string(data), 10)
	if !ok {
		return nil, errors.New("invalid balance value")
	}

	return balance, nil
}

func (c *postageContract) CreateBatch(ctx context.Context, initialBalance *big.Int, depth uint8, label string) ([]byte, error) {
	if depth < BucketDepth {
		return nil, ErrInvalidDepth
	}

	totalAmount := big.NewInt(0).Mul(initialBalance, big.NewInt(int64(1<<depth)))
	balance, err := c.getBalance(ctx)
	if err != nil {
		return nil, err
	}

	if balance.Cmp(totalAmount) < 0 {
		return nil, ErrInsufficientFunds
	}

	_, err = c.sendApproveTransaction(ctx, totalAmount)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 32)
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	receipt, err := c.sendCreateBatchTransaction(ctx, c.owner, initialBalance, depth, common.BytesToHash(nonce))
	if err != nil {
		return nil, err
	}

	for _, ev := range receipt.Events {
		if ev.ContractAddress == c.postageContractAddress && ev.EventName == batchCreatedTopicXwc {
			var createdEvent batchCreatedXwcEvent

			err := json.Unmarshal([]byte(ev.EventArg), &createdEvent)
			if err != nil {
				return nil, err
			}

			batchID, _ := hex.DecodeString(createdEvent.BatchId)

			c.postageService.Add(postage.NewStampIssuer(
				label,
				c.owner.Hex(),
				batchID,
				depth,
				BucketDepth,
			))

			return batchID, nil
		}
	}

	return nil, ErrBatchCreate
}

type batchCreatedEvent struct {
	BatchId           [32]byte
	TotalAmount       *big.Int
	NormalisedBalance *big.Int
	Owner             common.Address
	Depth             uint8
}

type batchCreatedXwcEvent struct {
	BatchId           string `json:"batchId"`
	TotalAmount       uint64 `json:"totalAmount"`
	NormalisedBalance uint64 `json:"normalisedBalance"`
	Owner             string `json:"_owner"`
	Depth             uint8  `json:"_depth"`
}

func parseABI(json string) abi.ABI {
	cabi, err := abi.JSON(strings.NewReader(json))
	if err != nil {
		panic(fmt.Sprintf("error creating ABI for postage contract: %v", err))
	}
	return cabi
}

func VerifyBytecode(ctx context.Context, backend *xwcclient.Client, postageStamp common.Address) error {
	code, err := backend.CodeAt(ctx, postageStamp, nil)
	if err != nil {
		return err
	}

	if !bytes.Equal(code, property.PostageStampDeployedCodeHash) {
		return errors.New("verify byte code, invalid postage stamp contract code hash")
	}

	return nil
}

func LookupERC20Address(ctx context.Context, transactionService transaction.Service, postageContractAddress common.Address) (common.Address, error) {
	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "PenToken"
	callData.CallArgs = ""

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return common.Address{}, err
	}

	request := &transaction.TxRequest{
		To:       &postageContractAddress,
		Data:     callDataBytes,
		GasPrice: nil,
		GasLimit: 0,
		Value:    big.NewInt(0),
	}

	data, err := transactionService.Call(ctx, request)
	if err != nil {
		return common.Address{}, err
	}

	addrHex, err := xwcfmt.XwcConAddrToHexAddr(string(data))
	if err != nil {
		return common.Address{}, err
	}
	addrBytes, _ := hex.DecodeString(addrHex)

	var addr common.Address
	addr.SetBytes(addrBytes)

	return addr, nil

}
