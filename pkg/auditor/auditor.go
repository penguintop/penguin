package auditor

import (
	"encoding/hex"
	"fmt"
	"github.com/penguintop/penguin/pkg/cac"
	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/localstore"
	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math"
	"time"
)

const (
	WAIT_SECONDS        = 5 * 60
	AUDITOR_RPC_TIMEOUT = 10
)

type Auditor struct {
	logger logging.Logger
	//
	AuditEndpoint string
	// local store db
	LocalDB *localstore.DB

	// node address
	PenguinAddress string
	// signer
	Signer crypto.Signer
	// signer pubkey (hex)
	SignerPubKey string
	// payer xwc address
	XwcAcctAddress string
}

func CreateNewAuditor(endpoint string, localDB *localstore.DB, signer crypto.Signer, logger logging.Logger) *Auditor {
	r := new(Auditor)
	r.AuditEndpoint = endpoint
	r.LocalDB = localDB
	r.Signer = signer
	r.logger = logger

	r.SignerPubKey, _ = signer.CompressedPubKeyHex()

	publicKey, _ := signer.PublicKey()
	penguinAddr, _ := crypto.NewOverlayAddress(*publicKey, uint64(property.CHAIN_ID_NUM))
	r.PenguinAddress = penguinAddr.String()

	xwcAcctAddr, _ := signer.XwcAddress()
	r.XwcAcctAddress, _ = xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(xwcAcctAddr[:]))

	return r
}

