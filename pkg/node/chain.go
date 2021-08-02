// Copyright 2021 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package node

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/swarm"
	"github.com/penguintop/penguin/pkg/xwcclient"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	//"github.com/ethereum/go-ethereum/ethclient"
	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/p2p/libp2p"
	"github.com/penguintop/penguin/pkg/sctx"
	"github.com/penguintop/penguin/pkg/settlement"
	"github.com/penguintop/penguin/pkg/settlement/swap"
	"github.com/penguintop/penguin/pkg/settlement/swap/chequebook"
	"github.com/penguintop/penguin/pkg/settlement/swap/swapprotocol"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/transaction"
)

const (
	maxDelay          = 1 * time.Minute
	cancellationDepth = 6
)

// InitChain will initialize the Ethereum backend at the given endpoint and
// set up the Transaction Service to interact with it using the provided signer.
func InitChain(
	ctx context.Context,
	logger logging.Logger,
	stateStore storage.StateStorer,
	endpoint string,
	signer crypto.Signer,
	blocktime uint64,
) (*xwcclient.Client, common.Address, swarm.Address, int64, transaction.Monitor, transaction.Service, error) {
	backend, err := xwcclient.Dial(endpoint)
	if err != nil {
		return nil, common.Address{}, swarm.Address{}, 0, nil, nil, fmt.Errorf("dial eth client: %w", err)
	}

	chainID, err := backend.ChainID(ctx)
	if err != nil {
		logger.Infof("could not connect to backend at %v. In a swap-enabled network a working blockchain node (for goerli network in production) is required. Check your node or specify another node using --swap-endpoint.", endpoint)
		return nil, common.Address{}, swarm.Address{}, 0, nil, nil, fmt.Errorf("get chain id: %w", err)
	}

	pollingInterval := time.Duration(blocktime) * time.Second
	overlayXwcAddress, err := signer.XwcAddress()
	if err != nil {
		return nil, common.Address{}, swarm.Address{}, 0, nil, nil, fmt.Errorf("xwc address: %w", err)
	}
	swarmNodeAddress := crypto.NewOverlayFromXwcAddress(overlayXwcAddress[:], uint64(property.CHAIN_ID_NUM))

	transactionMonitor := transaction.NewMonitor(logger, backend, overlayXwcAddress, pollingInterval, cancellationDepth)

	transactionService, err := transaction.NewService(logger, backend, signer, stateStore, big.NewInt(chainID), transactionMonitor)
	if err != nil {
		return nil, common.Address{}, swarm.Address{}, 0, nil, nil, fmt.Errorf("new transaction service: %w", err)
	}

	// Sync the with the given Xwc backend:
	isSynced, err := transaction.IsSynced(ctx, backend, maxDelay)
	if err != nil {
		return nil, common.Address{}, swarm.Address{}, 0, nil, nil, fmt.Errorf("is synced: %w", err)
	}
	if !isSynced {
		logger.Infof("waiting to sync with the XWC backend")
		err := transaction.WaitSynced(ctx, backend, maxDelay)
		if err != nil {
			return nil, common.Address{}, swarm.Address{}, 0, nil, nil, fmt.Errorf("waiting backend sync: %w", err)
		}
	}
	return backend, overlayXwcAddress, swarmNodeAddress, chainID, transactionMonitor, transactionService, nil
}

// InitChequebookFactory will initialize the chequebook factory with the given
// chain backend.
func InitChequebookFactory(
	logger logging.Logger,
	backend *xwcclient.Client,
	chainID int64,
	transactionService transaction.Service,
	factoryAddress string,
	legacyFactoryAddresses []string,
) (chequebook.Factory, error) {
	var currentFactory common.Address
	var legacyFactories []common.Address

	foundFactory, foundLegacyFactories, found := chequebook.DiscoverFactoryAddress(chainID)
	if factoryAddress == "" {
		if !found {
			return nil, errors.New("no known factory address for this network")
		}
		currentFactory = foundFactory

		conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(currentFactory[:]))

		logger.Infof("using default factory address for chain id %d: %s", chainID, conAddr)
	} else {
		factoryAddressHex, err := xwcfmt.XwcConAddrToHexAddr(factoryAddress)
		if err != nil {
			return nil, errors.New("malformed factory address")
		}
		currentFactoryBytes, _ := hex.DecodeString(factoryAddressHex)
		currentFactory.SetBytes(currentFactoryBytes)

		logger.Infof("using custom factory address: %s", factoryAddress)
	}

	if len(legacyFactoryAddresses) == 0 {
		if found {
			legacyFactories = foundLegacyFactories
		}
	} else {
		for _, legacyAddress := range legacyFactoryAddresses {
			legacyAddressHex, err := xwcfmt.XwcConAddrToHexAddr(legacyAddress)
			if err != nil {
				return nil, errors.New("malformed factory address")
			}
			legacyAddressBytes, _ := hex.DecodeString(legacyAddressHex)
			var addr common.Address
			addr.SetBytes(legacyAddressBytes)

			legacyFactories = append(legacyFactories, addr)
		}
	}

	foundEntrance, found := chequebook.DiscoverEntranceAddress(chainID)
	if !found {
		return nil, errors.New("no known entrance address for this network")
	}

	factory := chequebook.NewFactory(
		backend,
		transactionService,
		currentFactory,
		legacyFactories,
	)
	factory.SetEntranceAddress(foundEntrance)

	return factory, nil
}

