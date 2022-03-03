// Copyright 2021 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package listener

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"github.com/penguintop/penguin/pkg/xwctypes"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/postage"
	"github.com/ethersphere/go-storage-incentives-abi/postageabi"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	blockPage = 5000 // how many blocks to sync every time we page
	tailSize  = 4    // how many blocks to tail from the tip of the chain
)

var (
	postageStampABI = parseABI(postageabi.PostageStampABIv0_2_0)
	// batchCreatedTopic is the postage contract's batch created event topic
	batchCreatedTopic = postageStampABI.Events["BatchCreated"].ID
	// batchTopupTopic is the postage contract's batch topup event topic
	batchTopupTopic = postageStampABI.Events["BatchTopUp"].ID
	// batchDepthIncreaseTopic is the postage contract's batch dilution event topic
	batchDepthIncreaseTopic = postageStampABI.Events["BatchDepthIncrease"].ID
	// priceUpdateTopic is the postage contract's price update event topic
	priceUpdateTopic = postageStampABI.Events["PriceUpdate"].ID

	batchCreatedTopicXWC       = "BatchCreated"
	batchTopupTopicXWC         = "BatchTopUp"
	batchDepthIncreaseTopicXWC = "BatchDepthIncrease"
	priceUpdateTopicXWC        = "PriceUpdate"
)

type ContractFilterer interface {
	// FilterLogs executes a log filter operation, blocking during execution and
	// returning all the results in one batch.
	//
	// TODO(karalabe): Deprecate when the subscription one can return past data too.
	FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error)

	// SubscribeFilterLogs creates a background log filtering operation, returning
	// a subscription immediately, which can be used to stream the found events.
	SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)

	GetContractEventsInRange(ctx context.Context, account common.Address, start uint64, to uint64) ([]xwctypes.RpcEventJson, error)
}

type BlockHeightContractFilterer interface {
	ContractFilterer
	BlockNumber(context.Context) (uint64, error)
}

// Shutdowner interface is passed to the listener to shutdown the node if we hit
// error while listening for blockchain events.
type Shutdowner interface {
	Shutdown(context.Context) error
}

type listener struct {
	logger    logging.Logger
	ev        BlockHeightContractFilterer
	blockTime uint64

	postageStampAddress common.Address
	quit                chan struct{}
	wg                  sync.WaitGroup
	metrics             metrics
	shutdowner          Shutdowner
}

func New(
	logger logging.Logger,
	ev BlockHeightContractFilterer,
	postageStampAddress common.Address,
	blockTime uint64,
	shutdowner Shutdowner,
) postage.Listener {
	return &listener{
		logger:              logger,
		ev:                  ev,
		blockTime:           blockTime,
		postageStampAddress: postageStampAddress,
		quit:                make(chan struct{}),
		metrics:             newMetrics(),
		shutdowner:          shutdowner,
	}
}

func (l *listener) filterQuery(from, to *big.Int) ethereum.FilterQuery {
	return ethereum.FilterQuery{
		FromBlock: from,
		ToBlock:   to,
		Addresses: []common.Address{
			l.postageStampAddress,
		},
		Topics: [][]common.Hash{
			{
				batchCreatedTopic,
				batchTopupTopic,
				batchDepthIncreaseTopic,
				priceUpdateTopic,
			},
		},
	}
}

func filterEventsByName(events []xwctypes.RpcEventJson) []xwctypes.RpcEventJson {
	eventsFilter := make([]xwctypes.RpcEventJson, 0)
	for _, e := range events {
		if !(e.EventName != batchCreatedTopicXWC && e.EventName != batchTopupTopicXWC &&
			e.EventName != batchDepthIncreaseTopicXWC && e.EventName != priceUpdateTopicXWC) {
			eventsFilter = append(eventsFilter, e)
		}
	}
	return eventsFilter
}

func ParseEventBatchCreated(event_arg string) (batchCreatedEvent, error) {
	type batchCreatedXwcEvent struct {
		BatchId           string `json:"batchId"`
		TotalAmount       uint64 `json:"totalAmount"`
		NormalisedBalance uint64 `json:"normalisedBalance"`
		Owner             string `json:"_owner"`
		Depth             uint8  `json:"_depth"`
	}

	var xwcEv batchCreatedXwcEvent
	err := json.Unmarshal([]byte(event_arg), &xwcEv)
	if err != nil {
		return batchCreatedEvent{}, err
	}

	var ev batchCreatedEvent
	batchId, err := hex.DecodeString(xwcEv.BatchId)
	if err != nil {
		return batchCreatedEvent{}, err
	}
	copy(ev.BatchId[:], batchId)
	ev.TotalAmount = big.NewInt(0).SetUint64(xwcEv.TotalAmount)
	ev.NormalisedBalance = big.NewInt(0).SetUint64(xwcEv.NormalisedBalance)

	ownerHex, err := xwcfmt.XwcAddrToHexAddr(xwcEv.Owner)
	if err != nil {
		return batchCreatedEvent{}, err
	}
	owner, err := hex.DecodeString(ownerHex)
	if err != nil {
		return batchCreatedEvent{}, err
	}
	ev.Owner.SetBytes(owner[:])
	ev.Depth = xwcEv.Depth

	return ev, nil
}

