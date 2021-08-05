// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chequebook

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/transaction"
	"github.com/ethersphere/go-sw3-abi/sw3abi"
	"golang.org/x/net/context"
)

var (
	ErrInvalidFactory       = errors.New("not a valid factory contract")
	ErrNotDeployedByFactory = errors.New("chequebook not deployed by factory")
	errDecodeABI            = errors.New("could not decode abi data")

	factoryABI                  = transaction.ParseABIUnchecked(sw3abi.SimpleSwapFactoryABIv0_4_0)
	simpleSwapDeployedEventType = factoryABI.Events["SimpleSwapDeployed"]

	ErrInvalidChequeBook = errors.New("not a valid cheque book contract")
)

// Factory is the main interface for interacting with the chequebook factory.
type Factory interface {
	// ERC20Address returns the token for which this factory deploys chequebooks.
	ERC20Address(ctx context.Context) (common.Address, error)
	// Deploy deploys a new chequebook and returns once the transaction has been submitted.
	Deploy(ctx context.Context, issuer common.Address, defaultHardDepositTimeoutDuration *big.Int, nonce common.Hash) (common.Hash, error)
	// WaitDeployed waits for the deployment transaction to confirm and returns the chequebook address
	WaitDeployed(ctx context.Context, txHash common.Hash) (common.Address, error)
	// VerifyBytecode checks that the factory is valid.
	VerifyBytecode(ctx context.Context) error
	// VerifyChequebook checks that the supplied chequebook has been deployed by this factory.
	VerifyChequebook(ctx context.Context, chequebook common.Address) error

	InsureOfflineCallerExist(ctx context.Context) error

	QueryUserChequeBook(ctx context.Context, userAddr common.Address) (*common.Address, error)

	SetEntranceAddress(entranceAddress common.Address)

	VerifyChequebookOwner(ctx context.Context, chequebook common.Address, chequebookOwner common.Address) error
}

type factory struct {
	backend            transaction.Backend
	transactionService transaction.Service
	address            common.Address   // address of the factory to use for deployments
	legacyAddresses    []common.Address // addresses of old factories which were allowed for deployment

	entranceAddress common.Address
}

func (c *factory) InsureOfflineCallerExist(ctx context.Context) error {
	// wallet is unlock
	isLocked, err := c.backend.IsLocked(ctx)
	if err != nil {
		return err
	}
	if isLocked {
		return errors.New("node wallet should be unlocked first")
	}

	acct, err := c.backend.GetAccount(ctx, property.OfflineCaller)
	if err != nil {
		return err
	}
	if acct.Addr == "" {
		_, _ = c.backend.CreateAccount(ctx, property.OfflineCaller)
	}

	return nil
}

func (c *factory) QueryUserChequeBook(ctx context.Context, userAddr common.Address) (*common.Address, error) {
	xwcUserAddr, err := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(userAddr[:]))
	if err != nil {
		return nil, err
	}

	chequeAddr, err := c.backend.InvokeContractOffline(ctx, c.address, "deploySimpleSwap", xwcUserAddr)
	if err != nil {
		return nil, err
	}

	if chequeAddr == "" {
		return nil, nil
	}

	chequeAddrHex, err := xwcfmt.XwcConAddrToHexAddr(chequeAddr)
	if err != nil {
		return nil, err
	}

	var addr common.Address
	chequeAddrBytes, _ := hex.DecodeString(chequeAddrHex)
	addr.SetBytes(chequeAddrBytes)

	return &addr, nil
}

type simpleSwapDeployedEvent struct {
	ContractAddress common.Address
}

// the bytecode of factories which can be used for deployment
//var currentDeployVersion []byte = common.FromHex(sw3abi.SimpleSwapFactoryDeployedBinv0_4_0)

// the bytecode of factories from which we accept chequebooks
//var supportedVersions = [][]byte{
//	currentDeployVersion,
//	common.FromHex(sw3abi.SimpleSwapFactoryDeployedBinv0_3_1),
//}

