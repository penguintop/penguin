package staking

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/swarm"
	"github.com/penguintop/penguin/pkg/transaction"
	"github.com/penguintop/penguin/pkg/xwcclient"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"github.com/penguintop/penguin/pkg/xwctypes"
	"math/big"
	"strings"
)

var (
	ErrInvalidStaking = errors.New("not a valid staking contract")
)

type Interface interface {
	QueryStaking(ctx context.Context) (bool, error)
	Staking(ctx context.Context) (bool, error)
}

type stakingContract struct {
	owner                  common.Address
	swarmNode              swarm.Address
	stakingContractAddress common.Address
	penTokenAddress        common.Address
	transactionService     transaction.Service
}

func (s *stakingContract) QueryStaking(ctx context.Context) (bool, error) {
	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "queryStaking"
	callData.CallArgs, _ = xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(s.owner[:]))

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return false, err
	}

	request := &transaction.TxRequest{
		To:       &s.stakingContractAddress,
		Data:     callDataBytes,
		GasPrice: nil,
		GasLimit: 0,
		Value:    big.NewInt(0),
	}

	data, err := s.transactionService.Call(ctx, request)
	if err != nil {
		return false, err
	}

	res := make(map[string]interface{})
	err = json.Unmarshal(data, &res)
	if err != nil {
		return false, err
	}

	if len(res) != 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (s *stakingContract) Staking(ctx context.Context) (bool, error) {
	approveAmount := big.NewInt(50000000 * property.PEN_ERC20_PRCISION)
	_, err := s.sendApproveTransaction(ctx, approveAmount)
	if err != nil {
		return false, err
	}
	_, err = s.sendStakingTransaction(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

func New(
	owner common.Address,
	swarmNode swarm.Address,
	stakingContractAddress common.Address,
	penTokenAddress common.Address,
	transactionService transaction.Service,
) Interface {
	return &stakingContract{
		owner:                  owner,
		swarmNode:              swarmNode,
		stakingContractAddress: stakingContractAddress,
		penTokenAddress:        penTokenAddress,
		transactionService:     transactionService,
	}
}

func (s *stakingContract) sendApproveTransaction(ctx context.Context, amount *big.Int) (*xwctypes.RpcTransactionReceipt, error) {
	stakingAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(s.stakingContractAddress[:]))
	amountStr := amount.String()

	txHash, err := s.transactionService.Send(ctx, &transaction.TxRequest{
		To: &s.penTokenAddress,
		//Data:     callData,
		GasPrice: big.NewInt(10),
		GasLimit: 100000,
		Value:    big.NewInt(0),

		TxType:     transaction.TxTypeInvokeContract,
		InvokeApi:  "approve",
		InvokeArgs: strings.Join([]string{stakingAddr, amountStr}, ","),
	})
	if err != nil {
		return nil, err
	}

	receipt, err := s.transactionService.WaitForReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}

	if receipt.ExecSucceed == false {
		return nil, transaction.ErrTransactionReverted
	}

	return receipt, nil
}

func (s *stakingContract) sendStakingTransaction(ctx context.Context) (*xwctypes.RpcTransactionReceipt, error) {
	request := &transaction.TxRequest{
		To:       &s.stakingContractAddress,
		GasPrice: big.NewInt(10),
		GasLimit: 100000,
		Value:    big.NewInt(0),

		TxType:     transaction.TxTypeInvokeContract,
		InvokeApi:  "Staking",
		InvokeArgs: s.swarmNode.String(),
	}

	txHash, err := s.transactionService.Send(ctx, request)
	if err != nil {
		return nil, err
	}

	receipt, err := s.transactionService.WaitForReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}

	if receipt.ExecSucceed == false {
		return nil, transaction.ErrTransactionReverted
	}

	return receipt, nil
}

func LookupERC20Address(ctx context.Context, transactionService transaction.Service, stakingContractAddress common.Address) (common.Address, error) {
	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "PenToken"
	callData.CallArgs = ""

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return common.Address{}, err
	}

	request := &transaction.TxRequest{
		To:       &stakingContractAddress,
		Data:     callDataBytes,
		GasPrice: nil,
		GasLimit: 0,
		Value:    big.NewInt(0),
	}

	data, err := transactionService.Call(ctx, request)
	if err != nil {
		return common.Address{}, err
	}

	addrHex, err := xwcfmt.XwcConAddrToHexAddr(string(data))
	if err != nil {
		return common.Address{}, err
	}
	addrBytes, _ := hex.DecodeString(addrHex)

	var addr common.Address
	addr.SetBytes(addrBytes)

	return addr, nil
}

func VerifyBytecode(ctx context.Context, backend *xwcclient.Client, stakingContract common.Address) error {
	code, err := backend.CodeAt(ctx, stakingContract, nil)
	if err != nil {
		return err
	}

	if !bytes.Equal(code, property.StakingAddressDeployedCodeHash) {
		return errors.New("verify byte code, invalid staking contract code hash")
	}

	return nil
}

func VerifyStakingAdmin(ctx context.Context, transactionService transaction.Service, stakingContractAddress common.Address) (bool, error) {
	type CallData struct {
		CallApi  string `json:"CallApi"`
		CallArgs string `json:"CallArgs"`
	}

	var callData CallData
	callData.CallApi = "admin"
	callData.CallArgs = ""

	callDataBytes, err := json.Marshal(callData)
	if err != nil {
		return false, err
	}

	request := &transaction.TxRequest{
		To:       &stakingContractAddress,
		Data:     callDataBytes,
		GasPrice: nil,
		GasLimit: 0,
		Value:    big.NewInt(0),
	}

	data, err := transactionService.Call(ctx, request)
	if err != nil {
		return false, err
	}

	if bytes.Compare(data, property.StakingAdmin) != 0 {
		return false, errors.New("verify staking admin, invalid staking contract admin")
	}

	return true, nil
}
