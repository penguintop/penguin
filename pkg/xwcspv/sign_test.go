package xwcspv

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcclient"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"testing"
)

// transfer xwc to normal account
func TestXwcSignTx1(t *testing.T) {
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
	_, tx, _ := XwcBuildTxTransfer(refBlockNum, refBlockPrefix, fromAddr, toAddr, amount, fee, memo)

	privKeyWif := "5KcnSNrBJEdGAcmjVzzThtpncNtuZDDf74Fj81sEvYYkij7bs6u"
	txSig, txSigned, _ := XwcSignTx(property.CHAIN_ID, tx, privKeyWif)
	fmt.Println("XwcSignTx1 Sig:", hex.EncodeToString(txSig))
	txJson, _ := json.Marshal(*txSigned)
	fmt.Println("XwcSignTx1 Tx:", string(txJson))
}

// transfer xwc to contract
func TestXwcSignTx2(t *testing.T) {
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
	conAddr := "XWCCL3Jsf32yGfcHjnB3mu8DEPjL3nerUnjrR"

	pubKeyWif := "XWC6KL1fEMwbVVBUARcfueMGZSewrPcUVRtKipo5aE9JpHREDjsvg"
	hexPubKey, _ := xwcfmt.XwcPubkeyToHexPubkey(pubKeyWif)

	amount := uint64(1000000)
	// total fee
	fee := uint64(2000000)

	gasPrice := uint64(10)
	gasLimit := uint64(100000)
	param := "XWC6KL1fEMwbVVBUARcfueMGZSewrPcUVRtKipo5aE9JpHREDjsvg"

	_, tx, _ := XwcBuildTxTransferToContract(refBlockNum, refBlockPrefix, fromAddr, hexPubKey, conAddr, fee, gasPrice, gasLimit, amount, param)

	privKeyWif := "5KcnSNrBJEdGAcmjVzzThtpncNtuZDDf74Fj81sEvYYkij7bs6u"
	txSig, txSigned, _ := XwcSignTx(property.CHAIN_ID, tx, privKeyWif)
	fmt.Println("XwcSignTx2 Sig:", hex.EncodeToString(txSig))
	txJson, _ := json.Marshal(*txSigned)
	fmt.Println("XwcSignTx2 Tx:", string(txJson))
}

// invoke contract
func TestXwcSignTx3(t *testing.T) {
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
	conAddr := "XWCCL3Jsf32yGfcHjnB3mu8DEPjL3nerUnjrR"

	pubKeyWif := "XWC6KL1fEMwbVVBUARcfueMGZSewrPcUVRtKipo5aE9JpHREDjsvg"
	hexPubKey, _ := xwcfmt.XwcPubkeyToHexPubkey(pubKeyWif)

	// total fee
	fee := uint64(2000000)

	gasPrice := uint64(10)
	gasLimit := uint64(100000)
	conApi := "setERC20Address"
	conArg := "XWCCbayhzZMXu1Q9ab2qzWSbG3MC9T1tR1fKB"

	_, tx, _ := XwcBuildTxInvokeContract(refBlockNum, refBlockPrefix, fromAddr, hexPubKey, conAddr, fee, gasPrice, gasLimit, conApi, conArg)

	privKeyWif := "5KcnSNrBJEdGAcmjVzzThtpncNtuZDDf74Fj81sEvYYkij7bs6u"
	txSig, txSigned, _ := XwcSignTx(property.CHAIN_ID, tx, privKeyWif)
	fmt.Println("XwcSignTx3 Sig:", hex.EncodeToString(txSig))
	txJson, _ := json.Marshal(*txSigned)
	fmt.Println("XwcSignTx3 Tx:", string(txJson))
}