// NewFactory creates a new factory service for the provided factory contract.
func NewFactory(backend transaction.Backend, transactionService transaction.Service, address common.Address, legacyAddresses []common.Address) Factory {
	return &factory{
		backend:            backend,
		transactionService: transactionService,
		address:            address,
		legacyAddresses:    legacyAddresses,
	}
}

func (c *factory) SetEntranceAddress(entranceAddress common.Address) {
	c.entranceAddress = entranceAddress
}

// Deploy deploys a new chequebook and returns once the transaction has been submitted.
func (c *factory) Deploy(ctx context.Context, issuer common.Address, defaultHardDepositTimeoutDuration *big.Int, nonce common.Hash) (common.Hash, error) {
	callData, err := factoryABI.Pack("deploySimpleSwap", issuer, big.NewInt(0).Set(defaultHardDepositTimeoutDuration), nonce)
	if err != nil {
		return common.Hash{}, err
	}

	request := &transaction.TxRequest{
		To:       &c.address,
		Data:     callData,
		GasPrice: big.NewInt(10),
		GasLimit: 100000,
		Value:    big.NewInt(0),
	}

	txHash, err := c.transactionService.Send(ctx, request)
	if err != nil {
		return common.Hash{}, err
	}

	return txHash, nil
}

// WaitDeployed waits for the deployment transaction to confirm and returns the chequebook address
// no use
func (c *factory) WaitDeployed(ctx context.Context, txHash common.Hash) (common.Address, error) {
	//receipt, err := c.transactionService.WaitForReceipt(ctx, txHash)
	//if err != nil {
	//	return common.Address{}, err
	//}
	//
	//var event simpleSwapDeployedEvent
	//err = transaction.FindSingleEvent(&factoryABI, receipt, c.address, simpleSwapDeployedEventType, &event)
	//if err != nil {
	//	return common.Address{}, fmt.Errorf("contract deployment failed: %w", err)
	//}
	//
	//return event.ContractAddress, nil

	return common.Address{}, nil
}

// VerifyBytecode checks that the factory is valid.
func (c *factory) VerifyBytecode(ctx context.Context) (err error) {
	code, err := c.backend.CodeAt(ctx, c.address, nil)
	if err != nil {
		return err
	}

	if !bytes.Equal(code, property.FactoryDeployedCodeHash) {
		return ErrInvalidFactory
	}

	//LOOP:
	//	for _, factoryAddress := range c.legacyAddresses {
	//		code, err := c.backend.CodeAt(ctx, factoryAddress, nil)
	//		if err != nil {
	//			return err
	//		}
	//
	//		for _, referenceCode := range supportedVersions {
	//			if bytes.Equal(code, referenceCode) {
	//				continue LOOP
	//			}
	//		}
	//
	//		return fmt.Errorf("failed to find matching bytecode for factory %x: %w", factoryAddress, ErrInvalidFactory)
	//	}

	return nil
}

func (c *factory) verifyChequebookAgainstFactory(ctx context.Context, factory common.Address, chequebook common.Address) (bool, error) {
	//callData, err := factoryABI.Pack("deployedContracts", chequebook)
	//if err != nil {
	//	return false, err
	//}
	//
	//output, err := c.transactionService.Call(ctx, &transaction.TxRequest{
	//	To:   &factory,
	//	Data: callData,
	//})
	//if err != nil {
	//	return false, err
	//}
	//
	//results, err := factoryABI.Unpack("deployedContracts", output)
	//if err != nil {
	//	return false, err
	//}
	//
	//if len(results) != 1 {
	//	return false, errDecodeABI
	//}
	//
	//deployed, ok := abi.ConvertType(results[0], new(bool)).(*bool)
	//if !ok || deployed == nil {
	//	return false, errDecodeABI
	//}
	//if !*deployed {
	//	return false, nil
	//}

	// verify byte code
	code, err := c.backend.CodeAt(ctx, chequebook, nil)
	if err != nil {
		return false, err
	}

	if !bytes.Equal(code, property.ChequeBookDeployedCodeHash) {
		return false, ErrInvalidChequeBook
	}

	return true, nil
}

