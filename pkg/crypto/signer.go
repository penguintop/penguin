// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/bitnexty/secp256k1-go"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/penguintop/penguin/pkg/crypto/eip712"
)

var (
	ErrInvalidLength = errors.New("invalid signature length")
)

type Signer interface {
	// Sign signs data with ethereum prefix (eip191 type 0x45).
	Sign(data []byte) ([]byte, error)
	// SignTx signs an ethereum transaction.
	SignTx(transaction *types.Transaction, chainID *big.Int) (*types.Transaction, error)
	// SignTypedData signs data according to eip712.
	SignTypedData(typedData *eip712.TypedData) ([]byte, error)
	// PublicKey returns the public key this signer uses.
	PublicKey() (*ecdsa.PublicKey, error)
	// EthereumAddress returns the ethereum address this signer uses.
	EthereumAddress() (common.Address, error)

	XwcAddress() (common.Address, error)
	CompressedPubKeyHex() (string, error)
	SignXwcTx(transaction *xwcfmt.Transaction, chainID string) (*xwcfmt.Transaction, error)
	SignXwcData(data []byte) ([]byte, error)
	SignForAudit(data []byte) ([]byte, error)
}

// addEthereumPrefix adds the ethereum prefix to the data.
func addEthereumPrefix(data []byte) []byte {
	return []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data))
}

// hashWithEthereumPrefix returns the hash that should be signed for the given data.
func hashWithEthereumPrefix(data []byte) ([]byte, error) {
	return LegacyKeccak256(addEthereumPrefix(data))
}

// Recover verifies signature with the data base provided.
// It is using `btcec.RecoverCompact` function.
func Recover(signature, data []byte) (*ecdsa.PublicKey, error) {
	if len(signature) != 65 {
		return nil, ErrInvalidLength
	}
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, 65)
	btcsig[0] = signature[64]
	copy(btcsig[1:], signature)

	hash, err := hashWithEthereumPrefix(data)
	if err != nil {
		return nil, err
	}

	p, _, err := btcec.RecoverCompact(btcec.S256(), btcsig, hash)
	return (*ecdsa.PublicKey)(p), err
}

type defaultSigner struct {
	key *ecdsa.PrivateKey
}

func (d *defaultSigner) SignXwcTx(tx *xwcfmt.Transaction, chainID string) (*xwcfmt.Transaction, error) {
	// sign data
	chainIdBytes, err := hex.DecodeString(property.CHAIN_ID)
	if err != nil {
		return nil, err
	}

	txData := tx.Pack()

	s256 := sha256.New()
	_, _ = s256.Write(chainIdBytes)
	_, _ = s256.Write(txData)
	digestData := s256.Sum(nil)

	txSig := make([]byte, 0)
	for {
		txSig, err = secp256k1.BtsSign(digestData, d.key.D.Bytes(), true)
		if err != nil {
			return nil, err
		}
		if txSig[32] < 0x80 {
			break
		}
	}

	// tx data with sig
	txBytesWithSig := make([]byte, 0)
	txBytesWithSig = append(txBytesWithSig, txData...)

	// sig count
	txBytesWithSig = append(txBytesWithSig, xwcfmt.PackVarInt(1)...)

	txBytesWithSig = append(txBytesWithSig, xwcfmt.PackVarInt(uint64(len(txSig)))...)
	txBytesWithSig = append(txBytesWithSig, txSig...)

	tx.Signatures = append(tx.Signatures, txSig)

	return tx, nil
}

func (d *defaultSigner) SignXwcData(data []byte) ([]byte, error) {
	s256 := sha256.New()
	_, _ = s256.Write(data)
	digestData := s256.Sum(nil)

	sig := make([]byte, 0)
	var err error
	for {
		sig, err = secp256k1.BtsSign(digestData, d.key.D.Bytes(), true)
		if err != nil {
			return nil, err
		}
		if sig[32] < 0x80 {
			break
		}
	}

	return sig, nil
}

func (d *defaultSigner) SignForAudit(data []byte) ([]byte, error) {
	s256 := sha256.New()
	_, _ = s256.Write(data)
	digestData := s256.Sum(nil)

	sig := make([]byte, 0)
	var err error
	for {
		sig, err = secp256k1.Sign(digestData, d.key.D.Bytes())
		if err != nil {
			return nil, err
		}
		if sig[32] < 0x80 {
			break
		}
	}

	return sig, nil
}

