package property

import (
	"strings"
	"time"
)

const (
	GOERLI_CHAIN_ID     string = "b8a2de81756468753e02e2212da5f7ac9294532c777eb68099edd52e02732baa"
	GOERLI_CHAIN_ID_NUM int64  = 41656
	XWC_ASSET_ID               = "1.3.0"
	XWC_ASSET_ID_NUM           = 0

	XWC_ASSET_PRCISION = 100000000
	PEN_ERC20_PRCISION = 100000000
)

var CHAIN_ID string = GOERLI_CHAIN_ID
var CHAIN_ID_NUM int64 = GOERLI_CHAIN_ID_NUM
var DOMAIN_ID string = ""

var OfflineCaller string = "caller0"

// factory contract
var FactoryAddress string = "XWCCL3Jsf32yGfcHjnB3mu8DEPjL3nerUnjrR"

var EntranceAddress string = "XWCNhv8RU2cwPnLp1QEPDbsthL84ScYQNnpSY"

// postage stamp contract
var PostageStampAddress string = "XWCCR8hm6VueidBYC9GKkqQyXL2AhvKqyuzMt"

// staking contract
var StakingAddress string = "XWCCYyCKBTE6dVt8cPSTw1PxZYTs5NXvht3Fg"

// factory contract code hash
var FactoryDeployedCodeHash = []byte("f72c6b8fefe2410a9e3cf0caff1b53d068647f59")

// cheque book contract code hash
var ChequeBookDeployedCodeHash = []byte("8ade25fa0ddc62fc5e1765fba1cfe606160c5d74")

// postage stamp contract code hash
var PostageStampDeployedCodeHash = []byte("c8be25557e765d5c1fa26610c3ee93ed1ce7a065")

// staking contract code hash
var StakingAddressDeployedCodeHash = []byte("a9b34ce16a1be02469729ad0cbd5ac79e8560929")

// staking admin
var StakingAdmin = []byte("XWCNdbgFmQia2i58PcH918kSPMLrtwZ4kwK2V")

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
