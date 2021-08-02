// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package node defines the concept of a Bee node
// by bootstrapping and injecting all necessary
// dependencies.
package node

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/auditor"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/staking"
	"github.com/penguintop/penguin/pkg/xwcclient"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/penguintop/penguin/pkg/accounting"
	"github.com/penguintop/penguin/pkg/addressbook"
	"github.com/penguintop/penguin/pkg/api"
	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/debugapi"
	"github.com/penguintop/penguin/pkg/feeds/factory"
	"github.com/penguintop/penguin/pkg/hive"
	"github.com/penguintop/penguin/pkg/localstore"
	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/metrics"
	"github.com/penguintop/penguin/pkg/netstore"
	"github.com/penguintop/penguin/pkg/p2p"
	"github.com/penguintop/penguin/pkg/p2p/libp2p"
	"github.com/penguintop/penguin/pkg/pingpong"
	"github.com/penguintop/penguin/pkg/pinning"
	"github.com/penguintop/penguin/pkg/postage"
	"github.com/penguintop/penguin/pkg/postage/batchservice"
	"github.com/penguintop/penguin/pkg/postage/batchstore"
	"github.com/penguintop/penguin/pkg/postage/listener"
	"github.com/penguintop/penguin/pkg/postage/postagecontract"
	"github.com/penguintop/penguin/pkg/pricer"
	"github.com/penguintop/penguin/pkg/pricing"
	"github.com/penguintop/penguin/pkg/pss"
	"github.com/penguintop/penguin/pkg/puller"
	"github.com/penguintop/penguin/pkg/pullsync"
	"github.com/penguintop/penguin/pkg/pullsync/pullstorage"
	"github.com/penguintop/penguin/pkg/pusher"
	"github.com/penguintop/penguin/pkg/pushsync"
	"github.com/penguintop/penguin/pkg/recovery"
	"github.com/penguintop/penguin/pkg/resolver/multiresolver"
	"github.com/penguintop/penguin/pkg/retrieval"
	"github.com/penguintop/penguin/pkg/settlement/pseudosettle"
	"github.com/penguintop/penguin/pkg/settlement/swap"
	"github.com/penguintop/penguin/pkg/settlement/swap/chequebook"
	"github.com/penguintop/penguin/pkg/shed"
	"github.com/penguintop/penguin/pkg/steward"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/swarm"
	"github.com/penguintop/penguin/pkg/tags"
	"github.com/penguintop/penguin/pkg/topology"
	"github.com/penguintop/penguin/pkg/topology/kademlia"
	"github.com/penguintop/penguin/pkg/topology/lightnode"
	"github.com/penguintop/penguin/pkg/tracing"
	"github.com/penguintop/penguin/pkg/transaction"
	"github.com/penguintop/penguin/pkg/traversal"
	"github.com/hashicorp/go-multierror"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Bee struct {
	p2pService               io.Closer
	p2pHalter                p2p.Halter
	p2pCancel                context.CancelFunc
	apiCloser                io.Closer
	apiServer                *http.Server
	debugAPIServer           *http.Server
	resolverCloser           io.Closer
	errorLogWriter           *io.PipeWriter
	tracerCloser             io.Closer
	tagsCloser               io.Closer
	stateStoreCloser         io.Closer
	localstoreCloser         io.Closer
	topologyCloser           io.Closer
	topologyHalter           topology.Halter
	pusherCloser             io.Closer
	pullerCloser             io.Closer
	pullSyncCloser           io.Closer
	pssCloser                io.Closer
	ethClientCloser          func()
	transactionMonitorCloser io.Closer
	recoveryHandleCleanup    func()
	listenerCloser           io.Closer
	postageServiceCloser     io.Closer
}