func (d *defaultSigner) CompressedPubKeyHex() (string, error) {
	p, err := d.PublicKey()
	if err != nil {
		return "", err
	}
	pubBytes := elliptic.Marshal(btcec.S256(), p.X, p.Y)

	// compress pubkey
	pubCpsBytes := make([]byte, 33)
	if pubBytes[64]%2 == 0 {
		pubCpsBytes[0] = 0x2
	} else {
		pubCpsBytes[0] = 0x3
	}
	copy(pubCpsBytes[1:], pubBytes[1:33])

	return hex.EncodeToString(pubCpsBytes), nil
}

func NewDefaultSigner(key *ecdsa.PrivateKey) Signer {
	return &defaultSigner{
		key: key,
	}
}

// PublicKey returns the public key this signer uses.
func (d *defaultSigner) PublicKey() (*ecdsa.PublicKey, error) {
	return &d.key.PublicKey, nil
}

// Sign signs data with ethereum prefix (eip191 type 0x45).
func (d *defaultSigner) Sign(data []byte) (signature []byte, err error) {
	hash, err := hashWithEthereumPrefix(data)
	if err != nil {
		return nil, err
	}

	return d.sign(hash, true)
}

// SignTx signs an ethereum transaction.
func (d *defaultSigner) SignTx(transaction *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	txSigner := types.NewEIP155Signer(chainID)
	hash := txSigner.Hash(transaction).Bytes()
	// isCompressedKey is false here so we get the expected v value (27 or 28)
	signature, err := d.sign(hash, false)
	if err != nil {
		return nil, err
	}

	// v value needs to be adjusted by 27 as transaction.WithSignature expects it to be 0 or 1
	signature[64] -= 27
	return transaction.WithSignature(txSigner, signature)
}

// EthereumAddress returns the ethereum address this signer uses.
func (d *defaultSigner) EthereumAddress() (common.Address, error) {
	publicKey, err := d.PublicKey()
	if err != nil {
		return common.Address{}, err
	}
	eth, err := NewEthereumAddress(*publicKey)
	if err != nil {
		return common.Address{}, err
	}
	var ethAddress common.Address
	copy(ethAddress[:], eth)
	return ethAddress, nil
}

func (d *defaultSigner) XwcAddress() (common.Address, error) {
	publicKey, err := d.PublicKey()
	if err != nil {
		return common.Address{}, err
	}
	xwc, err := NewXwcAddress(*publicKey)
	if err != nil {
		return common.Address{}, err
	}
	var xwcAddress common.Address
	copy(xwcAddress[:], xwc)
	return xwcAddress, nil
}

// SignTypedData signs data according to eip712.
func (d *defaultSigner) SignTypedData(typedData *eip712.TypedData) ([]byte, error) {
	rawData, err := eip712.EncodeForSigning(typedData)
	if err != nil {
		return nil, err
	}

	sighash, err := LegacyKeccak256(rawData)
	if err != nil {
		return nil, err
	}

	return d.sign(sighash, false)
}

// Sign the provided hash and convert it to the ethereum (r,s,v) format.
func (d *defaultSigner) sign(sighash []byte, isCompressedKey bool) ([]byte, error) {
	signature, err := btcec.SignCompact(btcec.S256(), (*btcec.PrivateKey)(d.key), sighash, false)
	if err != nil {
		return nil, err
	}

	// Convert to Ethereum signature format with 'recovery id' v at the end.
	v := signature[0]
	copy(signature, signature[1:])
	signature[64] = v
	return signature, nil
}

// RecoverEIP712 recovers the public key for eip712 signed data.
func RecoverEIP712(signature []byte, data *eip712.TypedData) (*ecdsa.PublicKey, error) {
	if len(signature) != 65 {
		return nil, errors.New("invalid length")
	}
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, 65)
	btcsig[0] = signature[64]
	copy(btcsig[1:], signature)

	rawData, err := eip712.EncodeForSigning(data)
	if err != nil {
		return nil, err
	}

	sighash, err := LegacyKeccak256(rawData)
	if err != nil {
		return nil, err
	}

	p, _, err := btcec.RecoverCompact(btcec.S256(), btcsig, sighash)
	return (*ecdsa.PublicKey)(p), err
}
