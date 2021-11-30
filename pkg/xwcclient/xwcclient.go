package xwcclient

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"github.com/penguintop/penguin/pkg/xwctypes"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/penguintop/penguin/pkg/rpc"
)

var (
	ErrTransactionReceiptNotFound = errors.New("transaction receipt not found")
)

// Client defines typed wrappers for the Ethereum RPC API.
type Client struct {
	c *rpc.Client
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *Client {
	return &Client{c}
}

func (ec *Client) Close() {
	ec.c.Close()
}

// Blockchain Access

func (ec *Client) IsLocked(ctx context.Context) (bool, error) {
	var result bool
	err := ec.c.CallContext(ctx, &result, "is_locked")
	return result, err
}

func (ec *Client) GetAccount(ctx context.Context, acctName string) (xwctypes.RpcAccountJson, error) {
	var result xwctypes.RpcAccountJson
	err := ec.c.CallContext(ctx, &result, "get_account", acctName)
	return result, err
}

func (ec *Client) CreateAccount(ctx context.Context, acctName string) (string, error) {
	var acctAddr string
	err := ec.c.CallContext(ctx, &acctAddr, "wallet_create_account", acctName)
	return acctAddr, err
}

// ChainId retrieves the current chain ID for transaction replay protection.
func (ec *Client) ChainID(ctx context.Context) (int64, error) {
	var result xwctypes.RpcInfoJson
	err := ec.c.CallContext(ctx, &result, "info")
	if err != nil {
		return 0, err
	}
	chainIDBytes, _ := hex.DecodeString(result.ChainId)
	chainID := int64(binary.LittleEndian.Uint16(chainIDBytes[0:2]))
	return chainID, err
}

// BlockByHash returns the given full block.
//
// Note that loading full blocks requires two requests. Use HeaderByHash
// if you don't need all transactions or uncle headers.
func (ec *Client) BlockByHash(ctx context.Context, hash common.Hash) (*xwctypes.RpcBlock, error) {
	return nil, nil
}

// BlockByNumber returns a block from the current canonical chain. If number is nil, the
// latest known block is returned.
//
// Note that loading full blocks requires two requests. Use HeaderByNumber
// if you don't need all transactions or uncle headers.
func (ec *Client) BlockByNumber(ctx context.Context, number *big.Int) (*xwctypes.RpcBlock, error) {
	var res xwctypes.RpcBlockJson
	err := ec.c.CallContext(ctx, &res, "get_block", number.String())
	if err != nil {
		return nil, err
	}

	var result = xwctypes.RpcBlock{}
	previousBytes, _ := hex.DecodeString(res.Previous)
	copy(result.Previous[:], previousBytes)

	blockTime, _ := property.RFC3339ToUTC(res.Timestamp)
	result.Timestamp = blockTime
	result.Trxfee = res.Trxfee
	result.Miner = res.Miner

	merkleRootBytes, _ := hex.DecodeString(res.TransactionMerkleRoot)
	copy(result.TransactionMerkleRoot[:], merkleRootBytes)

	result.Extensions = res.Extensions

	nextSecretHashBytes, _ := hex.DecodeString(res.NextSecretHash)
	copy(result.NextSecretHash[:], nextSecretHashBytes)

	previousSecretBytes, _ := hex.DecodeString(res.PreviousSecret)
	copy(result.PreviousSecret[:], previousSecretBytes)

	result.MinerSignature, _ = hex.DecodeString(res.MinerSignature)
	result.Transactions = res.Transactions
	result.Number = res.Number

	blockIdBytes, _ := hex.DecodeString(res.BlockId)
	copy(result.BlockId[:], blockIdBytes)

	result.SigningKey = res.SigningKey
	result.Reward = res.Reward

	for _, k := range res.TransactionIds {
		transactionIdBytes, _ := hex.DecodeString(k)
		var transactionId xwcfmt.Hash
		copy(transactionId[:], transactionIdBytes)
		result.TransactionIds = append(result.TransactionIds, transactionId)
	}

	return &result, nil
}

// BlockNumber returns the most recent block number
func (ec *Client) BlockNumber(ctx context.Context) (uint64, error) {
	var result xwctypes.RpcInfoJson
	err := ec.c.CallContext(ctx, &result, "info")
	return result.HeadBlockNum, err
}

func (ec *Client) RefBlockInfo(ctx context.Context) (uint16, uint32, error) {
	var result string
	err := ec.c.CallContext(ctx, &result, "lightwallet_get_refblock_info")
	if err != nil {
		return 0, 0, err
	}
	l := strings.Split(result, ",")
	refBlockNum, _ := strconv.ParseInt(l[0], 10, 64)
	refBlockPrefix, _ := strconv.ParseInt(l[1], 10, 64)
	return uint16(refBlockNum), uint32(refBlockPrefix), nil
}

// HeaderByHash returns the block header with the given hash.
func (ec *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return nil, nil
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (ec *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, nil
}

// TransactionByHash returns the transaction with the given hash.
func (ec *Client) TransactionByHash(ctx context.Context, hash common.Hash) (tx *xwctypes.RpcTransaction, isPending bool, err error) {
	var res xwctypes.RpcTransactionJson
	err = ec.c.CallContext(ctx, &res, "get_transaction", hex.EncodeToString(hash[common.HashLength-xwcfmt.HashLength:common.HashLength]))
	if err != nil {
		return nil, false, err
	}

	var result = xwctypes.RpcTransaction{}
	result.RefBlockNum = res.RefBlockPrefix
	result.RefBlockPrefix = res.RefBlockPrefix

	expiration, _ := property.RFC3339ToUTC(res.Expiration)
	result.Expiration = expiration

	result.Operations = res.Operations
	result.Extensions = res.Extensions

	for _, sigHex := range res.Signatures {
		sigBytes, _ := hex.DecodeString(sigHex)
		result.Signatures = append(result.Signatures, sigBytes)
	}

	result.BlockNum = res.BlockNum

	trxIdBytes, _ := hex.DecodeString(res.TrxId)
	copy(result.TrxId[:], trxIdBytes)

	if result.BlockNum == 0 {
		isPending = true
	} else {
		isPending = false
	}

	return &result, isPending, nil
}

// TransactionSender returns the sender address of the given transaction. The transaction
// must be known to the remote node and included in the blockchain at the given block and
// index. The sender is the one derived by the protocol at the time of inclusion.
//
// There is a fast-path for transactions retrieved by TransactionByHash and
// TransactionInBlock. Getting their sender address can be done without an RPC interaction.
func (ec *Client) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	return common.Address{}, nil
}

// TransactionCount returns the total number of transactions in the given block.
func (ec *Client) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	return 0, nil
}

