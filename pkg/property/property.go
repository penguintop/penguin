package property

import (
	"strings"
	"time"
)

const (
	GOERLI_CHAIN_ID     string = "edad429038838754f84b4214613e33985e5957b09beec04ba4610bbb38fa12d0"
	GOERLI_CHAIN_ID_NUM int64  = 44525           //compute: (0xad<<8) + (0xed)
	XWC_ASSET_ID               = "1.3.0"
	XWC_ASSET_ID_NUM           = 0

	XWC_ASSET_PRCISION = 100000000
	PEN_ERC20_PRCISION = 100000000
)

var CHAIN_ID string = GOERLI_CHAIN_ID
var CHAIN_ID_NUM int64 = GOERLI_CHAIN_ID_NUM
var DOMAIN_ID string = ""

var OfflineCaller string = "penguin"

// [Test] The receiver account's address
var EntranceAddress string = "XWCNbncadY7UwGQpyP5jBMj8FLA9jTDDNU18E"

// [Test]factory contract
var FactoryAddress string = "XWCCbcYhJjF9vdt9Riv43Gat4ad2MnLCAYTeu"

// [Test]postage stamp contract
var PostageStampAddress string = "XWCCRyYKg5tn9yZ8gDfNAnGMX3HrZKs8g9hQG"

// [Test]staking contract
var StakingAddress string = "XWCCV6F9LNQ3XR3kSo75YnKaxipbnvt5rHbCL"

// [Test]factory contract code hash
var FactoryDeployedCodeHash = []byte("0d17f58ca648876af170ebed1a697cdbba8680cd")

// [Test]postage stamp contract code hash
var PostageStampDeployedCodeHash = []byte("b8ce22d64dcf5df1bb0b2cc5e94c6d3e4020a76b")

// [Test]staking contract code hash
var StakingAddressDeployedCodeHash = []byte("a9b34ce16a1be02469729ad0cbd5ac79e8560929")

// [Test]cheque book contract code hash(refer: XRC20SimpleSwap.glua, python service will create it)
var ChequeBookDeployedCodeHash = []byte("482186ef6e356cf90b087e4d300c776d7ec3e39a")

// [Test]staking admin
var StakingAdmin = []byte("XWCNbncadY7UwGQpyP5jBMj8FLA9jTDDNU18E")

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
