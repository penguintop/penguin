// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcclient"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"github.com/penguintop/penguin/pkg/xwcspv"
	"github.com/penguintop/penguin/pkg/xwctypes"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/storage"
	"golang.org/x/net/context"
)

const (
	noncePrefix             = "transaction_nonce_"
	storedTransactionPrefix = "transaction_stored_"
)

const (
	TxTypeTransfer           = 100
	TxTypeTransferToContract = 101
	TxTypeInvokeContract     = 102
)

var (
	// ErrTransactionReverted denotes that the sent transaction has been
	// reverted.
	ErrTransactionReverted = errors.New("transaction reverted")
)

// TxRequest describes a request for a transaction that can be executed.
type TxRequest struct {
	To       *common.Address // recipient of the transaction
	Data     []byte          // transaction data
	GasPrice *big.Int        // gas price or nil if suggested gas price should be used
	GasLimit uint64          // gas limit or 0 if it should be estimated
	Value    *big.Int        // amount of wei to send

	TxType     int    // 100: transfer   101: transfer to contract   102: invoke contract
	Memo       string // transfer memo
	InvokeApi  string // used for contract invoke
	InvokeArgs string // used for contract invoke
}

type storedTransaction struct {
	To       *common.Address // recipient of the transaction
	Data     []byte          // transaction data
	GasPrice *big.Int        // used gas price
	GasLimit uint64          // used gas limit
	Value    *big.Int        // amount of wei to send
	Nonce    uint64          // used nonce
}

// Service is the service to send transactions. It takes care of gas price, gas
// limit and nonce management.
type Service interface {
	// Send creates a transaction based on the request and sends it.
	Send(ctx context.Context, request *TxRequest) (txHash common.Hash, err error)
	// Call simulate a transaction based on the request.
	Call(ctx context.Context, request *TxRequest) (result []byte, err error)
	// WaitForReceipt waits until either the transaction with the given hash has been mined or the context is cancelled.
	// This is only valid for transaction sent by this service.
	WaitForReceipt(ctx context.Context, txHash common.Hash) (receipt *xwctypes.RpcTransactionReceipt, err error)
	// WatchSentTransaction start watching the given transaction.
	// This wraps the monitors watch function by loading the correct nonce from the store.
	// This is only valid for transaction sent by this service.
	WatchSentTransaction(txHash common.Hash) (<-chan xwctypes.RpcTransactionReceipt, <-chan error, error)
}

type transactionService struct {
	lock sync.Mutex

	logger  logging.Logger
	backend Backend
	signer  crypto.Signer
	sender  common.Address
	store   storage.StateStorer
	chainID *big.Int
	monitor Monitor
}

// NewService creates a new transaction service.
func NewService(logger logging.Logger, backend Backend, signer crypto.Signer, store storage.StateStorer, chainID *big.Int, monitor Monitor) (Service, error) {
	senderAddress, err := signer.XwcAddress()
	if err != nil {
		return nil, err
	}

	return &transactionService{
		logger:  logger,
		backend: backend,
		signer:  signer,
		sender:  senderAddress,
		store:   store,
		chainID: chainID,
		monitor: monitor,
	}, nil
}

// Send creates and signs a transaction based on the request and sends it.
func (t *transactionService) Send(ctx context.Context, request *TxRequest) (txHash common.Hash, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	refBlockNum, refBlockPrefix, err := t.backend.RefBlockInfo(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	pubKeyHex, _ := t.signer.CompressedPubKeyHex()

	from, _ := t.signer.XwcAddress()
	xwcFrom, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(from[:]))
	gasPrice := uint64(request.GasPrice.Int64())
	gasLimit := uint64(request.GasLimit)
	fee := uint64(2000000)

	var tx *xwcfmt.Transaction

	if request.TxType == TxTypeTransferToContract {
		// transfer to contract
		to := *(request.To)
		amount := uint64(request.Value.Int64())
		xwcConAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(to[:]))
		_, tx, _ = xwcspv.XwcBuildTxTransferToContract(refBlockNum, refBlockPrefix, xwcFrom, pubKeyHex, xwcConAddr, fee, gasPrice, gasLimit, amount, "")

		fmt.Printf("transfer XWC from %s to contract %s\n", xwcFrom, xwcConAddr)
	} else if request.TxType == TxTypeTransfer {
		// transfer to normal address
		to := *(request.To)
		amount := uint64(request.Value.Int64())
		xwcTo, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(to[:]))
		_, tx, _ = xwcspv.XwcBuildTxTransfer(refBlockNum, refBlockPrefix, xwcFrom, xwcTo, amount, fee, request.Memo)

		fmt.Printf("transfer XWC from %s to %s\n", xwcFrom, xwcTo)
	} else if request.TxType == TxTypeInvokeContract {
		// erc20 transfer
		xwcConAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(request.To[:]))
		_, tx, _ = xwcspv.XwcBuildTxInvokeContract(refBlockNum, refBlockPrefix, xwcFrom, pubKeyHex, xwcConAddr, fee, gasPrice, gasLimit, request.InvokeApi, request.InvokeArgs)

		fmt.Printf("%s invoke contract %s, Api: %s, Args: %s\n", xwcFrom, xwcConAddr, request.InvokeApi, request.InvokeArgs)
	} else {
		return common.Hash{}, errors.New("invalid TxType")
	}

	txSigned, err := t.signer.SignXwcTx(tx, property.CHAIN_ID)
	if err != nil {
		return common.Hash{}, err
	}

	txJson, _ := json.Marshal(txSigned)
	fmt.Printf("signed transaction: %s", string(txJson))

	txHash, err = t.backend.SendXwcTransaction(ctx, txSigned)
	if err != nil {
		return common.Hash{}, err
	}

	return txHash, nil
}

