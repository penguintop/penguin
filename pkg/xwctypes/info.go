package xwctypes

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/xwcfmt"
)

type RpcInfoJson struct {
	HeadBlockNum       uint64 `json:"head_block_num"`
	HeadBlockId        string `json:"head_block_id"`
	ChainId            string `json:"chain_id"`
	Participation      string `json:"participation"`
	RoundParticipation string `json:"round_participation"`
}

type RpcInfo struct {
	HeadBlockNum       uint64
	HeadBlockId        xwcfmt.Hash
	ChainId            common.Hash
	Participation      float64
	RoundParticipation float64
}