func (c *factory) VerifyChequebookOwner(ctx context.Context, chequebook common.Address, chequebookOwner common.Address) error {
	// verify cheque book contrace owner
	chequeBookOwner, err := c.backend.InvokeContractOffline(ctx, chequebook, "admin", "")
	if err != nil {
		return err
	}

	chequeBookOwnerHex, err := xwcfmt.XwcAddrToHexAddr(chequeBookOwner)
	if err != nil {
		return err
	}

	chequeBookOwnerBytes, _ := hex.DecodeString(chequeBookOwnerHex)
	var addr common.Address
	addr.SetBytes(chequeBookOwnerBytes)

	if addr != chequebookOwner {
		return errors.New("verify chequebook owner not match")
	}

	return nil
}

// VerifyChequebook checks that the supplied chequebook has been deployed by a supported factory.
func (c *factory) VerifyChequebook(ctx context.Context, chequebook common.Address) error {
	deployed, err := c.verifyChequebookAgainstFactory(ctx, c.address, chequebook)
	if err != nil {
		return err
	}
	if deployed {
		return nil
	}

	for _, factoryAddress := range c.legacyAddresses {
		deployed, err := c.verifyChequebookAgainstFactory(ctx, factoryAddress, chequebook)
		if err != nil {
			return err
		}
		if deployed {
			return nil
		}
	}

	return ErrNotDeployedByFactory
}

// ERC20Address returns the token for which this factory deploys chequebooks.
func (c *factory) ERC20Address(ctx context.Context) (common.Address, error) {
	//callData, err := factoryABI.Pack("ERC20Address")
	//if err != nil {
	//	return common.Address{}, err
	//}
	//
	//output, err := c.transactionService.Call(ctx, &transaction.TxRequest{
	//	To:   &c.address,
	//	Data: callData,
	//})
	//if err != nil {
	//	return common.Address{}, err
	//}
	//
	//results, err := factoryABI.Unpack("ERC20Address", output)
	//if err != nil {
	//	return common.Address{}, err
	//}
	//
	//if len(results) != 1 {
	//	return common.Address{}, errDecodeABI
	//}
	//
	//erc20Address, ok := abi.ConvertType(results[0], new(common.Address)).(*common.Address)
	//if !ok || erc20Address == nil {
	//	return common.Address{}, errDecodeABI
	//}
	//return *erc20Address, nil

	erc20Addr, err := c.backend.InvokeContractOffline(ctx, c.address, "getErc20Address", "")
	if err != nil {
		return common.Address{}, err
	}

	erc20AddrHex, err := xwcfmt.XwcConAddrToHexAddr(erc20Addr)
	if err != nil {
		return common.Address{}, err
	}

	erc20AddrBytes, _ := hex.DecodeString(erc20AddrHex)

	var addr common.Address
	addr.SetBytes(erc20AddrBytes)

	return addr, nil
}

// DiscoverFactoryAddress returns the canonical factory for this chainID
func DiscoverFactoryAddress(chainID int64) (currentFactory common.Address, legacyFactories []common.Address, found bool) {
	factoryAddrHex, err := xwcfmt.XwcConAddrToHexAddr(property.FactoryAddress)
	if err != nil {
		return common.Address{}, nil, false
	}
	factoryAddr, _ := hex.DecodeString(factoryAddrHex)

	var addr common.Address
	addr.SetBytes(factoryAddr)

	return addr, []common.Address{}, true
}

func DiscoverEntranceAddress(chainID int64) (currentEntrance common.Address, found bool) {
	entranceAddrHex, err := xwcfmt.XwcAddrToHexAddr(property.EntranceAddress)
	if err != nil {
		return common.Address{}, false
	}
	entranceAddr, _ := hex.DecodeString(entranceAddrHex)

	var addr common.Address
	addr.SetBytes(entranceAddr)

	return addr, true
}
