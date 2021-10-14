package xwcfmt

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/ripemd160"
)

const (
	XWC_PREFIX = "XWC"
)

const (
	WIF_VERSION = 0x80
)

const (
	ADDR_NORMAL   = 0x35
	ADDR_MULTISIG = 0x32
	ADDR_CONTRACT = 0x1c
)

func WifKeyToHexKey(wifKey string) (string, error) {
	keyBytes, err := base58.Decode(wifKey)
	if err != nil {
		return "", err
	}
	if len(keyBytes) != 37 {
		return "", fmt.Errorf("invalid wif key")
	}
	return hex.EncodeToString(keyBytes[1:33]), nil
}

func HexKeyToWifKey(hexKey string) (string, error) {
	hexKeyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", err
	}
	if len(hexKeyBytes) != 32 {
		return "", fmt.Errorf("invalid hex key")
	}
	calcBytes := make([]byte, 0)
	calcBytes = append(calcBytes, WIF_VERSION)
	calcBytes = append(calcBytes, hexKeyBytes...)

	// double sha256
	s256 := sha256.New()
	_, err = s256.Write(calcBytes)
	if err != nil {
		return "", err
	}
	checkSum := s256.Sum(nil)

	s256 = sha256.New()
	_, err = s256.Write(checkSum)
	if err != nil {
		return "", err
	}
	checkSum = s256.Sum(nil)

	calcBytes = append(calcBytes, checkSum[0:4]...)

	return base58.Encode(calcBytes), nil
}

func XwcAddrToHexAddr(xwcAddr string) (string, error) {
	if len(xwcAddr) <= len(XWC_PREFIX) || xwcAddr[0:len(XWC_PREFIX)] != XWC_PREFIX {
		return "", fmt.Errorf("invalid xwc key")
	}
	addrBytes, err := base58.Decode(xwcAddr[len(XWC_PREFIX):])
	if err != nil {
		return "", err
	}
	if len(addrBytes) != 25 {
		return "", fmt.Errorf("invalid xwc key")
	}
	if addrBytes[0] != ADDR_NORMAL {
		return "", fmt.Errorf("invalid xwc key")
	}
	return hex.EncodeToString(addrBytes[1:21]), nil
}

func XwcConAddrToHexAddr(xwcConAddr string) (string, error) {
	if len(xwcConAddr) <= len(XWC_PREFIX) || xwcConAddr[0:len(XWC_PREFIX)] != XWC_PREFIX {
		return "", fmt.Errorf("invalid xwc contract key:%s", xwcConAddr)
	}
	addrBytes, err := base58.Decode(xwcConAddr[len(XWC_PREFIX):])
	if err != nil {
		return "", err
	}
	if len(addrBytes) != 25 {
		return "", fmt.Errorf("invalid contract key")
	}
	if addrBytes[0] != ADDR_CONTRACT {
		return "", fmt.Errorf("invalid contract key")
	}
	return hex.EncodeToString(addrBytes[1:21]), nil
}

func HexAddrToXwcAddr(hexAddr string) (string, error) {
	hexAddrBytes, err := hex.DecodeString(hexAddr)
	if err != nil {
		return "", err
	}
	if len(hexAddrBytes) != 20 {
		return "", fmt.Errorf("invalid hex addr")
	}
	calcBytes := make([]byte, 0)
	calcBytes = append(calcBytes, ADDR_NORMAL)
	calcBytes = append(calcBytes, hexAddrBytes...)

	// ripemd160
	r160 := ripemd160.New()
	_, _ = r160.Write(calcBytes)
	checkSum := r160.Sum(nil)

	calcBytes = append(calcBytes, checkSum[0:4]...)

	return XWC_PREFIX + base58.Encode(calcBytes), nil
}

func HexAddrToXwcConAddr(hexAddr string) (string, error) {
	hexAddrBytes, err := hex.DecodeString(hexAddr)
	if err != nil {
		return "", err
	}
	if len(hexAddrBytes) != 20 {
		return "", fmt.Errorf("invalid hex addr")
	}
	calcBytes := make([]byte, 0)
	calcBytes = append(calcBytes, ADDR_CONTRACT)
	calcBytes = append(calcBytes, hexAddrBytes...)

	// ripemd160
	r160 := ripemd160.New()
	_, _ = r160.Write(calcBytes)
	checkSum := r160.Sum(nil)

	calcBytes = append(calcBytes, checkSum[0:4]...)

	return XWC_PREFIX + base58.Encode(calcBytes), nil
}

func XwcPubkeyToHexPubkey(xwcPubkey string) (string, error) {
	if len(xwcPubkey) <= len(XWC_PREFIX) || xwcPubkey[0:len(XWC_PREFIX)] != XWC_PREFIX {
		return "", fmt.Errorf("invalid xwc pubkey")
	}
	xwcPubkeyBytes, err := base58.Decode(xwcPubkey[len(XWC_PREFIX):])
	if err != nil {
		return "", err
	}
	if len(xwcPubkeyBytes) != 37 {
		return "", fmt.Errorf("invalid xwc pubkey")
	}
	return hex.EncodeToString(xwcPubkeyBytes[0:33]), nil
}

func HexPubkeyToXwcPubkey(hexPubkey string) (string, error) {
	hexPubkeyBytes, err := hex.DecodeString(hexPubkey)
	if err != nil {
		return "", err
	}
	if len(hexPubkeyBytes) != 33 {
		return "", fmt.Errorf("invalid hex pubkey")
	}

	calcBytes := make([]byte, 0)
	calcBytes = append(calcBytes, hexPubkeyBytes...)

	r160 := ripemd160.New()
	_, _ = r160.Write(calcBytes)
	checkSum := r160.Sum(nil)

	calcBytes = append(calcBytes, checkSum[0:4]...)

	return XWC_PREFIX + base58.Encode(calcBytes), nil
}

func XwcPubkeyToXwcAddr(xwcPubkey string) (string, error) {
	if len(xwcPubkey) <= len(XWC_PREFIX) || xwcPubkey[0:len(XWC_PREFIX)] != XWC_PREFIX {
		return "", fmt.Errorf("invalid xwc pubkey")
	}
	xwcPubkeyBytes, err := base58.Decode(xwcPubkey[len(XWC_PREFIX):])
	if err != nil {
		return "", err
	}
	if len(xwcPubkeyBytes) != 37 {
		return "", fmt.Errorf("invalid xwc pubkey")
	}
	pubCpsBytes := xwcPubkeyBytes[0:33]

	// ripemd160(sha512(compress pubkey))
	s512 := sha512.New()
	_, err = s512.Write(pubCpsBytes)
	if err != nil {
		return "", err
	}
	pubHash := s512.Sum(nil)

	r160 := ripemd160.New()
	_, err = r160.Write(pubHash)
	if err != nil {
		return "", err
	}
	pubHash = r160.Sum(nil)

	return HexAddrToXwcAddr(hex.EncodeToString(pubHash))
}
