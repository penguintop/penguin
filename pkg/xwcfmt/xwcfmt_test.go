package xwcfmt

import (
	"fmt"
	"testing"
)

func TestWifKeyToHexKey(t *testing.T) {
	wifKey := "5KcnSNrBJEdGAcmjVzzThtpncNtuZDDf74Fj81sEvYYkij7bs6u"
	hexKey, _ := WifKeyToHexKey(wifKey)
	fmt.Println("TestWifKeyToHexKey:", hexKey)
}

func TestHexKeyToWifKey(t *testing.T) {
	hexKey := "ed4640fd09578c07bc180298b8ea4f454d0daa2fff791b5f4b3e9ae42b0e4af5"
	wifKey, _ := HexKeyToWifKey(hexKey)
	fmt.Println("TestHexKeyToWifKey:", wifKey)
}

func TestXwcAddrToHexAddr(t *testing.T) {
	xwcAddr := "XWCNdbgFmQia2i58PcH918kSPMLrtwZ4kwK2V"
	hexAddr, _ := XwcAddrToHexAddr(xwcAddr)
	fmt.Println("TestXwcAddrToHexAddr:", hexAddr)
}

func TestHexAddrToXwcAddr(t *testing.T) {
	hexAddr := "c1fca4c50a85ad2dec15732c760aef8a1360dcfe"
	xwcAddr, _ := HexAddrToXwcAddr(hexAddr)
	fmt.Println("TestHexAddrToXwcAddr:", xwcAddr)
}

func TestXwcConAddrToHexAddr(t *testing.T) {
	xwcConAddr := "XWCCKnETx6f26x3XMpbJ7sYiXDKE2Vta8JkVJ"
	hexAddr, _ := XwcConAddrToHexAddr(xwcConAddr)
	fmt.Println("TestXwcConAddrToHexAddr:", hexAddr)
}

func TestHexAddrToXwcConAddr(t *testing.T) {
	hexAddr := "246078dfe115f112ee8c1bc0e8b8953b2ab570b7"
	xwcConAddr, _ := HexAddrToXwcConAddr(hexAddr)
	fmt.Println("TestHexAddrToXwcConAddr:", xwcConAddr)
}

func TestXwcPubkeyToHexPubkey(t *testing.T) {
	xwcPubkey := "XWC6KL1fEMwbVVBUARcfueMGZSewrPcUVRtKipo5aE9JpHREDjsvg"
	hexPubkey, _ := XwcPubkeyToHexPubkey(xwcPubkey)
	fmt.Println("XwcPubkeyToHexPubkey:", hexPubkey)
}

func TestHexPubkeyToXwcPubkey(t *testing.T) {
	hexPubkey := "02bc18900d005a1e832c4f4e0d41d90037e281d759441d9c8075d3c2c07b13d0b0"
	xwcPubkey, _ := HexPubkeyToXwcPubkey(hexPubkey)
	fmt.Println("HexPubkeyToXwcPubkey:", xwcPubkey)
}

func TestXwcPubkeyToXwcAddr(t *testing.T) {
	XwcPubkey := "XWC6KL1fEMwbVVBUARcfueMGZSewrPcUVRtKipo5aE9JpHREDjsvg"
	XwcAddr, _ := XwcPubkeyToXwcAddr(XwcPubkey)
	fmt.Println("XwcPubkeyToXwcAddr:", XwcAddr)
}