func (t *transactionService) Call(ctx context.Context, request *TxRequest) ([]byte, error) {
	msg := ethereum.CallMsg{
		From:     t.sender,
		To:       request.To,
		Data:     request.Data,
		GasPrice: request.GasPrice,
		Gas:      request.GasLimit,
		Value:    request.Value,
	}
	data, err := t.backend.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *transactionService) getStoredTransaction(txHash common.Hash) (*storedTransaction, error) {
	var tx storedTransaction
	err := t.store.Get(storedTransactionKey(txHash), &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

// prepareTransaction creates a signable transaction based on a request.
func prepareTransaction(ctx context.Context, request *TxRequest, from common.Address, backend Backend, nonce uint64) (tx *types.Transaction, err error) {
	var gasLimit uint64
	if request.GasLimit == 0 {
		gasLimit, err = backend.EstimateGas(ctx, ethereum.CallMsg{
			From: from,
			To:   request.To,
			Data: request.Data,
		})
		if err != nil {
			return nil, err
		}

		gasLimit += gasLimit / 5 // add 20% on top

	} else {
		gasLimit = request.GasLimit
	}

	var gasPrice *big.Int
	if request.GasPrice == nil {
		gasPrice, err = backend.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		gasPrice = request.GasPrice
	}

	if request.To != nil {
		return types.NewTransaction(
			nonce,
			*request.To,
			request.Value,
			gasLimit,
			gasPrice,
			request.Data,
		), nil
	}

	return types.NewContractCreation(
		nonce,
		request.Value,
		gasLimit,
		gasPrice,
		request.Data,
	), nil
}

func (t *transactionService) nonceKey() string {
	return fmt.Sprintf("%s%x", noncePrefix, t.sender)
}

func storedTransactionKey(txHash common.Hash) string {
	return fmt.Sprintf("%s%x", storedTransactionPrefix, txHash)
}

func (t *transactionService) nextNonce(ctx context.Context) (uint64, error) {
	onchainNonce, err := t.backend.PendingNonceAt(ctx, t.sender)
	if err != nil {
		return 0, err
	}

	var nonce uint64
	err = t.store.Get(t.nonceKey(), &nonce)
	if err != nil {
		// If no nonce was found locally used whatever we get from the backend.
		if errors.Is(err, storage.ErrNotFound) {
			return onchainNonce, nil
		}
		return 0, err
	}

	// If the nonce onchain is larger than what we have there were external
	// transactions and we need to update our nonce.
	if onchainNonce > nonce {
		return onchainNonce, nil
	}
	return nonce, nil
}

func (t *transactionService) putNonce(nonce uint64) error {
	return t.store.Put(t.nonceKey(), nonce)
}

// WaitForReceipt waits until either the transaction with the given hash has
// been mined or the context is cancelled.
func (t *transactionService) WaitForReceipt(ctx context.Context, txHash common.Hash) (receipt *xwctypes.RpcTransactionReceipt, err error) {
	for {
		receipt, err := t.backend.TransactionReceipt(ctx, txHash)

		if err != nil {
			if err != xwcclient.ErrTransactionReceiptNotFound {
				return nil, err
			} else {
				// wait for transaction confirmed
				select {
				case <-time.After(time.Duration(5 * time.Second)):
					continue
				}
			}
		}

		return receipt, nil
	}
}

func (t *transactionService) WatchSentTransaction(txHash common.Hash) (<-chan xwctypes.RpcTransactionReceipt, <-chan error, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// loading the tx here guarantees it was in fact sent from this transaction service
	// also it allows us to avoid having to load the transaction during the watch loop
	storedTransaction, err := t.getStoredTransaction(txHash)
	if err != nil {
		return nil, nil, err
	}

	return t.monitor.WatchTransaction(txHash, storedTransaction.Nonce)
}
