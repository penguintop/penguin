// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transaction

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"github.com/penguintop/penguin/pkg/xwctypes"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Backend is the minimum of blockchain backend functions we need.
type Backend interface {
	ContractBackend
	TransactionReceipt(ctx context.Context, txHash common.Hash) (receipt *xwctypes.RpcTransactionReceipt, err error)
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *xwctypes.RpcTransaction, isPending bool, err error)
	BlockNumber(ctx context.Context) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*xwctypes.RpcBlock, error)
	BalanceAt(ctx context.Context, address common.Address, block *big.Int) (*big.Int, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)

	// for xwc
	IsLocked(ctx context.Context) (bool, error)
	GetAccount(ctx context.Context, acctName string) (xwctypes.RpcAccountJson, error)
	CreateAccount(ctx context.Context, acctName string) (string, error)
	RefBlockInfo(ctx context.Context) (uint16, uint32, error)
	SendXwcTransaction(ctx context.Context, tx *xwcfmt.Transaction) (common.Hash, error)
}

// IsSynced will check if we are synced with the given blockchain backend. This
// is true if the current wall clock is after the block time of last block
// with the given maxDelay as the maximum duration we can be behind the block
// time.
func IsSynced(ctx context.Context, backend Backend, maxDelay time.Duration) (bool, error) {
	number, err := backend.BlockNumber(ctx)
	if err != nil {
		return false, err
	}

	block, err := backend.BlockByNumber(ctx, big.NewInt(int64(number)))
	if err != nil {
		return false, err
	}

	blockTime := time.Unix(int64(block.Timestamp), 0)

	return blockTime.After(time.Now().UTC().Add(-maxDelay)), nil
}

// WaitSynced will wait until we are synced with the given blockchain backend,
// with the given maxDelay duration as the maximum time we can be behind the
// last block.
func WaitSynced(ctx context.Context, backend Backend, maxDelay time.Duration) error {
	for {
		synced, err := IsSynced(ctx, backend, maxDelay)
		if err != nil {
			return err
		}

		if synced {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// ParseABIUnchecked will parse a valid json abi. Only use this with string constants known to be correct.
func ParseABIUnchecked(json string) abi.ABI {
	cabi, err := abi.JSON(strings.NewReader(json))
	if err != nil {
		panic(fmt.Sprintf("error creating ABI for contract: %v", err))
	}
	return cabi
}

var (
	// ErrNoCode is returned by call and transact operations for which the requested
	// recipient contract to operate on does not exist in the state db or does not
	// have any code associated with it (i.e. suicided).
	ErrNoCode = errors.New("no contract code at given address")

	// This error is raised when attempting to perform a pending state action
	// on a backend that doesn't implement PendingContractCaller.
	ErrNoPendingState = errors.New("backend does not support pending state")

	// This error is returned by WaitDeployed if contract creation leaves an
	// empty contract behind.
	ErrNoCodeAfterDeploy = errors.New("no contract code after deployment")
)

// ContractCaller defines the methods needed to allow operating with a contract on a read
// only basis.
type ContractCaller interface {
	// CodeAt returns the code of the given account. This is needed to differentiate
	// between contract internal errors and the local chain being out of sync.
	CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)
	// ContractCall executes an Ethereum contract call with the specified data as the
	// input.
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// PendingContractCaller defines methods to perform contract calls on the pending state.
// Call will try to discover this interface when access to the pending state is requested.
// If the backend does not support the pending state, Call returns ErrNoPendingState.
type PendingContractCaller interface {
	// PendingCodeAt returns the code of the given account in the pending state.
	PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error)
	// PendingCallContract executes an Ethereum contract call against the pending state.
	PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error)
}

// ContractTransactor defines the methods needed to allow operating with a contract
// on a write only basis. Besides the transacting method, the remainder are helpers
// used when the user does not provide some needed values, but rather leaves it up
// to the transactor to decide.
type ContractTransactor interface {
	// PendingCodeAt returns the code of the given account in the pending state.
	PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error)
	// PendingNonceAt retrieves the current pending nonce associated with an account.
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
	// execution of a transaction.
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	// EstimateGas tries to estimate the gas needed to execute a specific
	// transaction based on the current pending state of the backend blockchain.
	// There is no guarantee that this is the true gas limit requirement as other
	// transactions may be added or removed by miners, but it should provide a basis
	// for setting a reasonable default.
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error)
	// SendTransaction injects the transaction into the pending pool for execution.
	SendTransaction(ctx context.Context, tx *types.Transaction) error
}

// ContractFilterer defines the methods needed to access log events using one-off
// queries or continuous event subscriptions.
type ContractFilterer interface {
	// FilterLogs executes a log filter operation, blocking during execution and
	// returning all the results in one batch.
	//
	// TODO(karalabe): Deprecate when the subscription one can return past data too.
	FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error)

	// SubscribeFilterLogs creates a background log filtering operation, returning
	// a subscription immediately, which can be used to stream the found events.
	SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
}

// DeployBackend wraps the operations needed by WaitMined and WaitDeployed.
type DeployBackend interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
}

type ContractXWC interface {
	CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)
	InvokeContractOffline(ctx context.Context, contract common.Address, api string, arg string) (string, error)
}

// ContractBackend defines the methods needed to work with contracts on a read-write basis.
type ContractBackend interface {
	ContractCaller
	ContractTransactor
	ContractFilterer

	ContractXWC
}
