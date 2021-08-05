package transaction

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/penguintop/penguin/pkg/crypto"
    "github.com/penguintop/penguin/pkg/penguin"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"github.com/penguintop/penguin/pkg/xwcspv"
)

type Matcher struct {
	backend Backend
	signer  types.Signer
}

var (
	ErrTransactionNotFound      = errors.New("transaction not found")
	ErrTransactionPending       = errors.New("transaction in pending status")
	ErrTransactionSenderInvalid = errors.New("invalid transaction sender")
)

func NewMatcher(backend Backend, signer types.Signer) *Matcher {
	return &Matcher{
		backend: backend,
		signer:  signer,
	}
}

func (m Matcher) Matches(ctx context.Context, tx []byte, networkID uint64, senderOverlay penguin.Address) (bool, error) {
	incomingTx := common.BytesToHash(tx)

	nTx, isPending, err := m.backend.TransactionByHash(ctx, incomingTx)
	if err != nil {
		return false, fmt.Errorf("%v: %w", err, ErrTransactionNotFound)
	}

	if isPending {
		return false, ErrTransactionPending
	}

	fromAddr := ""
	op := nTx.Operations[0].([]interface{})
	opType := uint64(op[0].(float64))
	if opType == xwcspv.TxOpTypeTransfer {
		v, ok := op[1].(map[string]interface{})["from_addr"]
		if !ok {
			return false, errors.New("invalid tx: TxOpTypeTransfer")
		}
		fromAddr = v.(string)
	} else if opType == xwcspv.TxOpTypeTransferToContract {
		v, ok := op[1].(map[string]interface{})["caller_addr"]
		if !ok {
			return false, errors.New("invalid tx: TxOpTypeTransferToContract")
		}
		fromAddr = v.(string)
	} else if opType == xwcspv.TxOpTypeInvokeContract {
		v, ok := op[1].(map[string]interface{})["caller_addr"]
		if !ok {
			return false, errors.New("invalid tx: TxOpTypeInvokeContract")
		}
		fromAddr = v.(string)
	} else {
		// other tx type
		return true, nil
	}

	hexAddr, err := xwcfmt.XwcAddrToHexAddr(fromAddr)
	if err != nil {
		return false, err
	}
	bytesAddr, _ := hex.DecodeString(hexAddr)

	expectedRemotePenAddress := crypto.NewOverlayFromXwcAddress(bytesAddr, networkID)
	return expectedRemotePenAddress.Equal(senderOverlay), nil
}