func ParseEventBatchTopup(event_arg string) (batchTopUpEvent, error) {
	type batchTopUpXwcEvent struct {
		BatchId           string `json:"_batchId"`
		TopupAmount       uint64 `json:"totalAmount"`
		NormalisedBalance uint64 `json:"normalisedBalance"`
	}

	var xwcEv batchTopUpXwcEvent
	err := json.Unmarshal([]byte(event_arg), &xwcEv)
	if err != nil {
		return batchTopUpEvent{}, err
	}

	var ev batchTopUpEvent
	batchId, err := hex.DecodeString(xwcEv.BatchId)
	if err != nil {
		return batchTopUpEvent{}, err
	}
	copy(ev.BatchId[:], batchId)
	ev.TopupAmount = big.NewInt(0).SetUint64(xwcEv.TopupAmount)
	ev.NormalisedBalance = big.NewInt(0).SetUint64(xwcEv.NormalisedBalance)

	return ev, nil
}

func ParseEventBatchDepthIncrease(event_arg string) (batchDepthIncreaseEvent, error) {
	type batchDepthIncreaseXwcEvent struct {
		BatchId           string `json:"batchId"`
		NewDepth          uint64 `json:"newDepth"`
		NormalisedBalance uint64 `json:"batch_normalisedBalance"`
	}

	var xwcEv batchDepthIncreaseXwcEvent
	err := json.Unmarshal([]byte(event_arg), &xwcEv)
	if err != nil {
		return batchDepthIncreaseEvent{}, err
	}

	var ev batchDepthIncreaseEvent
	batchId, err := hex.DecodeString(xwcEv.BatchId)
	if err != nil {
		return batchDepthIncreaseEvent{}, err
	}
	copy(ev.BatchId[:], batchId)
	ev.NewDepth = uint8(xwcEv.NewDepth)
	ev.NormalisedBalance = big.NewInt(0).SetUint64(xwcEv.NormalisedBalance)

	return ev, nil
}

func ParseEventPriceUpdate(event_arg string) (priceUpdateEvent, error) {
	type priceUpdateXwcEvent struct {
		Price uint64 `json:"price"`
	}

	var xwcEv priceUpdateXwcEvent
	err := json.Unmarshal([]byte(event_arg), &xwcEv)
	if err != nil {
		return priceUpdateEvent{}, err
	}

	var ev priceUpdateEvent
	ev.Price = big.NewInt(0).SetUint64(xwcEv.Price)

	return ev, nil
}

func (l *listener) processEvent(e xwctypes.RpcEventJson, updater postage.EventUpdater) error {
	defer l.metrics.EventsProcessed.Inc()
	switch e.EventName {
	case batchCreatedTopicXWC:
		c, err := ParseEventBatchCreated(e.EventArg)
		if err != nil {
			return err
		}
		l.metrics.CreatedCounter.Inc()
		return updater.Create(
			c.BatchId[:],
			c.Owner.Bytes(),
			c.NormalisedBalance,
			c.Depth,
		)
	case batchTopupTopicXWC:
		c, err := ParseEventBatchTopup(e.EventArg)
		if err != nil {
			return err
		}
		l.metrics.TopupCounter.Inc()
		return updater.TopUp(
			c.BatchId[:],
			c.NormalisedBalance,
		)
	case batchDepthIncreaseTopicXWC:
		c, err := ParseEventBatchDepthIncrease(e.EventArg)
		if err != nil {
			return err
		}
		l.metrics.DepthCounter.Inc()
		return updater.UpdateDepth(
			c.BatchId[:],
			c.NewDepth,
			c.NormalisedBalance,
		)
	case priceUpdateTopicXWC:
		c, err := ParseEventPriceUpdate(e.EventArg)
		if err != nil {
			return err
		}
		l.metrics.PriceCounter.Inc()
		return updater.UpdatePrice(
			c.Price,
		)
	default:
		l.metrics.EventErrors.Inc()
		return errors.New("unknown event")
	}
}

