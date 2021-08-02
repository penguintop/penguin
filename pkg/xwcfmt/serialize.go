package xwcfmt

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/penguintop/penguin/pkg/property"
)

func PackUint8(v uint8) []byte {
	return []byte{v}
}

func PackUint16(v uint16) []byte {
	res := make([]byte, 2)
	binary.LittleEndian.PutUint16(res, v)
	return res
}

func PackUint32(v uint32) []byte {
	res := make([]byte, 4)
	binary.LittleEndian.PutUint32(res, v)
	return res
}

func PackUint64(v uint64) []byte {
	res := make([]byte, 8)
	binary.LittleEndian.PutUint64(res, v)
	return res
}

func PackVarInt(v uint64) []byte {
	res := make([]byte, 0)
	var dest uint8 = 0
	for {
		if v < 0x80 {
			break
		} else {
			dest = uint8((v & 0x7f) | 0x80)
			v = v >> 7
		}
		res = append(res, dest)
	}
	res = append(res, uint8(v))
	return res
}

type OperationType interface {
	Pack() []byte
}

type Asset struct {
	Amount  int64  `json:"amount"`
	AssetId string `json:"asset_id"`
}

func (a *Asset) SetDefault() {
	a.Amount = 0
	a.AssetId = property.XWC_ASSET_ID
}

func (a Asset) Pack() []byte {
	bytesRet := make([]byte, 0)
	bytesAmount := PackUint64(uint64(a.Amount))
	bytesAssetId := PackUint8(uint8(property.XWC_ASSET_ID_NUM))
	bytesRet = append(bytesRet, bytesAmount...)
	bytesRet = append(bytesRet, bytesAssetId...)
	return bytesRet
}

type Memo struct {
	From    PubKey      `json:"from"`
	To      PubKey      `json:"to"`
	Nonce   uint64      `json:"nonce"`
	Message MemoMessage `json:"message"`
}

func (m Memo) Pack() []byte {
	bytesRet := make([]byte, 0)
	bytesRet = append(bytesRet, m.From[:]...)
	bytesRet = append(bytesRet, m.To[:]...)
	bytesNonce := PackUint64(m.Nonce)
	bytesRet = append(bytesRet, bytesNonce...)
	// length
	bytesLength := PackVarInt(uint64(len(m.Message) + 4))
	bytesRet = append(bytesRet, bytesLength...)
	// checksum
	bytesRet = append(bytesRet, []byte{0, 0, 0, 0}...)
	bytesRet = append(bytesRet, []byte(m.Message)...)

	return bytesRet
}

type TransferOperation struct {
	Fee         Asset         `json:"fee"`
	GuaranteeId string        `json:"guarantee_id,omitempty"`
	From        string        `json:"from"`
	To          string        `json:"to"`
	FromAddr    Address       `json:"from_addr"`
	ToAddr      Address       `json:"to_addr"`
	Amount      Asset         `json:"amount"`
	Memo        *Memo         `json:"memo,omitempty"`
	Extensions  []interface{} `json:"extensions"`
}

func (to *TransferOperation) SetValue(fromAddr string, toAddr string,
	amount uint64, fee uint64, memo string) error {

	to.Fee.SetDefault()
	to.Fee.Amount = int64(fee)

	to.Amount.SetDefault()
	to.Amount.Amount = int64(amount)

	to.From = "1.2.0"
	to.To = "1.2.0"

	to.Extensions = make([]interface{}, 0)

	fromAddrHex, err := XwcAddrToHexAddr(fromAddr)
	if err != nil {
		return err
	}
	fromAddrBytes, _ := hex.DecodeString(fromAddrHex)
	to.FromAddr.SetBytes(fromAddrBytes)

	toAddrHex, err := XwcAddrToHexAddr(toAddr)
	if err != nil {
		return err
	}
	toAddrBytes, _ := hex.DecodeString(toAddrHex)
	to.ToAddr.SetBytes(toAddrBytes)

	if len(memo) > 0 {
		to.Memo = &Memo{Message: MemoMessage(memo)}
	} else {
		to.Memo = nil
	}

	return nil
}