type Options struct {
	DataDir                    string
	CacheCapacity              uint64
	DBOpenFilesLimit           uint64
	DBWriteBufferSize          uint64
	DBBlockCacheCapacity       uint64
	DBDisableSeeksCompaction   bool
	APIAddr                    string
	DebugAPIAddr               string
	Addr                       string
	NATAddr                    string
	EnableWS                   bool
	EnableQUIC                 bool
	WelcomeMessage             string
	Bootnodes                  []string
	CORSAllowedOrigins         []string
	Logger                     logging.Logger
	Standalone                 bool
	TracingEnabled             bool
	TracingEndpoint            string
	TracingServiceName         string
	GlobalPinningEnabled       bool
	PaymentThreshold           string
	PaymentTolerance           string
	PaymentEarly               string
	ResolverConnectionCfgs     []multiresolver.ConnectionConfig
	GatewayMode                bool
	BootnodeMode               bool
	SwapEndpoint               string
	SwapFactoryAddress         string
	SwapLegacyFactoryAddresses []string
	SwapInitialDeposit         string
	SwapEnable                 bool
	FullNodeMode               bool
	Transaction                string
	PostageContractAddress     string
	PriceOracleAddress         string
	BlockTime                  uint64
	DeployGasPrice             string

	//
	AuditNodeMode bool
	AuditEndpoint string
}

const (
	refreshRate = int64(1000000000000)
	basePrice   = 10
)