// TransactionInBlock returns a single transaction at index in the given block.
func (ec *Client) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	return nil, nil
}

// TransactionReceipt returns the receipt of a transaction by transaction hash.
// Note that the receipt is not available for pending transactions.
func (ec *Client) TransactionReceipt(ctx context.Context, txHash common.Hash) (receipt *xwctypes.RpcTransactionReceipt, err error) {
	reslist := make([]xwctypes.RpcTransactionReceiptJson, 0)
	err = ec.c.CallContext(ctx, &reslist, "get_contract_invoke_object", hex.EncodeToString(txHash[common.HashLength-xwcfmt.HashLength:common.HashLength]))
	if err != nil {
		return nil, err
	}

	if len(reslist) == 0 {
		return nil, ErrTransactionReceiptNotFound
	}

	res := reslist[0]
	var result = xwctypes.RpcTransactionReceipt{}

	result.TrxId = txHash
	result.BlockNum = res.BlockNum

	for _, ev := range res.Events {
		var event xwctypes.RpcEvent

		event.BlockNum = ev.BlockNum
		event.OpNum = ev.OpNum
		event.EventName = ev.EventName
		event.EventArg = ev.EventArg

		contractAddrHex, _ := xwcfmt.XwcConAddrToHexAddr(ev.ContractAddress)
		contractAddrBytes, _ := hex.DecodeString(contractAddrHex)
		event.ContractAddress.SetBytes(contractAddrBytes[:])

		callerAddrHex, _ := xwcfmt.XwcAddrToHexAddr(ev.CallerAddr)
		callerAddrBytes, _ := hex.DecodeString(callerAddrHex)
		event.CallerAddr.SetBytes(callerAddrBytes[:])

		result.Events = append(result.Events, event)
	}

	result.ExecSucceed = res.ExecSucceed
	result.AcctualFee = res.AcctualFee

	invokerAddrHex, _ := xwcfmt.XwcAddrToHexAddr(res.Invoker)
	invokerAddrBytes, _ := hex.DecodeString(invokerAddrHex)
	result.Invoker.SetBytes(invokerAddrBytes[:])

	receipt = &result

	return receipt, nil
}

func (ec *Client) GetContractEventsInRange(ctx context.Context, account common.Address, start uint64, to uint64) ([]xwctypes.RpcEventJson, error) {
	conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(account[:]))

	res := make([]xwctypes.RpcEventJson, 0)
	count := to - start

	err := ec.c.CallContext(ctx, &res, "get_contract_events_in_range", conAddr, start, count)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

// SyncProgress retrieves the current progress of the sync algorithm. If there's
// no sync currently running, it returns nil.
func (ec *Client) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	return nil, nil
}

// SubscribeNewHead subscribes to notifications about the current blockchain head
// on the given channel.
func (ec *Client) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return ec.c.EthSubscribe(ctx, ch, "newHeads")
}

// State Access

// NetworkID returns the network ID (also known as the chain ID) for this chain.
func (ec *Client) NetworkID(ctx context.Context) (*big.Int, error) {
	return nil, nil
}

// BalanceAt returns the wei balance of the given account.
// The block number can be nil, in which case the balance is taken from the latest known block.
func (ec *Client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var res []xwctypes.RpcBalanceJson
	xwcAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(account[:]))

	err := ec.c.CallContext(ctx, &res, "get_addr_balances", xwcAddr)
	if err != nil {
		return nil, err
	}

	for _, k := range res {
		if k.AssetId == property.XWC_ASSET_ID {
			typeStr := reflect.TypeOf(k.Amount).String()
			if typeStr == "string" {
				balance, err := strconv.ParseUint(k.Amount.(string), 10, 64)
				return big.NewInt(int64(balance)), err
			} else {
				return big.NewInt(int64(k.Amount.(float64))), nil
			}
		}
	}
	return big.NewInt(0), nil
}

