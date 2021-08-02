package xwctypes

type RpcBalanceJson struct {
	Amount  interface{} `json:"amount"`
	AssetId string      `json:"asset_id"`
}

type RpcBalance struct {
	Amount  uint64
	AssetId string
}