func NewBee(addr string, swarmAddress swarm.Address, publicKey ecdsa.PublicKey, signer crypto.Signer, networkID uint64, logger logging.Logger, libp2pPrivateKey, pssPrivateKey *ecdsa.PrivateKey, o Options) (b *Bee, err error) {
	tracer, tracerCloser, err := tracing.NewTracer(&tracing.Options{
		Enabled:     o.TracingEnabled,
		Endpoint:    o.TracingEndpoint,
		ServiceName: o.TracingServiceName,
	})
	if err != nil {
		return nil, fmt.Errorf("tracer: %w", err)
	}

	p2pCtx, p2pCancel := context.WithCancel(context.Background())
	defer func() {
		// if there's been an error on this function
		// we'd like to cancel the p2p context so that
		// incoming connections will not be possible
		if err != nil {
			p2pCancel()
		}
	}()

	b = &Bee{
		p2pCancel:      p2pCancel,
		errorLogWriter: logger.WriterLevel(logrus.ErrorLevel),
		tracerCloser:   tracerCloser,
	}

	var debugAPIService *debugapi.Service
	if o.DebugAPIAddr != "" {
		overlayXwcAddress, err := signer.XwcAddress()
		if err != nil {
			return nil, fmt.Errorf("xwc address: %w", err)
		}
		// set up basic debug api endpoints for debugging and /health endpoint
		debugAPIService = debugapi.New(swarmAddress, publicKey, pssPrivateKey.PublicKey, overlayXwcAddress, logger, tracer, o.CORSAllowedOrigins)

		debugAPIListener, err := net.Listen("tcp", o.DebugAPIAddr)
		if err != nil {
			return nil, fmt.Errorf("debug api listener: %w", err)
		}

		debugAPIServer := &http.Server{
			IdleTimeout:       30 * time.Second,
			ReadHeaderTimeout: 3 * time.Second,
			Handler:           debugAPIService,
			ErrorLog:          log.New(b.errorLogWriter, "", 0),
		}

		go func() {
			logger.Infof("debug api address: %s", debugAPIListener.Addr())

			if err := debugAPIServer.Serve(debugAPIListener); err != nil && err != http.ErrServerClosed {
				logger.Debugf("debug api server: %v", err)
				logger.Error("unable to serve debug api")
			}
		}()

		b.debugAPIServer = debugAPIServer
	}

	stateStore, err := InitStateStore(logger, o.DataDir)
	if err != nil {
		return nil, err
	}
	b.stateStoreCloser = stateStore

	err = CheckOverlayWithStore(swarmAddress, stateStore)
	if err != nil {
		return nil, err
	}

	addressbook := addressbook.New(stateStore)

	var (
		swapBackend        *xwcclient.Client
		overlayXwcAddress  common.Address
		swarmNodeAddress   swarm.Address
		chainID            int64
		transactionService transaction.Service
		transactionMonitor transaction.Monitor
		chequebookFactory  chequebook.Factory
		chequebookService  chequebook.Service
		chequeStore        chequebook.ChequeStore
		cashoutService     chequebook.CashoutService
	)
	if !o.Standalone {
		swapBackend, overlayXwcAddress, swarmNodeAddress, chainID, transactionMonitor, transactionService, err = InitChain(
			p2pCtx,
			logger,
			stateStore,
			o.SwapEndpoint,
			signer,
			o.BlockTime,
		)
		if err != nil {
			return nil, fmt.Errorf("init chain: %w", err)
		}
		b.ethClientCloser = swapBackend.Close
		b.transactionMonitorCloser = transactionMonitor
	}

	if o.SwapEnable {
		chequebookFactory, err = InitChequebookFactory(
			logger,
			swapBackend,
			chainID,
			transactionService,
			o.SwapFactoryAddress,
			o.SwapLegacyFactoryAddresses,
		)
		if err != nil {
			return nil, err
		}

		if err = chequebookFactory.VerifyBytecode(p2pCtx); err != nil {
			return nil, fmt.Errorf("factory fail: %w", err)
		}

		chequebookService, err = InitChequebookService(
			p2pCtx,
			logger,
			stateStore,
			signer,
			chainID,
			swapBackend,
			overlayXwcAddress,
			transactionService,
			chequebookFactory,
			o.SwapInitialDeposit,
			o.DeployGasPrice,
		)
		if err != nil {
			return nil, err
		}

		chequeStore, cashoutService = initChequeStoreCashout(
			stateStore,
			swapBackend,
			chequebookFactory,
			chainID,
			overlayXwcAddress,
			transactionService,
		)
	}

	lightNodes := lightnode.NewContainer(swarmAddress)

	txHash, err := getTxHash(stateStore, logger, o)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction hash: %w", err)
	}

	senderMatcher := transaction.NewMatcher(swapBackend, types.NewEIP155Signer(big.NewInt(chainID)))

	p2ps, err := libp2p.New(p2pCtx, signer, networkID, swarmAddress, addr, addressbook, stateStore, lightNodes, senderMatcher, logger, tracer, libp2p.Options{
		PrivateKey:     libp2pPrivateKey,
		NATAddr:        o.NATAddr,
		EnableWS:       o.EnableWS,
		EnableQUIC:     o.EnableQUIC,
		Standalone:     o.Standalone,
		WelcomeMessage: o.WelcomeMessage,
		FullNode:       o.FullNodeMode,
		Transaction:    txHash,
	})
	if err != nil {
		return nil, fmt.Errorf("p2p service: %w", err)
	}
	b.p2pService = p2ps
	b.p2pHalter = p2ps

	// localstore depends on batchstore
	var path string

	if o.DataDir != "" {
		logger.Infof("using datadir in: '%s'", o.DataDir)
		path = filepath.Join(o.DataDir, "localstore")
	}
	lo := &localstore.Options{
		Capacity:               o.CacheCapacity,
		OpenFilesLimit:         o.DBOpenFilesLimit,
		BlockCacheCapacity:     o.DBBlockCacheCapacity,
		WriteBufferSize:        o.DBWriteBufferSize,
		DisableSeeksCompaction: o.DBDisableSeeksCompaction,
	}

	storer, err := localstore.New(path, swarmAddress.Bytes(), lo, logger)
	if err != nil {
		return nil, fmt.Errorf("localstore: %w", err)
	}
	b.localstoreCloser = storer

	batchStore, err := batchstore.New(stateStore, storer.UnreserveBatch)
	if err != nil {
		return nil, fmt.Errorf("batchstore: %w", err)
	}
	validStamp := postage.ValidStamp(batchStore)
	post, err := postage.NewService(stateStore, chainID)
	if err != nil {
		return nil, fmt.Errorf("postage service load: %w", err)
	}
	b.postageServiceCloser = post

	var (
		postageContractService postagecontract.Interface
		batchSvc               postage.EventUpdater
		eventListener          postage.Listener

		stakingContractService staking.Interface
	)

	var postageSyncStart uint64 = 0
	if !o.Standalone {
		postageContractAddress, startBlock, found := listener.DiscoverAddresses(chainID)
		if o.PostageContractAddress != "" {
			hexAddr, err := xwcfmt.XwcConAddrToHexAddr(o.PostageContractAddress)
			if err != nil {
				return nil, errors.New("malformed postage stamp address")
			}
			bytesAddr, _ := hex.DecodeString(hexAddr)

			var addr common.Address
			addr.SetBytes(bytesAddr)

			postageContractAddress = addr

		} else if !found {
			return nil, errors.New("no known postage stamp addresses for this network")
		}
		if found {
			postageSyncStart = startBlock
		}

		eventListener = listener.New(logger, swapBackend, postageContractAddress, o.BlockTime, &pidKiller{node: b})
		b.listenerCloser = eventListener

		batchSvc = batchservice.New(stateStore, batchStore, logger, eventListener)

		err = postagecontract.VerifyBytecode(p2pCtx, swapBackend, postageContractAddress)
		if err != nil {
			return nil, err
		}

		erc20Address, err := postagecontract.LookupERC20Address(p2pCtx, transactionService, postageContractAddress)
		if err != nil {
			return nil, err
		}

		postageContractService = postagecontract.New(
			overlayXwcAddress,
			postageContractAddress,
			erc20Address,
			transactionService,
			post,
		)
	}

	if !o.Standalone {
		if natManager := p2ps.NATManager(); natManager != nil {
			// wait for nat manager to init
			logger.Debug("initializing NAT manager")
			select {
			case <-natManager.Ready():
				// this is magic sleep to give NAT time to sync the mappings
				// this is a hack, kind of alchemy and should be improved
				time.Sleep(3 * time.Second)
				logger.Debug("NAT manager initialized")
			case <-time.After(10 * time.Second):
				logger.Warning("NAT manager init timeout")
			}
		}
	}

	if o.AuditNodeMode && o.AuditEndpoint != "" {
		logger.Infof("audit mode is enabled, audit endpoint: %s, run a new auditor", o.AuditEndpoint)

		addrHex, err := xwcfmt.XwcConAddrToHexAddr(property.StakingAddress)
		if err != nil {
			return nil, err
		}
		addrByte, err := hex.DecodeString(addrHex)
		if err != nil {
			return nil, err
		}
		var stakingContractAddress common.Address
		stakingContractAddress.SetBytes(addrByte[:])

		// check code hash
		err = staking.VerifyBytecode(p2pCtx, swapBackend, stakingContractAddress)
		if err != nil {
			return nil, err
		}

		// check admin of staking contract
		_, err = staking.VerifyStakingAdmin(p2pCtx, transactionService, stakingContractAddress)
		if err != nil {
			return nil, err
		}

		erc20Address, err := staking.LookupERC20Address(p2pCtx, transactionService, stakingContractAddress)
		if err != nil {
			return nil, err
		}

		stakingContractService = staking.New(
			overlayXwcAddress,
			swarmNodeAddress,
			stakingContractAddress,
			erc20Address,
			transactionService,
		)

		staked, err := stakingContractService.QueryStaking(p2pCtx)
		if err != nil {
			return nil, err
		}
		if !staked {
			logger.Info("staking...")
			_, err := stakingContractService.Staking(p2pCtx)
			if err != nil {
				return nil, err
			}
		} else {
			logger.Info("staked before...")
		}

		adt := auditor.CreateNewAuditor(o.AuditEndpoint, storer, signer, logger)
		go adt.Run()
	}

	// Construct protocols.
	pingPong := pingpong.New(p2ps, logger, tracer)

	if err = p2ps.AddProtocol(pingPong.Protocol()); err != nil {
		return nil, fmt.Errorf("pingpong service: %w", err)
	}

	hive := hive.New(p2ps, addressbook, networkID, logger)
	if err = p2ps.AddProtocol(hive.Protocol()); err != nil {
		return nil, fmt.Errorf("hive service: %w", err)
	}

	var bootnodes []ma.Multiaddr
	if o.Standalone {
		logger.Info("Starting node in standalone mode, no p2p connections will be made or accepted")
	} else {
		for _, a := range o.Bootnodes {
			addr, err := ma.NewMultiaddr(a)
			if err != nil {
				logger.Debugf("multiaddress fail %s: %v", a, err)
				logger.Warningf("invalid bootnode address %s", a)
				continue
			}

			bootnodes = append(bootnodes, addr)
		}
	}

	var swapService *swap.Service

	metricsDB, err := shed.NewDBWrap(stateStore.DB())
	if err != nil {
		return nil, fmt.Errorf("unable to create metrics storage for kademlia: %w", err)
	}

	kad := kademlia.New(swarmAddress, addressbook, hive, p2ps, metricsDB, logger, kademlia.Options{Bootnodes: bootnodes, StandaloneMode: o.Standalone, BootnodeMode: o.BootnodeMode})
	b.topologyCloser = kad
	b.topologyHalter = kad
	hive.SetAddPeersHandler(kad.AddPeers)
	p2ps.SetPickyNotifier(kad)
	batchStore.SetRadiusSetter(kad)

	if batchSvc != nil {
		syncedChan, err := batchSvc.Start(postageSyncStart)
		if err != nil {
			return nil, fmt.Errorf("unable to start batch service: %w", err)
		}
		// wait for the postage contract listener to sync
		logger.Info("waiting to sync postage contract data, this may take a while... more info available in Debug loglevel")

		// arguably this is not a very nice solution since we dont support
		// interrupts at this stage of the application lifecycle. some changes
		// would be needed on the cmd level to support context cancellation at
		// this stage
		<-syncedChan

	}
	paymentThreshold, ok := new(big.Int).SetString(o.PaymentThreshold, 10)
	if !ok {
		return nil, fmt.Errorf("invalid payment threshold: %s", paymentThreshold)
	}

	pricer := pricer.NewFixedPricer(swarmAddress, basePrice)

	minThreshold := pricer.MostExpensive()

	pricing := pricing.New(p2ps, logger, paymentThreshold, minThreshold)

	if err = p2ps.AddProtocol(pricing.Protocol()); err != nil {
		return nil, fmt.Errorf("pricing service: %w", err)
	}

	addrs, err := p2ps.Addresses()
	if err != nil {
		return nil, fmt.Errorf("get server addresses: %w", err)
	}

	for _, addr := range addrs {
		logger.Debugf("p2p address: %s", addr)
	}

	paymentTolerance, ok := new(big.Int).SetString(o.PaymentTolerance, 10)
	if !ok {
		return nil, fmt.Errorf("invalid payment tolerance: %s", paymentTolerance)
	}
	paymentEarly, ok := new(big.Int).SetString(o.PaymentEarly, 10)
	if !ok {
		return nil, fmt.Errorf("invalid payment early: %s", paymentEarly)
	}

	acc, err := accounting.NewAccounting(
		paymentThreshold,
		paymentTolerance,
		paymentEarly,
		logger,
		stateStore,
		pricing,
		big.NewInt(refreshRate),
	)
	if err != nil {
		return nil, fmt.Errorf("accounting: %w", err)
	}

	pseudosettleService := pseudosettle.New(p2ps, logger, stateStore, acc, big.NewInt(refreshRate), p2ps)
	if err = p2ps.AddProtocol(pseudosettleService.Protocol()); err != nil {
		return nil, fmt.Errorf("pseudosettle service: %w", err)
	}

	acc.SetRefreshFunc(pseudosettleService.Pay)

	if o.SwapEnable {
		swapService, err = InitSwap(
			p2ps,
			logger,
			stateStore,
			networkID,
			overlayXwcAddress,
			chequebookService,
			chequeStore,
			cashoutService,
			acc,
		)
		if err != nil {
			return nil, err
		}
		acc.SetPayFunc(swapService.Pay)
	}

	pricing.SetPaymentThresholdObserver(acc)

	retrieve := retrieval.New(swarmAddress, storer, p2ps, kad, logger, acc, pricer, tracer)
	tagService := tags.NewTags(stateStore, logger)
	b.tagsCloser = tagService

	pssService := pss.New(pssPrivateKey, logger)
	b.pssCloser = pssService

	var ns storage.Storer
	if o.GlobalPinningEnabled {
		// create recovery callback for content repair
		recoverFunc := recovery.NewCallback(pssService)
		ns = netstore.New(storer, validStamp, recoverFunc, retrieve, logger)
	} else {
		ns = netstore.New(storer, validStamp, nil, retrieve, logger)
	}

	traversalService := traversal.New(ns)

	pinningService := pinning.NewService(storer, stateStore, traversalService)

	pushSyncProtocol := pushsync.New(swarmAddress, p2ps, storer, kad, tagService, o.FullNodeMode, pssService.TryUnwrap, validStamp, logger, acc, pricer, signer, tracer)

	// set the pushSyncer in the PSS
	pssService.SetPushSyncer(pushSyncProtocol)

	if o.GlobalPinningEnabled {
		// register function for chunk repair upon receiving a trojan message
		chunkRepairHandler := recovery.NewRepairHandler(ns, logger, pushSyncProtocol)
		b.recoveryHandleCleanup = pssService.Register(recovery.Topic, chunkRepairHandler)
	}

	pusherService := pusher.New(networkID, storer, kad, pushSyncProtocol, tagService, logger, tracer)
	b.pusherCloser = pusherService

	pullStorage := pullstorage.New(storer)

	pullSyncProtocol := pullsync.New(p2ps, pullStorage, pssService.TryUnwrap, validStamp, logger)
	b.pullSyncCloser = pullSyncProtocol

	var pullerService *puller.Puller
	if o.FullNodeMode {
		pullerService := puller.New(stateStore, kad, pullSyncProtocol, logger, puller.Options{})
		b.pullerCloser = pullerService
	}

	retrieveProtocolSpec := retrieve.Protocol()
	pushSyncProtocolSpec := pushSyncProtocol.Protocol()
	pullSyncProtocolSpec := pullSyncProtocol.Protocol()

	if o.FullNodeMode {
		logger.Info("starting in full mode")
	} else {
		logger.Info("starting in light mode")
		p2p.WithBlocklistStreams(p2p.DefaultBlocklistTime, retrieveProtocolSpec)
		p2p.WithBlocklistStreams(p2p.DefaultBlocklistTime, pushSyncProtocolSpec)
		p2p.WithBlocklistStreams(p2p.DefaultBlocklistTime, pullSyncProtocolSpec)
	}

	if err = p2ps.AddProtocol(retrieveProtocolSpec); err != nil {
		return nil, fmt.Errorf("retrieval service: %w", err)
	}
	if err = p2ps.AddProtocol(pushSyncProtocolSpec); err != nil {
		return nil, fmt.Errorf("pushsync service: %w", err)
	}
	if err = p2ps.AddProtocol(pullSyncProtocolSpec); err != nil {
		return nil, fmt.Errorf("pullsync protocol: %w", err)
	}

	multiResolver := multiresolver.NewMultiResolver(
		multiresolver.WithConnectionConfigs(o.ResolverConnectionCfgs),
		multiresolver.WithLogger(o.Logger),
	)
	b.resolverCloser = multiResolver

	var apiService api.Service
	if o.APIAddr != "" {
		// API server
		feedFactory := factory.New(ns)
		steward := steward.New(storer, traversalService, pushSyncProtocol)
		apiService = api.New(tagService, ns, multiResolver, pssService, traversalService, pinningService, feedFactory, post, postageContractService, steward, signer, logger, tracer, api.Options{
			CORSAllowedOrigins: o.CORSAllowedOrigins,
			GatewayMode:        o.GatewayMode,
			WsPingPeriod:       60 * time.Second,
		})
		apiListener, err := net.Listen("tcp", o.APIAddr)
		if err != nil {
			return nil, fmt.Errorf("api listener: %w", err)
		}

		apiServer := &http.Server{
			IdleTimeout:       30 * time.Second,
			ReadHeaderTimeout: 3 * time.Second,
			Handler:           apiService,
			ErrorLog:          log.New(b.errorLogWriter, "", 0),
		}

		go func() {
			logger.Infof("api address: %s", apiListener.Addr())

			if err := apiServer.Serve(apiListener); err != nil && err != http.ErrServerClosed {
				logger.Debugf("api server: %v", err)
				logger.Error("unable to serve api")
			}
		}()

		b.apiServer = apiServer
		b.apiCloser = apiService
	}

	if debugAPIService != nil {
		// register metrics from components
		debugAPIService.MustRegisterMetrics(p2ps.Metrics()...)
		debugAPIService.MustRegisterMetrics(pingPong.Metrics()...)
		debugAPIService.MustRegisterMetrics(acc.Metrics()...)
		debugAPIService.MustRegisterMetrics(storer.Metrics()...)

		if pullerService != nil {
			debugAPIService.MustRegisterMetrics(pullerService.Metrics()...)
		}

		debugAPIService.MustRegisterMetrics(pushSyncProtocol.Metrics()...)
		debugAPIService.MustRegisterMetrics(pusherService.Metrics()...)
		debugAPIService.MustRegisterMetrics(pullSyncProtocol.Metrics()...)
		debugAPIService.MustRegisterMetrics(pullStorage.Metrics()...)
		debugAPIService.MustRegisterMetrics(retrieve.Metrics()...)

		if bs, ok := batchStore.(metrics.Collector); ok {
			debugAPIService.MustRegisterMetrics(bs.Metrics()...)
		}

		if eventListener != nil {
			if ls, ok := eventListener.(metrics.Collector); ok {
				debugAPIService.MustRegisterMetrics(ls.Metrics()...)
			}
		}

		if pssServiceMetrics, ok := pssService.(metrics.Collector); ok {
			debugAPIService.MustRegisterMetrics(pssServiceMetrics.Metrics()...)
		}

		if apiService != nil {
			debugAPIService.MustRegisterMetrics(apiService.Metrics()...)
		}
		if l, ok := logger.(metrics.Collector); ok {
			debugAPIService.MustRegisterMetrics(l.Metrics()...)
		}

		debugAPIService.MustRegisterMetrics(pseudosettleService.Metrics()...)

		if swapService != nil {
			debugAPIService.MustRegisterMetrics(swapService.Metrics()...)
		}

		// inject dependencies and configure full debug api http path routes
		debugAPIService.Configure(p2ps, pingPong, kad, lightNodes, storer, tagService, acc, pseudosettleService, o.SwapEnable, swapService, chequebookService, batchStore)
	}

	if err := kad.Start(p2pCtx); err != nil {
		return nil, err
	}
	p2ps.Ready()

	return b, nil
}