// InitChequebookService will initialize the chequebook service with the given
// chequebook factory and chain backend.
func InitChequebookService(
	ctx context.Context,
	logger logging.Logger,
	stateStore storage.StateStorer,
	signer crypto.Signer,
	chainID int64,
	backend *xwcclient.Client,
	overlayXwcAddress common.Address,
	transactionService transaction.Service,
	chequebookFactory chequebook.Factory,
	initialDeposit string,
	deployGasPrice string,
) (chequebook.Service, error) {
	chequeSigner := chequebook.NewChequeSigner(signer, chainID)

	deposit, ok := new(big.Int).SetString(initialDeposit, 10)
	if !ok {
		return nil, fmt.Errorf("initial swap deposit \"%s\" cannot be parsed", initialDeposit)
	}

	if deployGasPrice != "" {
		gasPrice, ok := new(big.Int).SetString(deployGasPrice, 10)
		if !ok {
			return nil, fmt.Errorf("deploy gas price \"%s\" cannot be parsed", deployGasPrice)
		}
		ctx = sctx.SetGasPrice(ctx, gasPrice)
	}

	// check node offline caller exist, if not create new
	err := chequebookFactory.InsureOfflineCallerExist(ctx)
	if err != nil {
		return nil, fmt.Errorf("chequebook insure offline caller exist: %w", err)
	}

	// modify to transfer to factory address
	chequebookService, err := chequebook.Init(
		ctx,
		chequebookFactory,
		stateStore,
		logger,
		deposit,
		transactionService,
		backend,
		chainID,
		overlayXwcAddress,
		chequeSigner,
	)
	if err != nil {
		return nil, fmt.Errorf("chequebook init: %w", err)
	}

	return chequebookService, nil
}

func initChequeStoreCashout(
	stateStore storage.StateStorer,
	swapBackend transaction.Backend,
	chequebookFactory chequebook.Factory,
	chainID int64,
	overlayEthAddress common.Address,
	transactionService transaction.Service,
) (chequebook.ChequeStore, chequebook.CashoutService) {
	chequeStore := chequebook.NewChequeStore(
		stateStore,
		chequebookFactory,
		chainID,
		overlayEthAddress,
		transactionService,
		//TODO
		chequebook.RecoverCheque,
	)

	cashout := chequebook.NewCashoutService(
		stateStore,
		swapBackend,
		transactionService,
		chequeStore,
	)

	return chequeStore, cashout
}

// InitSwap will initialize and register the swap service.
func InitSwap(
	p2ps *libp2p.Service,
	logger logging.Logger,
	stateStore storage.StateStorer,
	networkID uint64,
	overlayEthAddress common.Address,
	chequebookService chequebook.Service,
	chequeStore chequebook.ChequeStore,
	cashoutService chequebook.CashoutService,
	accounting settlement.Accounting,
) (*swap.Service, error) {
	swapProtocol := swapprotocol.New(p2ps, logger, overlayEthAddress)
	swapAddressBook := swap.NewAddressbook(stateStore)

	swapService := swap.New(
		swapProtocol,
		logger,
		stateStore,
		chequebookService,
		chequeStore,
		swapAddressBook,
		networkID,
		cashoutService,
		p2ps,
		accounting,
	)

	swapProtocol.SetSwap(swapService)

	err := p2ps.AddProtocol(swapProtocol.Protocol())
	if err != nil {
		return nil, err
	}

	return swapService, nil
}
