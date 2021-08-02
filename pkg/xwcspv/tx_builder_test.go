package xwcspv

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/penguintop/penguin/pkg/xwcclient"
	"testing"
)

func TestXwcBuildTxTransfer(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	endpoint := "http://192.168.1.123:19890/rpc"
	backend, err := xwcclient.Dial(endpoint)
	if err != nil {
		fmt.Println("Dail to endpoint fail:", err)
		return
	}
	refBlockNum, refBlockPrefix, err := backend.RefBlockInfo(ctx)
	if err != nil {
		fmt.Println("backend.RefBlockInfo fail:", err)
		return
	}

	fromAddr := "XWCNdbgFmQia2i58PcH918kSPMLrtwZ4kwK2V"
	toAddr := "XWCNdbgFmQia2i58PcH918kSPMLrtwZ4kwK2V"
	amount := uint64(1000000)
	fee := uint64(1000000)
	memo := "test"

	txBytes, tx, _ := XwcBuildTxTransfer(refBlockNum, refBlockPrefix, fromAddr, toAddr, amount, fee, memo)
	fmt.Println("XwcBuildTxTransfer Hex:", hex.EncodeToString(txBytes))
	txJson, _ := json.Marshal(*tx)
	fmt.Println("XwcBuildTxTransfer Tx:", string(txJson))
}