func (b *Bee) Shutdown(ctx context.Context) error {
	var mErr error

	// halt kademlia while shutting down other
	// components.
	b.topologyHalter.Halt()

	// halt p2p layer from accepting new connections
	// while shutting down other components
	b.p2pHalter.Halt()

	// tryClose is a convenient closure which decrease
	// repetitive io.Closer tryClose procedure.
	tryClose := func(c io.Closer, errMsg string) {
		if c == nil {
			return
		}
		if err := c.Close(); err != nil {
			mErr = multierror.Append(mErr, fmt.Errorf("%s: %w", errMsg, err))
		}
	}

	tryClose(b.apiCloser, "api")

	var eg errgroup.Group
	if b.apiServer != nil {
		eg.Go(func() error {
			if err := b.apiServer.Shutdown(ctx); err != nil {
				return fmt.Errorf("api server: %w", err)
			}
			return nil
		})
	}
	if b.debugAPIServer != nil {
		eg.Go(func() error {
			if err := b.debugAPIServer.Shutdown(ctx); err != nil {
				return fmt.Errorf("debug api server: %w", err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		mErr = multierror.Append(mErr, err)
	}

	if b.recoveryHandleCleanup != nil {
		b.recoveryHandleCleanup()
	}
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		tryClose(b.pssCloser, "pss")
	}()
	go func() {
		defer wg.Done()
		tryClose(b.pusherCloser, "pusher")
	}()
	go func() {
		defer wg.Done()
		tryClose(b.pullerCloser, "puller")
	}()

	b.p2pCancel()
	go func() {
		defer wg.Done()
		tryClose(b.pullSyncCloser, "pull sync")
	}()

	wg.Wait()

	tryClose(b.p2pService, "p2p server")

	wg.Add(3)
	go func() {
		defer wg.Done()
		tryClose(b.transactionMonitorCloser, "transaction monitor")
	}()
	go func() {
		defer wg.Done()
		tryClose(b.listenerCloser, "listener")
	}()
	go func() {
		defer wg.Done()
		tryClose(b.postageServiceCloser, "postage service")
	}()

	wg.Wait()

	if c := b.ethClientCloser; c != nil {
		c()
	}

	tryClose(b.tracerCloser, "tracer")
	tryClose(b.tagsCloser, "tag persistence")
	tryClose(b.topologyCloser, "topology driver")
	tryClose(b.stateStoreCloser, "statestore")
	tryClose(b.localstoreCloser, "localstore")
	tryClose(b.errorLogWriter, "error log writer")
	tryClose(b.resolverCloser, "resolver service")

	return mErr
}

func getTxHash(stateStore storage.StateStorer, logger logging.Logger, o Options) ([]byte, error) {
	if o.Standalone {
		return nil, nil // in standalone mode tx hash is not used
	}

	var txHash common.Hash
	key := chequebook.ChequebookDeploymentKey
	if err := stateStore.Get(key, &txHash); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, errors.New("chequebook deployment transaction hash not found. Please specify the transaction hash manually.")
		}
		return nil, err
	}

	logger.Infof("using the chequebook transaction hash %x", txHash[common.HashLength-xwcfmt.HashLength:])
	return txHash.Bytes(), nil
}

// pidKiller is used to issue a forced shut down of the node from sub modules. The issue with using the
// node's Shutdown method is that it only shuts down the node and does not exit the start process
// which is waiting on the os.Signals. This is not desirable, but currently bee node cannot handle
// rate-limiting blockchain API calls properly. We will shut down the node in this case to allow the
// user to rectify the API issues (by adjusting limits or using a different one). There is no platform
// agnostic way to trigger os.Signals in go unfortunately. Which is why we will use the process.Kill
// approach which works on windows as well.
type pidKiller struct {
	node *Bee
}

func (p *pidKiller) Shutdown(ctx context.Context) error {
	err := p.node.Shutdown(ctx)
	if err != nil {
		return err
	}
	ps, err := os.FindProcess(syscall.Getpid())
	if err != nil {
		return err
	}
	return ps.Kill()
}
