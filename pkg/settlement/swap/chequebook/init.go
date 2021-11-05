// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chequebook

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/settlement/swap/erc20"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/transaction"
)

const (
	chequebookKey           = "swap_chequebook"
	ChequebookDeploymentKey = "swap_chequebook_transaction_deployment"

	balanceCheckBackoffDuration = 20 * time.Second
	balanceCheckMaxRetries      = 10
	CHECK_BALANCE_INTERVAL      = 60 * time.Second

	chequeBookDeployedCheckBackoffDuration = 10 * time.Second
)

func checkBalance(
	ctx context.Context,
	logger logging.Logger,
	swapInitialDeposit *big.Int,
	swapBackend transaction.Backend,
	chainId int64,
	overlayXwcAddress common.Address,
	erc20Token erc20.Service,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, balanceCheckBackoffDuration*time.Duration(balanceCheckMaxRetries))
	defer cancel()
	for {
		erc20Balance, err := erc20Token.BalanceOf(timeoutCtx, overlayXwcAddress)
		if err != nil {
			return err
		}
		logger.Info("pen token balance:", erc20Balance)

		xwcBalance, err := swapBackend.BalanceAt(timeoutCtx, overlayXwcAddress, nil)
		if err != nil {
			return err
		}
		logger.Info("xwc balance:", xwcBalance)

		minimumXwc := big.NewInt(1).Mul(big.NewInt(1), big.NewInt(property.XWC_ASSET_PRCISION))

		insufficientERC20 := erc20Balance.Cmp(swapInitialDeposit) < 0
		insufficientXWC := xwcBalance.Cmp(minimumXwc) < 0

		if insufficientERC20 || insufficientXWC {
			neededERC20, mod := new(big.Int).DivMod(swapInitialDeposit, big.NewInt(property.PEN_ERC20_PRCISION), new(big.Int))
			if mod.Cmp(big.NewInt(0)) > 0 {
				// always round up the division as the bzzaar cannot handle decimals
				neededERC20.Add(neededERC20, big.NewInt(1))
			}

			overlayXwcAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(overlayXwcAddress[:]))

			if insufficientXWC && insufficientERC20 {
				logger.Warningf("cannot continue until there is sufficient XWC (for Gas) and at least %d PEN available on %s", neededERC20, overlayXwcAddr)
			} else if insufficientXWC {
				logger.Warningf("cannot continue until there is sufficient XWC (for Gas) available on %s", overlayXwcAddr)
			} else {
				logger.Warningf("cannot continue until there is at least %d PEN available on %s", neededERC20, overlayXwcAddr)
			}

			select {
			case <-time.After(balanceCheckBackoffDuration):
			case <-timeoutCtx.Done():
				if insufficientERC20 {
					return fmt.Errorf("insufficient PEN for initial deposit")
				} else {
					return fmt.Errorf("insufficient ETH for initial deposit")
				}
			}
			continue
		}

		return nil
	}
}