func (l *listener) Listen(from uint64, updater postage.EventUpdater) <-chan struct{} {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-l.quit
		cancel()
	}()

	chainUpdateInterval := (time.Duration(l.blockTime) * time.Second) / 2

	synced := make(chan struct{})
	closeOnce := new(sync.Once)
	paged := make(chan struct{}, 1)
	paged <- struct{}{}

	l.wg.Add(1)
	listenf := func() error {
		defer l.wg.Done()

		evRpcRetryTimes := 0
		fastRetryTimesLimit := 60
		retryTimesMaxAllow := 60000

		for {
			select {
			case <-paged:
				// if we paged then it means there's more things to sync on
			case <-time.After(chainUpdateInterval):
			case <-l.quit:
				return nil
			}
			start := time.Now()

			l.metrics.BackendCalls.Inc()
			to, err := l.ev.BlockNumber(ctx)
			if err != nil {
				l.metrics.BackendErrors.Inc()
				evRpcRetryTimes += 1

				if evRpcRetryTimes > retryTimesMaxAllow {
					return err
				} else {
					l.logger.Warningf("l.ev.BlockNumber, retry [%d/%d]", evRpcRetryTimes, retryTimesMaxAllow)

					if evRpcRetryTimes < fastRetryTimesLimit {
						time.Sleep(1 * time.Second)
					} else {
						time.Sleep(10 * time.Second)
					}

					continue
				}
			}

			if to < tailSize {
				// in a test blockchain there might be not be enough blocks yet
				continue
			}

			// consider to-tailSize as the "latest" block we need to sync to
			to = to - tailSize

			if to < from {
				// if the blockNumber is actually less than what we already, it might mean the backend is not synced or some reorg scenario
				continue
			}

			// do some paging (sub-optimal)
			if to-from > blockPage {
				paged <- struct{}{}
				to = from + blockPage
			} else {
				closeOnce.Do(func() { close(synced) })
			}
			l.metrics.BackendCalls.Inc()

			postageAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(l.postageStampAddress[:]))
			l.logger.Infof("GetContractEventsInRange: %s, from %d to %d", postageAddr, from, to)

			events, err := l.ev.GetContractEventsInRange(ctx, l.postageStampAddress, from, to)
			if err != nil {
				l.metrics.BackendErrors.Inc()
				evRpcRetryTimes += 1

				if evRpcRetryTimes > retryTimesMaxAllow {
					return err
				} else {
					l.logger.Warningf("l.ev.GetContractEventsInRange, retry [%d/%d]", evRpcRetryTimes, retryTimesMaxAllow)

					if evRpcRetryTimes < fastRetryTimesLimit {
						time.Sleep(1 * time.Second)
					} else {
						time.Sleep(10 * time.Second)
					}

					continue
				}
			}

			// recover evRpcRetryTimes
			evRpcRetryTimes = 0

			// filter by event name
			events = filterEventsByName(events)

			l.logger.Infof("events count %d", len(events))

			if err := updater.TransactionStart(); err != nil {
				return err
			}

			for _, e := range events {
				startEv := time.Now()
				err = updater.UpdateBlockNumber(e.BlockNum)
				if err != nil {
					return err
				}

				l.logger.Infof("processEvent %s: %s", e.EventName, e.EventArg)

				if err = l.processEvent(e, updater); err != nil {
					return err
				}
				totalTimeMetric(l.metrics.EventProcessDuration, startEv)
			}

			err = updater.UpdateBlockNumber(to)
			if err != nil {
				return err
			}

			if err := updater.TransactionEnd(); err != nil {
				return err
			}

			from = to + 1
			totalTimeMetric(l.metrics.PageProcessDuration, start)
			l.metrics.PagesProcessed.Inc()
		}
	}

	go func() {
		err := listenf()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// context cancelled is returned on shutdown,
				// therefore we do nothing here
				return
			}
			l.logger.Errorf("failed syncing event listener, shutting down node err: %v", err)
			if l.shutdowner != nil {
				err = l.shutdowner.Shutdown(context.Background())
				if err != nil {
					l.logger.Errorf("failed shutting down node: %v", err)
				}
			}
		}
	}()

	return synced
}

func (l *listener) Close() error {
	close(l.quit)
	done := make(chan struct{})

	go func() {
		defer close(done)
		l.wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		return errors.New("postage listener closed with running goroutines")
	}
	return nil
}

func parseABI(json string) abi.ABI {
	cabi, err := abi.JSON(strings.NewReader(json))
	if err != nil {
		panic(fmt.Sprintf("error creating ABI for postage contract: %v", err))
	}
	return cabi
}

type batchCreatedEvent struct {
	BatchId           [32]byte
	TotalAmount       *big.Int
	NormalisedBalance *big.Int
	Owner             common.Address
	Depth             uint8
}

type batchTopUpEvent struct {
	BatchId           [32]byte
	TopupAmount       *big.Int
	NormalisedBalance *big.Int
}

type batchDepthIncreaseEvent struct {
	BatchId           [32]byte
	NewDepth          uint8
	NormalisedBalance *big.Int
}

type priceUpdateEvent struct {
	Price *big.Int
}

var (
	//GoerliPostageStampContractAddress = common.HexToAddress("0xB3B7f2eD97B735893316aEeA849235de5e8972a2")
	//GoerliStartBlock                  = uint64(4818979)

	GoerliStartBlock = uint64(0)
)

// DiscoverAddresses returns the canonical contracts for this chainID
func DiscoverAddresses(chainID int64) (postageStamp common.Address, startBlock uint64, found bool) {
	hexAddr, _ := xwcfmt.XwcConAddrToHexAddr(property.PostageStampAddress)
	bytesAddr, _ := hex.DecodeString(hexAddr)

	var addr common.Address
	addr.SetBytes(bytesAddr)

	return addr, GoerliStartBlock, true
}

func totalTimeMetric(metric prometheus.Counter, start time.Time) {
	totalTime := time.Since(start)
	metric.Add(float64(totalTime))
}
