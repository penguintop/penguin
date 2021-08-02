package auditor

import (
	"encoding/hex"
	"fmt"
	"testing"
)

var (
	hash1, _ = hex.DecodeString("6995c6609e2240d814151600ea992f8c065358207cff6f65d2f704c99210e2bc")
	hash2, _ = hex.DecodeString("62bdfe3880283f3f95e73f81783169383d02c28a6ce46f49f9070dd6e7f9e300")
	hash3, _ = hex.DecodeString("0cfcef7924808894fc1d8fb6acfac1a68df37f726951278a4a5bc04ae872e0dc")
	hash4, _ = hex.DecodeString("73728a5f008e4fb929f5befb5b050de075ad5294b1b1c8575094a2c410b621f7")
	hash5, _ = hex.DecodeString("281a2cff7d373620b9b8321e531c5cacab4dc049fbd0a68b448ca5996a127643")
	hash6, _ = hex.DecodeString("4056c176870315c0d8187cb24cdf2d0fd99193ec166c8d0fef8f54607665a753")
	hash7, _ = hex.DecodeString("75096fbeb236476b094322183fd0e1d44637946e9481353d4bfde11d8cddc1fa")
	hash8, _ = hex.DecodeString("d9f3f5f10a7730dd14984472539d215048a996c1cc374b8300d046847f1b7f02")
)

func TestBtree(t *testing.T) {
	retrievalAddresses := [][]byte{
		hash1, hash2, hash3, hash4,
		hash5, hash6, hash7, hash8,
	}

	treeRootNode, err := BuildBTreeFromRetrievalAddresses(retrievalAddresses)
	if err != nil {
		fmt.Println("BuildBTreeFromRetrievalAddresses, err:", err)
		return
	}

	rootHashHex, nextHashHexPair := treeRootNode.GetRootRelatedHashHex()
	if err != nil {
		fmt.Println("GetRootRelatedHashHex, err:", err)
		return
	}

	fmt.Println("rootHashHex:", rootHashHex)
	fmt.Println("leftSonHashHex:", nextHashHexPair[0])
	fmt.Println("rightSonHashHex:", nextHashHexPair[1])
}
