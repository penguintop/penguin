package xwctypes

import (
	"github.com/penguintop/penguin/pkg/xwcfmt"
)

type RpcBlockJson struct {
	Previous              string        `json:"previous"`
	Timestamp             string        `json:"timestamp"`
	Trxfee                uint64        `json:"trxfee"`
	Miner                 string        `json:"miner"`
	TransactionMerkleRoot string        `json:"transaction_merkle_root"`
	Extensions            []interface{} `json:"extensions"`
	NextSecretHash        string        `json:"next_secret_hash"`
	PreviousSecret        string        `json:"previous_secret"`
	MinerSignature        string        `json:"miner_signature"`
	Transactions          []interface{} `json:"transactions"`
	Number                uint64        `json:"number"`
	BlockId               string        `json:"block_id"`
	SigningKey            string        `json:"signing_key"`
	Reward                uint64        `json:"reward"`
	TransactionIds        []string      `json:"transaction_ids"`
}

type RpcBlock struct {
	Previous              xwcfmt.Hash
	Timestamp             uint64
	Trxfee                uint64
	Miner                 string
	TransactionMerkleRoot xwcfmt.Hash
	Extensions            []interface{}
	NextSecretHash        xwcfmt.Hash
	PreviousSecret        xwcfmt.Hash
	MinerSignature        []byte
	Transactions          []interface{}
	Number                uint64
	BlockId               xwcfmt.Hash
	SigningKey            string
	Reward                uint64
	TransactionIds        []xwcfmt.Hash
}

type RpcHeader struct {
}