func (to *TransferOperation) Pack() []byte {
	bytesRet := make([]byte, 0)
	bytesMemo := make([]byte, 0)

	bytesFee := to.Fee.Pack()
	bytesAmount := to.Amount.Pack()

	if to.Memo != nil {
		bytesMemo = to.Memo.Pack()
	}

	bytesRet = append(bytesRet, bytesFee...)
	//guarantee_id
	bytesRet = append(bytesRet, byte(0))
	//from
	bytesRet = append(bytesRet, byte(0))
	//to
	bytesRet = append(bytesRet, byte(0))

	bytesRet = append(bytesRet, byte(ADDR_NORMAL))
	bytesRet = append(bytesRet, to.FromAddr[:]...)
	bytesRet = append(bytesRet, byte(ADDR_NORMAL))
	bytesRet = append(bytesRet, to.ToAddr[:]...)
	bytesRet = append(bytesRet, bytesAmount...)
	if to.Memo != nil {
		bytesRet = append(bytesRet, byte(1))
		bytesRet = append(bytesRet, bytesMemo...)
	} else {
		bytesRet = append(bytesRet, byte(0))
	}
	// Extensions
	bytesRet = append(bytesRet, PackVarInt(uint64(len(to.Extensions)))...)

	return bytesRet
}

type ContractInvokeOperation struct {
	Fee          Asset      `json:"fee"`
	InvokeCost   uint64     `json:"invoke_cost"`
	GasPrice     uint64     `json:"gas_price"`
	CallerAddr   Address    `json:"caller_addr"`
	CallerPubkey PubKeyType `json:"caller_pubkey"`
	ContractId   ConAddress `json:"contract_id"`
	ContractApi  string     `json:"contract_api"`
	ContractArg  string     `json:"contract_arg"`
	GuaranteeId  string     `json:"guarantee_id,omitempty"`
}

func (cio *ContractInvokeOperation) SetValue(callerAddr string,
	callerPubKey string, conAddr string, fee uint64, gasPrice uint64,
	gasLimit uint64, conApi string, conArg string) error {

	cio.Fee.SetDefault()
	cio.Fee.Amount = int64(fee)

	cio.GasPrice = gasPrice
	cio.InvokeCost = gasLimit

	callerAddrHex, err := XwcAddrToHexAddr(callerAddr)
	if err != nil {
		return err
	}
	fromAddrBytes, _ := hex.DecodeString(callerAddrHex)
	cio.CallerAddr.SetBytes(fromAddrBytes)

	callerPubKeyBytes, _ := hex.DecodeString(callerPubKey)
	copy(cio.CallerPubkey[:], callerPubKeyBytes)

	conAddrHex, err := XwcConAddrToHexAddr(conAddr)
	if err != nil {
		return err
	}
	conAddrBytes, _ := hex.DecodeString(conAddrHex)
	cio.ContractId.SetBytes(conAddrBytes)

	cio.ContractApi = conApi
	cio.ContractArg = conArg

	return nil
}

func (cio *ContractInvokeOperation) Pack() []byte {
	bytesRet := make([]byte, 0)
	bytesFee := cio.Fee.Pack()

	bytesInvokeCost := PackUint64(cio.InvokeCost)
	bytesGasPrice := PackUint64(cio.GasPrice)

	bytesRet = append(bytesRet, bytesFee...)
	bytesRet = append(bytesRet, bytesInvokeCost...)
	bytesRet = append(bytesRet, bytesGasPrice...)

	bytesRet = append(bytesRet, byte(ADDR_NORMAL))
	bytesRet = append(bytesRet, cio.CallerAddr[:]...)
	bytesRet = append(bytesRet, cio.CallerPubkey[:]...)
	bytesRet = append(bytesRet, byte(ADDR_CONTRACT))
	bytesRet = append(bytesRet, cio.ContractId[:]...)

	bytesRet = append(bytesRet, PackVarInt(uint64(len(cio.ContractApi)))...)
	bytesRet = append(bytesRet, []byte(cio.ContractApi)...)

	bytesRet = append(bytesRet, PackVarInt(uint64(len(cio.ContractArg)))...)
	bytesRet = append(bytesRet, []byte(cio.ContractArg)...)

	//guarantee_id
	bytesRet = append(bytesRet, byte(0))

	return bytesRet
}

type ContractTransferOperation struct {
	Fee          Asset      `json:"fee"`
	InvokeCost   uint64     `json:"invoke_cost"`
	GasPrice     uint64     `json:"gas_price"`
	CallerAddr   Address    `json:"caller_addr"`
	CallerPubkey PubKeyType `json:"caller_pubkey"`
	ContractId   ConAddress `json:"contract_id"`
	Amount       Asset      `json:"amount"`
	Param        string     `json:"param"`
	GuaranteeId  string     `json:"guarantee_id,omitempty"`
}

