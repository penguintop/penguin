package xwctypes

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/xwcfmt"
)

type RpcTransactionJson struct {
	RefBlockNum    uint64        `json:"ref_block_num"`
	RefBlockPrefix uint64        `json:"ref_block_prefix"`
	Expiration     string        `json:"expiration"`
	Operations     []interface{} `json:"operations"`
	Extensions     []interface{} `json:"extensions"`
	Signatures     []string      `json:"signatures"`
	BlockNum       uint64        `json:"block_num"`
	TrxId          string        `json:"trxid"`
}

type RpcTransaction struct {
	RefBlockNum    uint64
	RefBlockPrefix uint64
	Expiration     uint64
	Operations     []interface{}
	Extensions     []interface{}
	Signatures     [][]byte
	BlockNum       uint64
	TrxId          xwcfmt.Hash
}

type RpcEventJson struct {
	ContractAddress string `json:"contract_address"`
	CallerAddr      string `json:"caller_addr"`
	EventName       string `json:"event_name"`
	EventArg        string `json:"event_arg"`
	BlockNum        uint64 `json:"block_num"`
	OpNum           uint64 `json:"op_num"`
}

type RpcTransactionReceiptJson struct {
	TrxId       string         `json:"trx_id"`
	BlockNum    uint64         `json:"block_num"`
	Events      []RpcEventJson `json:"events"`
	ExecSucceed bool           `json:"exec_succeed"`
	AcctualFee  uint64         `json:"acctual_fee"`
	Invoker     string         `json:"invoker"`
}

type RpcEvent struct {
	ContractAddress common.Address
	CallerAddr      common.Address
	EventName       string
	EventArg        string
	BlockNum        uint64
	OpNum           uint64
}

type RpcTransactionReceipt struct {
	TrxId       common.Hash
	BlockNum    uint64
	Events      []RpcEvent
	ExecSucceed bool
	AcctualFee  uint64
	Invoker     common.Address
}
