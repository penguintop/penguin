package xwctypes

type RpcCodePrintable struct {
	Abi                        []string    `json:"abi"`
	OfflineAbi                 []string    `json:"offline_abi"`
	Events                     []string    `json:"events"`
	PrintableStorageProperties interface{} `json:"printable_storage_properties"`
	PrintableCode              string      `json:"printable_code"`
	CodeHash                   string      `json:"code_hash"`
}

type RpcContractJson struct {
	Id                string           `json:"id"`
	OwnerAddress      string           `json:"owner_address"`
	OwnerName         string           `json:"owner_name"`
	Name              string           `json:"name"`
	Description       string           `json:"description"`
	TypeOfContract    string           `json:"type_of_contract"`
	RegisteredBlock   uint64           `json:"registered_block"`
	RegisteredTrx     string           `json:"registered_trx"`
	NativeContractKey string           `json:"native_contract_key"`
	Derived           interface{}      `json:"derived"`
	CodePrintable     RpcCodePrintable `json:"code_printable"`
	CreateTime        string           `json:"createtime"`
}