// Function Init will initialize the chequebook service.
func Init(
	ctx context.Context,
	chequebookFactory Factory,
	stateStore storage.StateStorer,
	logger logging.Logger,
	swapInitialDeposit *big.Int,
	transactionService transaction.Service,
	swapBackend transaction.Backend,
	chainId int64,
	overlayXwcAddress common.Address,
	chequeSigner ChequeSigner,
) (chequebookService Service, err error) {
	// verify that the supplied factory is valid
	err = chequebookFactory.VerifyBytecode(ctx)
	if err != nil {
		return nil, err
	}

	erc20Address, err := chequebookFactory.ERC20Address(ctx)
	if err != nil {
		return nil, err
	}

	conAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(erc20Address[:]))
	logger.Info("pen token contract: ", conAddr)

	erc20Service := erc20.New(swapBackend, transactionService, erc20Address)

	/////////////////////////////////////////////////////////////
	//var chequebookAddress common.Address
	//err = stateStore.Get(chequebookKey, &chequebookAddress)
	//if err != nil {
	//	if err != storage.ErrNotFound {
	//		return nil, err
	//	}
	//
	//	var txHash common.Hash
	//	err = stateStore.Get(ChequebookDeploymentKey, &txHash)
	//	if err != nil && err != storage.ErrNotFound {
	//		return nil, err
	//	}
	//	if err == storage.ErrNotFound {
	//		logger.Info("no chequebook found, deploying new one.")
	//		if swapInitialDeposit.Cmp(big.NewInt(0)) != 0 {
	//			err = checkBalance(ctx, logger, swapInitialDeposit, swapBackend, chainId, overlayXwcAddress, erc20Service)
	//			if err != nil {
	//				return nil, err
	//			}
	//		}
	//
	//		nonce := make([]byte, 32)
	//		_, err = rand.Read(nonce)
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		// if we don't yet have a chequebook, deploy a new one
	//		txHash, err = chequebookFactory.Deploy(ctx, overlayXwcAddress, big.NewInt(0), common.BytesToHash(nonce))
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		logger.Infof("deploying new chequebook in transaction %x", txHash)
	//
	//		err = stateStore.Put(ChequebookDeploymentKey, txHash)
	//		if err != nil {
	//			return nil, err
	//		}
	//	} else {
	//		logger.Infof("waiting for chequebook deployment in transaction %x", txHash)
	//	}
	//
	//	chequebookAddress, err = chequebookFactory.WaitDeployed(ctx, txHash)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	logger.Infof("deployed chequebook at address %x", chequebookAddress)
	//
	//	// save the address for later use
	//	err = stateStore.Put(chequebookKey, chequebookAddress)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	chequebookService, err = New(transactionService, chequebookAddress, overlayXwcAddress, stateStore, chequeSigner, erc20Service)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	if swapInitialDeposit.Cmp(big.NewInt(0)) != 0 {
	//		logger.Infof("depositing %d token into new chequebook", swapInitialDeposit)
	//		depositHash, err := chequebookService.Deposit(ctx, swapInitialDeposit)
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		logger.Infof("sent deposit transaction %x", depositHash)
	//		err = chequebookService.WaitForDeposit(ctx, depositHash)
	//		if err != nil {
	//			return nil, err
	//		}
	//
	//		logger.Info("successfully deposited to chequebook")
	//	}
	//} else {
	//	chequebookService, err = New(transactionService, chequebookAddress, overlayXwcAddress, stateStore, chequeSigner, erc20Service)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	logger.Infof("using existing chequebook %x", chequebookAddress)
	//}
	//
	//// regardless of how the chequebook service was initialized make sure that the chequebook is valid
	//err = chequebookFactory.VerifyChequebook(ctx, chequebookService.Address())
	//if err != nil {
	//	return nil, err
	//}
	///////////////////////////////////////////////////////////////

	p, err := chequebookFactory.QueryUserChequeBook(ctx, overlayXwcAddress)
	if err != nil {
		return nil, err
	}

	if p == nil {
		// chequebook contract not exist
		var txHash common.Hash
		err = stateStore.Get(ChequebookDeploymentKey, &txHash)
		if err != nil && err != storage.ErrNotFound {
			return nil, err
		}

		if err == storage.ErrNotFound {
			logger.Info("no chequebook found, deploying new one.")
			if swapInitialDeposit.Cmp(big.NewInt(0)) != 0 {
				for {
					err = checkBalance(ctx, logger, swapInitialDeposit, swapBackend, chainId, overlayXwcAddress, erc20Service)
					if err != nil {
						// Never skip the process and retry it until the account balance is available.
						logger.Errorf("Failed to check balance: %s", err.Error())
						logger.Info("Wait a moment to retry the checking balance.")

						time.Sleep(CHECK_BALANCE_INTERVAL)
						continue
					}

					break
				}

				// transfer token
				EntranceAddressHex, _ := xwcfmt.XwcAddrToHexAddr(property.EntranceAddress)
				EntranceAddressBytes, _ := hex.DecodeString(EntranceAddressHex)
				var toAddr common.Address
				toAddr.SetBytes(EntranceAddressBytes)

				txHash, err = erc20Service.Transfer(ctx, toAddr, swapInitialDeposit)
				if err != nil {
					return nil, err
				}
				logger.Infof("deploying new chequebook in transaction %x", txHash[12:])

				err = stateStore.Put(ChequebookDeploymentKey, txHash)
				if err != nil {
					return nil, err
				}
			}
		} else {
			logger.Infof("waiting for chequebook deployment in transaction %x", txHash[12:])
		}

		// wait deploy
		for {
			p, err = chequebookFactory.QueryUserChequeBook(ctx, overlayXwcAddress)
			if err != nil {
				return nil, err
			}

			if p == nil {
				logger.Infof("waiting for cheque book contract deployed...")

				// wait 10 seconds
				select {
				case <-time.After(chequeBookDeployedCheckBackoffDuration):
					continue
				}
			} else {
				chequebookAddress := *p
				chequebookService, err = New(transactionService, chequebookAddress, overlayXwcAddress, stateStore, chequeSigner, erc20Service)
				if err != nil {
					return nil, err
				}

				chequebookXwcAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(chequebookAddress[:]))
				logger.Infof("chequebook deployed %s", chequebookXwcAddr)
				break
			}
		}

	} else {
		// chequebook contract exist
		chequebookAddress := *p
		chequebookService, err = New(transactionService, chequebookAddress, overlayXwcAddress, stateStore, chequeSigner, erc20Service)
		if err != nil {
			return nil, err
		}

		chequebookXwcAddr, _ := xwcfmt.HexAddrToXwcConAddr(hex.EncodeToString(chequebookAddress[:]))
		logger.Infof("using existing chequebook %s", chequebookXwcAddr)
	}

	// regardless of how the chequebook service was initialized make sure that the chequebook is valid
	err = chequebookFactory.VerifyChequebook(ctx, chequebookService.Address())
	if err != nil {
		return nil, err
	}

	err = chequebookFactory.VerifyChequebookOwner(ctx, chequebookService.Address(), overlayXwcAddress)
	if err != nil {
		return nil, err
	}

	return chequebookService, nil
}