func (r *Auditor) Run() {
	for {
		time.Sleep(WAIT_SECONDS * time.Second)
		r.logger.Infof("start audit at %s", time.Now().String())

		// dump retrieval keys
		retrievalAddresses, err := r.LocalDB.DumpAllRetrievalKeys()
		if err != nil {
			r.logger.Errorf("DumpAllRetrievalKeys failed: %s", err.Error())
			continue
		}
		if len(retrievalAddresses) == 0 {
			r.logger.Warning("empty retrievalAddresses")
			continue
		}

		paddingCnt := paddingCount(uint64(len(retrievalAddresses)))
		retrievalAddresses = append(retrievalAddresses, retrievalAddresses[0:paddingCnt]...)
		r.logger.Infof("After padding %d item, now retrievalAddresses size: %d", paddingCnt, len(retrievalAddresses))

		treeDepth := int(math.Log2(float64(len(retrievalAddresses))))
		r.logger.Infof("Your Contribution Weight is %d", treeDepth)

		// build full binary tree
		treeRootNode, err := BuildBTreeFromRetrievalAddresses(retrievalAddresses)
		if err != nil {
			r.logger.Errorf("BuildBTreeFromRetrievalAddresses: %s", err.Error())
			continue
		}

		// 1st step, get server timestamp, and calc timestamp diff
		serverTimestamp, err := RequestServerTimestamp(r.AuditEndpoint, AUDITOR_RPC_TIMEOUT)
		if err != nil {
			r.logger.Errorf("RequestServerTimestamp: %s", err.Error())
			continue
		}
		nodeTimestamp := time.Now().Unix()
		secondDiff := serverTimestamp - nodeTimestamp
		r.logger.Infof("server timestamp: %d", serverTimestamp)
		r.logger.Infof("time diff: %d seconds", secondDiff)

		// 2st step, get task
		adjustTimestamp := time.Now().Unix() + secondDiff
		adjustTimestampStr := fmt.Sprintf("%d", adjustTimestamp)
		signature, err := r.Signer.SignForAudit([]byte(adjustTimestampStr))
		if err != nil {
			r.logger.Errorf("SignForAudit: %s", err.Error())
			continue
		}

		taskId, err := RequestTask(r.AuditEndpoint, AUDITOR_RPC_TIMEOUT, adjustTimestamp, r.XwcAcctAddress, r.SignerPubKey, r.PenguinAddress, hex.EncodeToString(signature))
		if err != nil {
			r.logger.Errorf("RequestTask: %s", err.Error())
			continue
		}

		r.logger.Infof("RequestTask task id: %d", taskId)
		// 3rd step, report merkle root
		rootHashHex, nextHashHexPair := treeRootNode.GetRootRelatedHashHex()
		pathData := make([][]string, 0)
		pathData = append(pathData, []string{rootHashHex})
		if !(nextHashHexPair == nil || len(nextHashHexPair) == 0) {
			pathData = append(pathData, nextHashHexPair)
		}

		r.logger.Infof("root hash: %s", rootHashHex)
		if nextHashHexPair != nil && len(nextHashHexPair) == 2 {
			r.logger.Infof("root left son hash: %s", nextHashHexPair[0])
			r.logger.Infof("root right son hash: %s", nextHashHexPair[1])
		}

		taskIdStr := fmt.Sprintf("%d", taskId)
		signature, err = r.Signer.SignForAudit([]byte(taskIdStr))
		if err != nil {
			r.logger.Errorf("SignForAudit: %s", err.Error())
			continue
		}
		taskId, pathInt, err := RequestReportMerkleRoot(r.AuditEndpoint, AUDITOR_RPC_TIMEOUT, taskId, r.XwcAcctAddress, r.SignerPubKey, r.PenguinAddress, hex.EncodeToString(signature), pathData)
		if err != nil {
			r.logger.Errorf("RequestReportMerkleRoot: %s", err.Error())
			continue
		}
		r.logger.Infof("RequestReportMerkleRoot task id: %d, path int: %d", taskId, pathInt)

		// 4th step, report path way data
		rootHashHex, pathWayHexPairList, pathWayFinalNodeHashHex := treeRootNode.GetPathWayHashHex(pathInt)
		pathData = make([][]string, 0)
		pathData = append(pathData, []string{rootHashHex})
		if !(nextHashHexPair == nil || len(nextHashHexPair) == 0) {
			pathData = append(pathData, pathWayHexPairList...)
		}

		r.logger.Infof("Root Hash: %s", rootHashHex)
		for _, pathWayHexPair := range pathWayHexPairList {
			r.logger.Infof("L Son Hash: %s", pathWayHexPair[0])
			r.logger.Infof("R son hash: %s", pathWayHexPair[1])
		}
		r.logger.Infof("Final Node Hash: %s", pathWayFinalNodeHashHex)

		pathWayFinalNodeHash, err := hex.DecodeString(pathWayFinalNodeHashHex)
		if err != nil {
			r.logger.Errorf("hex.DecodeString: %s", err.Error())
			continue
		}

		item, err := r.LocalDB.GetRetrievalData(pathWayFinalNodeHash)
		if err != nil {
			r.logger.Errorf("GetRetrievalData: %s", err.Error())
			continue
		}
		// calc item info by data, and verify it
		chunk, err := cac.NewWithDataSpan(item.Data)
		if chunk.Address().String() != pathWayFinalNodeHashHex {
			r.logger.Errorf("chunk.Address().String() != pathWayFinalNodeHashHex")
			continue
		}

		taskIdStr = fmt.Sprintf("%d", taskId)
		signature, err = r.Signer.SignForAudit([]byte(taskIdStr))
		if err != nil {
			r.logger.Errorf("SignForAudit: %s", err.Error())
			continue
		}
		err = RequestReportPathData(r.AuditEndpoint, AUDITOR_RPC_TIMEOUT, taskId, r.XwcAcctAddress, r.SignerPubKey, r.PenguinAddress, hex.EncodeToString(signature), pathData,
			hex.EncodeToString(item.Data))
		if err != nil {
			r.logger.Errorf("RequestReportPathData: %s", err.Error())
			continue
		}

		// done
		r.logger.Infof("audit end at %s", time.Now().String())
	}
}

func paddingCount(val uint64) uint64 {
	s := uint64(1)
	for {
		if s >= val {
			return s - val
		} else {
			s = s << 1
		}
	}
}