func (cto *ContractTransferOperation) SetValue(callerAddr string,
	callerPubKey string, conAddr string, fee uint64, gasPrice uint64,
	gasLimit uint64, amount uint64, param string) error {

	cto.Fee.SetDefault()
	cto.Fee.Amount = int64(fee)

	cto.Amount.SetDefault()
	cto.Amount.Amount = int64(amount)

	cto.GasPrice = gasPrice
	cto.InvokeCost = gasLimit

	callerAddrHex, err := XwcAddrToHexAddr(callerAddr)
	if err != nil {
		return err
	}
	fromAddrBytes, _ := hex.DecodeString(callerAddrHex)
	cto.CallerAddr.SetBytes(fromAddrBytes)

	callerPubKeyBytes, _ := hex.DecodeString(callerPubKey)
	copy(cto.CallerPubkey[:], callerPubKeyBytes)

	conAddrHex, err := XwcConAddrToHexAddr(conAddr)
	if err != nil {
		return err
	}
	conAddrBytes, _ := hex.DecodeString(conAddrHex)
	cto.ContractId.SetBytes(conAddrBytes)

	cto.Param = param

	return nil
}

func (cto *ContractTransferOperation) Pack() []byte {
	bytesRet := make([]byte, 0)
	bytesFee := cto.Fee.Pack()
	bytesAmount := cto.Amount.Pack()

	bytesInvokeCost := PackUint64(cto.InvokeCost)
	bytesGasPrice := PackUint64(cto.GasPrice)

	bytesRet = append(bytesRet, bytesFee...)
	bytesRet = append(bytesRet, bytesInvokeCost...)
	bytesRet = append(bytesRet, bytesGasPrice...)

	bytesRet = append(bytesRet, byte(ADDR_NORMAL))
	bytesRet = append(bytesRet, cto.CallerAddr[:]...)
	bytesRet = append(bytesRet, cto.CallerPubkey[:]...)
	bytesRet = append(bytesRet, byte(ADDR_CONTRACT))
	bytesRet = append(bytesRet, cto.ContractId[:]...)

	bytesRet = append(bytesRet, bytesAmount...)

	bytesRet = append(bytesRet, PackVarInt(uint64(len(cto.Param)))...)
	bytesRet = append(bytesRet, []byte(cto.Param)...)

	//guarantee_id
	bytesRet = append(bytesRet, byte(0))

	return bytesRet
}

type OperationPair [2]interface{}

type Transaction struct {
	RefBlockNum    uint16          `json:"ref_block_num"`
	RefBlockPrefix uint32          `json:"ref_block_prefix"`
	Expiration     UTCTime         `json:"expiration"`
	Operations     []OperationPair `json:"operations"`
	Extensions     []interface{}   `json:"extensions"`
	Signatures     []Signature     `json:"signatures"`
}

func (tx *Transaction) Pack() []byte {
	bytesRet := make([]byte, 0)

	bytesRefBlockNum := PackUint16(tx.RefBlockNum)
	bytesRefBlockPrefix := PackUint32(tx.RefBlockPrefix)
	bytesExpiration := PackUint32(uint32(tx.Expiration))

	bytesRet = append(bytesRet, bytesRefBlockNum...)
	bytesRet = append(bytesRet, bytesRefBlockPrefix...)
	bytesRet = append(bytesRet, bytesExpiration...)

	bytesRet = append(bytesRet, PackVarInt(uint64(len(tx.Operations)))...)
	for _, opPair := range tx.Operations {
		bytesRet = append(bytesRet, PackVarInt(uint64(opPair[0].(byte)))...)
		bytesOP := opPair[1].(OperationType).Pack()
		bytesRet = append(bytesRet, bytesOP...)
	}

	//extension
	bytesRet = append(bytesRet, byte(0))

	//without sig
	return bytesRet
}

// TODO
type Code struct {
	// need to order by ascii
	Abi []string `json:"fee"`
	// need to order by ascii
	OfflineAbi []string `json:"offline_abi"`
	// need to order by ascii
	Events []string `json:"events"`
	// need to order by ascii
	StorageProperties map[string]uint32 `json:"storage_properties"`
	Code              []byte            `json:"code"`
	CodeHash          string            `json:"code_hash"`
}

