package property

import (
	"strings"
	"time"
)

const (
	GOERLI_CHAIN_ID     string = "a3c762d4c7bcbbfa59327c35c2a6e98558f6ca90d9fd71dfc59a15d09c8c52e4"
	GOERLI_CHAIN_ID_NUM int64  = 51107           //compute: (0xc7<<8) + (0xa3)
	XWC_ASSET_ID               = "1.3.0"
	XWC_ASSET_ID_NUM           = 0

	XWC_ASSET_PRCISION = 100000000
	PEN_ERC20_PRCISION = 100000000
)

var CHAIN_ID string = GOERLI_CHAIN_ID
var CHAIN_ID_NUM int64 = GOERLI_CHAIN_ID_NUM
var DOMAIN_ID string = ""

var OfflineCaller string = "pen-caller"

// [Test] The receiver account's address
var EntranceAddress string = "XWCNhr1NeszTwe75B8qGehaTPPku6TVJEg9kU"

// [Test]factory contract
var FactoryAddress string = "XWCCJ8pa6Bz2Un3fTT8QRu7u1XRUG5QpVbRsd"

// [Test]postage stamp contract
var PostageStampAddress string = "XWCCYzE9banNUQhXB1JYVvWL4ynmxoBRh2j9W"

// [Test]staking contract
var StakingAddress string = "XWCCJy83LxyFm1zFZ9SxfJw18orqbNjP4R3ZM"

// [Test]factory contract code hash
var FactoryDeployedCodeHash = []byte("0d17f58ca648876af170ebed1a697cdbba8680cd")

// [Test]postage stamp contract code hash
var PostageStampDeployedCodeHash = []byte("b8ce22d64dcf5df1bb0b2cc5e94c6d3e4020a76b")

// [Test]staking contract code hash
var StakingAddressDeployedCodeHash = []byte("a9b34ce16a1be02469729ad0cbd5ac79e8560929")

// [Test]cheque book contract code hash(refer: XRC20SimpleSwap.glua, python service will create it)
var ChequeBookDeployedCodeHash = []byte("482186ef6e356cf90b087e4d300c776d7ec3e39a")

// [Test]staking admin
var StakingAdmin = []byte("XWCNRLpDWifjQeTz27Kf5Sos5iH9LRgcbZo6D")

func RFC3339ToUTC(timeFormatStr string) (uint64, error) {
	t, err := time.Parse(
		time.RFC3339, timeFormatStr+"+00:00")
	if err != nil {
		return 0, err
	}
	return uint64(t.Unix()), nil
}

func UTCToRFC3339(t uint64) string {
	timeStr := time.Unix(int64(t), 0).UTC().String()
	timeStr = timeStr[0:10] + "T" + timeStr[11:19]
	return timeStr
}

func Domain() string {
	return strings.Join([]string{CHAIN_ID, DOMAIN_ID}, "")
}
