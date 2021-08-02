package xwcspv

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/penguintop/penguin/pkg/xwcfmt"

	//"fmt"
	secp256k1 "github.com/bitnexty/secp256k1-go"
)

func XwcSignTx(chainIdHex string, tx *xwcfmt.Transaction, privKeyWif string) ([]byte, *xwcfmt.Transaction, error) {
	chainIdBytes, err := hex.DecodeString(chainIdHex)
	if err != nil {
		return nil, nil, err
	}

	txData := tx.Pack()

	//fmt.Println("unsigned tx:", hex.EncodeToString(txData))
	//fmt.Println("chain id:", chainIdHex)

	s256 := sha256.New()
	_, _ = s256.Write(chainIdBytes)
	_, _ = s256.Write(txData)
	digestData := s256.Sum(nil)

	//fmt.Println("digest:", hex.EncodeToString(digestData))

	privKeyHex, _ := xwcfmt.WifKeyToHexKey(privKeyWif)
	privKeyBytes, _ := hex.DecodeString(privKeyHex)

	txSig, err := secp256k1.BtsSign(digestData, privKeyBytes, true)
	if err != nil {
		return nil, nil, err
	}

	// tx data with sig
	txBytesWithSig := make([]byte, 0)
	txBytesWithSig = append(txBytesWithSig, txData...)

	// sig count
	txBytesWithSig = append(txBytesWithSig, xwcfmt.PackVarInt(1)...)

	txBytesWithSig = append(txBytesWithSig, xwcfmt.PackVarInt(uint64(len(txSig)))...)
	txBytesWithSig = append(txBytesWithSig, txSig...)

	tx.Signatures = append(tx.Signatures, txSig)

	return txSig, tx, nil
}
