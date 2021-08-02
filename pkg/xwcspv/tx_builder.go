package xwcspv

import (
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"time"
)

const (
	// tx expire seconds
	EXPIRE_SECONDS = 600
)

const (
	TxOpTypeTransfer           = 0
	TxOpTypeTransferToContract = 81
	TxOpTypeInvokeContract     = 79
)

func XwcBuildTxTransfer(refBlockNum uint16, refBlockPrefix uint32,
	fromAddr string, toAddr string, amount uint64, fee uint64, memo string) ([]byte, *xwcfmt.Transaction, error) {

	var tx xwcfmt.Transaction
	tx.RefBlockNum = refBlockNum
	tx.RefBlockPrefix = refBlockPrefix
	//expire 10 min
	tx.Expiration = xwcfmt.UTCTime(time.Now().Unix() + EXPIRE_SECONDS)
	tx.Extensions = make([]interface{}, 0)
	tx.Signatures = make([]xwcfmt.Signature, 0)

	var op xwcfmt.TransferOperation
	err := op.SetValue(fromAddr, toAddr, amount, fee, memo)
	if err != nil {
		return nil, nil, err
	}
	var opPair xwcfmt.OperationPair
	opPair[0] = byte(TxOpTypeTransfer)
	opPair[1] = &op
	tx.Operations = append(tx.Operations, opPair)

	return tx.Pack(), &tx, nil
}

func XwcBuildTxTransferToContract(refBlockNum uint16, refBlockPrefix uint32, callerAddr string,
	callerPubKey string, conAddr string, fee uint64, gasPrice uint64,
	gasLimit uint64, amount uint64, param string) ([]byte, *xwcfmt.Transaction, error) {

	var tx xwcfmt.Transaction
	tx.RefBlockNum = refBlockNum
	tx.RefBlockPrefix = refBlockPrefix
	//expire 10 min
	tx.Expiration = xwcfmt.UTCTime(time.Now().Unix() + EXPIRE_SECONDS)
	tx.Extensions = make([]interface{}, 0)
	tx.Signatures = make([]xwcfmt.Signature, 0)

	var op xwcfmt.ContractTransferOperation
	err := op.SetValue(callerAddr, callerPubKey, conAddr, fee, gasPrice, gasLimit, amount, param)
	if err != nil {
		return nil, nil, err
	}
	var opPair xwcfmt.OperationPair
	opPair[0] = byte(TxOpTypeTransferToContract)
	opPair[1] = &op
	tx.Operations = append(tx.Operations, opPair)

	return tx.Pack(), &tx, nil
}

func XwcBuildTxInvokeContract(refBlockNum uint16, refBlockPrefix uint32, callerAddr string,
	callerPubKey string, conAddr string, fee uint64, gasPrice uint64,
	gasLimit uint64, conApi string, conArg string) ([]byte, *xwcfmt.Transaction, error) {

	var tx xwcfmt.Transaction
	tx.RefBlockNum = refBlockNum
	tx.RefBlockPrefix = refBlockPrefix
	//expire 10 min
	tx.Expiration = xwcfmt.UTCTime(time.Now().Unix() + EXPIRE_SECONDS)
	tx.Extensions = make([]interface{}, 0)
	tx.Signatures = make([]xwcfmt.Signature, 0)

	var op xwcfmt.ContractInvokeOperation
	err := op.SetValue(callerAddr, callerPubKey, conAddr, fee, gasPrice, gasLimit, conApi, conArg)
	if err != nil {
		return nil, nil, err
	}
	var opPair xwcfmt.OperationPair
	opPair[0] = byte(TxOpTypeInvokeContract)
	opPair[1] = &op
	tx.Operations = append(tx.Operations, opPair)

	return tx.Pack(), &tx, nil
}