// StorageAt returns the value of key in the contract storage of the given account.
// The block number can be nil, in which case the value is taken from the latest known block.
func (ec *Client) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	return nil, nil
}

// CodeAt returns the contract code of the given account.
// The block number can be nil, in which case the code is taken from the latest known block.
func (ec *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	var res xwctypes.RpcContractJson
	conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(account[:]))

	err := ec.c.CallContext(ctx, &res, "get_contract_info", conAddr)
	if err != nil {
		return nil, err
	}
	return []byte(res.CodePrintable.CodeHash), nil
}

// NonceAt returns the account nonce of the given account.
// The block number can be nil, in which case the nonce is taken from the latest known block.
func (ec *Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return 0, nil
}

// Filters

// FilterLogs executes a filter query.
func (ec *Client) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

// SubscribeFilterLogs subscribes to the results of a streaming filter query.
func (ec *Client) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	return ec.c.EthSubscribe(ctx, ch, "logs", arg)
}

func toFilterArg(q ethereum.FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{
		"address": q.Addresses,
		"topics":  q.Topics,
	}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock == nil {
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = toBlockNumArg(q.FromBlock)
		}
		arg["toBlock"] = toBlockNumArg(q.ToBlock)
	}
	return arg, nil
}

// Pending State

// PendingBalanceAt returns the wei balance of the given account in the pending state.
func (ec *Client) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	return nil, nil
}

// PendingStorageAt returns the value of key in the contract storage of the given account in the pending state.
func (ec *Client) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	return nil, nil
}

// PendingCodeAt returns the contract code of the given account in the pending state.
func (ec *Client) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	var res xwctypes.RpcContractJson
	conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(account[:]))

	err := ec.c.CallContext(ctx, &res, "get_contract_info", conAddr)
	if err != nil {
		return nil, err
	}
	return []byte(res.CodePrintable.CodeHash), nil
}

// PendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (ec *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result hexutil.Uint64
	err := ec.c.CallContext(ctx, &result, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

// PendingTransactionCount returns the total number of transactions in the pending state.
func (ec *Client) PendingTransactionCount(ctx context.Context) (uint, error) {
	return 0, nil
}

// TODO: SubscribePendingTransactions (needs server side)

// Contract Calling

// CallContract executes a message call transaction, which is directly executed in the VM
// of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be nil, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
func (ec *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	err := json.Unmarshal(msg.Data, &callData)
	if err != nil {
		return nil, err
	}

	result, err := ec.InvokeContractOffline(ctx, *msg.To, callData.CallApi, callData.CallArgs)
	if err != nil {
		return nil, err
	}
	return []byte(result), nil
}

// PendingCallContract executes a message call transaction using the EVM.
// The state seen by the contract call is the pending state.
func (ec *Client) PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	err := json.Unmarshal(msg.Data, &callData)
	if err != nil {
		return nil, err
	}

	result, err := ec.InvokeContractOffline(ctx, *msg.To, callData.CallApi, callData.CallArgs)
	if err != nil {
		return nil, err
	}
	return []byte(result), nil
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (ec *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := ec.c.CallContext(ctx, &hex, "eth_gasPrice"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

// EstimateGas tries to estimate the gas needed to execute a specific transaction based on
// the current pending state of the backend blockchain. There is no guarantee that this is
// the true gas limit requirement as other transactions may be added or removed by miners,
// but it should provide a basis for setting a reasonable default.
func (ec *Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	var hex hexutil.Uint64
	err := ec.c.CallContext(ctx, &hex, "eth_estimateGas", toCallArg(msg))
	if err != nil {
		return 0, err
	}
	return uint64(hex), nil
}

func (ec *Client) InvokeContractOffline(ctx context.Context, account common.Address, api string, arg string) (string, error) {
	contractAddr, err := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(account[:]))
	if err != nil {
		return "", err
	}

	var result string
	err = ec.c.CallContext(ctx, &result, "invoke_contract_offline", property.OfflineCaller, contractAddr, api, arg)
	if err != nil {
		return "", err
	}
	return result, nil
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func (ec *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	return ec.c.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data))
}

func (ec *Client) SendXwcTransaction(ctx context.Context, tx *xwcfmt.Transaction) (common.Hash, error) {
	var result string
	err := ec.c.CallContext(ctx, &result, "lightwallet_broadcast", *tx)
	if err != nil {
		return common.Hash{}, err
	}
	hashBytes, _ := hex.DecodeString(result)
	var hash common.Hash
	hash.SetBytes(hashBytes)
	return hash, nil
}

func (ec *Client) RawCall(ctx context.Context, id uint64, data interface{}) (*rpc.JsonrpcMessage, error) {
	return ec.c.RawCallContext(ctx, id, data)
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
