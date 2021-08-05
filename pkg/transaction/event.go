// Copyright 2021 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transaction

import (
	"errors"
	"github.com/penguintop/penguin/pkg/xwctypes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrNoTopic       = errors.New("no topic")
)

// ParseEvent will parse the specified abi event from the given log
func ParseEvent(a *abi.ABI, eventName string, c interface{}, e types.Log) error {
	if len(e.Topics) == 0 {
		return ErrNoTopic
	}
	if len(e.Data) > 0 {
		if err := a.UnpackIntoInterface(c, eventName, e.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range a.Events[eventName].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(c, indexed, e.Topics[1:])
}

// FindSingleEvent will find the first event of the given kind.
func FindSingleEvent(abi *abi.ABI, receipt *types.Receipt, contractAddress common.Address, event abi.Event, out interface{}) error {
	if receipt.Status != 1 {
		return ErrTransactionReverted
	}
	for _, log := range receipt.Logs {
		if log.Address != contractAddress {
			continue
		}
		if len(log.Topics) == 0 {
			continue
		}
		if log.Topics[0] != event.ID {
			continue
		}

		return ParseEvent(abi, event.Name, out, *log)
	}
	return ErrEventNotFound
}

func FindSingleEventXwc(receipt *xwctypes.RpcTransactionReceipt, contractAddress common.Address, eventName string) (string, error) {
	if receipt.ExecSucceed != true {
		return "", ErrTransactionReverted
	}
	for _, ev := range receipt.Events {
		if ev.ContractAddress != contractAddress {
			continue
		}
		if ev.EventName == eventName {
			return ev.EventArg, nil
		}
	}
	return "", ErrEventNotFound
}
