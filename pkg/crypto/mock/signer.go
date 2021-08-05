// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"crypto/ecdsa"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/crypto/eip712"
)

type signerMock struct {
	signTx          func(transaction *types.Transaction, chainID *big.Int) (*types.Transaction, error)
	signTypedData   func(*eip712.TypedData) ([]byte, error)
	ethereumAddress func() (common.Address, error)
	signFunc        func([]byte) ([]byte, error)
}

func (m *signerMock) SignForAudit(data []byte) ([]byte, error) {
	return nil, nil
}

func (m *signerMock) SignXwcData(data []byte) ([]byte, error) {
	return nil, nil
}

func (m *signerMock) CompressedPubKeyHex() (string, error) {
	return "", nil
}

func (m *signerMock) SignXwcTx(transaction *xwcfmt.Transaction, chainID string) (*xwcfmt.Transaction, error) {
	return nil, nil
}

func (m *signerMock) EthereumAddress() (common.Address, error) {
	if m.ethereumAddress != nil {
		return m.ethereumAddress()
	}
	return common.Address{}, nil
}

func (m *signerMock) XwcAddress() (common.Address, error) {
	if m.ethereumAddress != nil {
		return m.ethereumAddress()
	}
	return common.Address{}, nil
}

func (m *signerMock) Sign(data []byte) ([]byte, error) {
	return m.signFunc(data)
}

func (m *signerMock) SignTx(transaction *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return m.signTx(transaction, chainID)
}

func (*signerMock) PublicKey() (*ecdsa.PublicKey, error) {
	return nil, nil
}

func (m *signerMock) SignTypedData(d *eip712.TypedData) ([]byte, error) {
	return m.signTypedData(d)
}

func New(opts ...Option) crypto.Signer {
	mock := new(signerMock)
	for _, o := range opts {
		o.apply(mock)
	}
	return mock
}

// Option is the option passed to the mock Chequebook service
type Option interface {
	apply(*signerMock)
}

type optionFunc func(*signerMock)

func (f optionFunc) apply(r *signerMock) { f(r) }

func WithSignFunc(f func(data []byte) ([]byte, error)) Option {
	return optionFunc(func(s *signerMock) {
		s.signFunc = f
	})
}

func WithSignTxFunc(f func(transaction *types.Transaction, chainID *big.Int) (*types.Transaction, error)) Option {
	return optionFunc(func(s *signerMock) {
		s.signTx = f
	})
}

func WithSignTypedDataFunc(f func(*eip712.TypedData) ([]byte, error)) Option {
	return optionFunc(func(s *signerMock) {
		s.signTypedData = f
	})
}

func WithEthereumAddressFunc(f func() (common.Address, error)) Option {
	return optionFunc(func(s *signerMock) {
		s.ethereumAddress = f
	})
}