func (c *Code) LoadFromHex(codeHex string) error {
	leftBytes, err := hex.DecodeString(codeHex)
	if err != nil {
		return err
	}
	codeHash := leftBytes[0:20]
	leftBytes = leftBytes[20:]

	codeLength := bytesToNumber(leftBytes[0:4])
	codeBytes := leftBytes[4 : codeLength+4]

	s1 := sha1.New()
	_, _ = s1.Write(codeBytes)
	codeHashCalc := s1.Sum(nil)
	if bytes.Compare(codeHash, codeHashCalc) != 0 {
		return errors.New("Invalid CodeHash")
	}

	c.Code = codeBytes
	c.CodeHash = hex.EncodeToString(codeHashCalc)
	//fmt.Println("codehash:", c.CodeHash)

	leftBytes = leftBytes[codeLength+4:]

	// abi
	offSet := 0
	abiCount := bytesToNumber(leftBytes[offSet : offSet+4])
	offSet += 4
	for i := 0; i < int(abiCount); i++ {
		abiLength := bytesToNumber(leftBytes[offSet : offSet+4])
		offSet += 4
		c.Abi = append(c.Abi, string(leftBytes[offSet:offSet+int(abiLength)]))
		offSet += int(abiLength)
	}
	leftBytes = leftBytes[offSet:]
	//fmt.Println("abi:", c.Abi)

	// offline_abi
	offSet = 0
	offlineAbiCount := bytesToNumber(leftBytes[offSet : offSet+4])
	offSet += 4
	for i := 0; i < int(offlineAbiCount); i++ {
		offlineAbiLength := bytesToNumber(leftBytes[offSet : offSet+4])
		offSet += 4
		c.OfflineAbi = append(c.OfflineAbi, string(leftBytes[offSet:offSet+int(offlineAbiLength)]))
		offSet += int(offlineAbiLength)
	}
	leftBytes = leftBytes[offSet:]
	//fmt.Println("offline_abi:", c.OfflineAbi)

	// events
	offSet = 0
	eventsCount := bytesToNumber(leftBytes[offSet : offSet+4])
	offSet += 4
	for i := 0; i < int(eventsCount); i++ {
		eventsLength := bytesToNumber(leftBytes[offSet : offSet+4])
		offSet += 4
		c.Events = append(c.Events, string(leftBytes[offSet:offSet+int(eventsLength)]))
		offSet += int(eventsLength)
	}
	leftBytes = leftBytes[offSet:]
	//fmt.Println("offline_abi:", c.OfflineAbi)

	// storage
	offSet = 0
	storageCount := bytesToNumber(leftBytes[offSet : offSet+4])
	offSet += 4
	c.StorageProperties = make(map[string]uint32)
	for i := 0; i < int(storageCount); i++ {
		storageLength := bytesToNumber(leftBytes[offSet : offSet+4])
		offSet += 4
		storageBuf := string(leftBytes[offSet : offSet+int(storageLength)])
		offSet += int(storageLength)
		storageType := bytesToNumber(leftBytes[offSet : offSet+4])
		offSet += 4
		c.StorageProperties[storageBuf] = storageType
	}
	leftBytes = leftBytes[offSet:]
	//fmt.Println("storage:", c.StorageProperties)

	return nil
}

type ContractRegisterOperation struct {
	Fee          Asset         `json:"fee"`
	InitCost     uint64        `json:"init_cost"`
	GasPrice     uint64        `json:"gas_price"`
	OwnerAddr    Address       `json:"owner_addr"`
	OwnerPubkey  PubKey        `json:"owner_pubkey"`
	RegisterTime uint32        `json:"register_time"`
	ContractId   ConAddress    `json:"contract_id"`
	ContractCode Code          `json:"contract_code"`
	InheritFrom  ConAddress    `json:"inherit_from"`
	Extensions   []interface{} `json:"extensions"`
	GuaranteeId  string        `json:"guarantee_id,omitempty"`
}

func bytesToNumber(bs []byte) uint32 {
	if len(bs) != 4 {
		return 0
	}
	return (uint32(bs[0]) << 24) + (uint32(bs[1]) << 16) + (uint32(bs[2]) << 8) + uint32(bs[3])
}
