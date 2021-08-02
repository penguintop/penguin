package xwcclient

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"
	"testing"
)

func TestIsLock(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	isLock, err := backend.IsLocked(ctx)
	if err != nil {
		fmt.Println("backend.IsLocked fail:", err)
		return
	}
	fmt.Println("isLock:", isLock)
}

func TestGetAccount(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	acctInfo, err := backend.GetAccount(ctx, "test")
	if err != nil {
		fmt.Println("backend.GetAccount fail:", err)
		return
	}
	fmt.Println("acctInfo:", acctInfo)
}

func TestCreateAccount(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	acctAddr, err := backend.CreateAccount(ctx, "caller")
	if err != nil {
		fmt.Println("backend.CreateAccount fail:", err)
		return
	}
	fmt.Println("acctAddr:", acctAddr)
}

func TestChainID(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	chainId, err := backend.ChainID(ctx)
	if err != nil {
		fmt.Println("backend.ChainId fail:", err)
		return
	}
	fmt.Println("chainId:", chainId)
}

func TestBlockNumber(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	blockNumber, err := backend.BlockNumber(ctx)
	if err != nil {
		fmt.Println("backend.BlockNumber fail:", err)
		return
	}
	fmt.Println("blockNumber:", blockNumber)
}

func TestRefBlockInfo(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	refBlockNum, refBlockPrefix, err := backend.RefBlockInfo(ctx)
	if err != nil {
		fmt.Println("backend.RefBlockInfo fail:", err)
		return
	}
	fmt.Println("RefBlockInfo:", refBlockNum, refBlockPrefix)
}

func TestBlockByNumber(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	block, err := backend.BlockByNumber(ctx, big.NewInt(100))
	if err != nil {
		fmt.Println("backend.BlockByNumber fail:", err)
		return
	}
	fmt.Println("block:", block)
}

func TestTransactionByHash(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	trxIdHex := "7b0f9be321223d54a4ad4029ff8252aaa7c21728"
	var trxId common.Hash
	trxIdBytes, _ := hex.DecodeString(trxIdHex)
	trxId.SetBytes(trxIdBytes)
	tx, pending, err := backend.TransactionByHash(ctx, trxId)

	if err != nil {
		fmt.Println("backend.TransactionByHash fail:", err)
		return
	}
	fmt.Println("tx:", *tx)
	fmt.Println("tx pending:", pending)
}

func TestBalanceAt(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	addrXWC := "XWCNdbgFmQia2i58PcH918kSPMLrtwZ4kwK2V"
	addrHex, _ := xwcfmt.XwcAddrToHexAddr(addrXWC)
	addrBytes, _ := hex.DecodeString(addrHex)
	var addr common.Address
	addr.SetBytes(addrBytes)
	balance, err := backend.BalanceAt(ctx, addr, nil)
	if err != nil {
		fmt.Println("backend.BalanceAt fail:", err)
		return
	}
	fmt.Println("balance:", balance.String())
}

func TestCodeAt(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	addrCon := "XWCCL3Jsf32yGfcHjnB3mu8DEPjL3nerUnjrR"
	addrHex, _ := xwcfmt.XwcConAddrToHexAddr(addrCon)
	addrBytes, _ := hex.DecodeString(addrHex)
	var addr common.Address
	addr.SetBytes(addrBytes)
	codeHash, err := backend.CodeAt(ctx, addr, nil)
	if err != nil {
		fmt.Println("backend.CodeAt fail:", err)
		return
	}
	fmt.Println("code_hash:", string(codeHash))
}
