package cmd

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/keystore"
	filekeystore "github.com/penguintop/penguin/pkg/keystore/file"
	memkeystore "github.com/penguintop/penguin/pkg/keystore/mem"
	"github.com/penguintop/penguin/pkg/property"
	"github.com/penguintop/penguin/pkg/xwcfmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"
)

func (c *command) initDumpKeyCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "dumpkey",
		Short: "Dump Penguin Private Wif Key",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) > 0 {
				return cmd.Help()
			}

			v := strings.ToLower(c.config.GetString(optionNameVerbosity))
			logger, err := newLogger(cmd, v)
			if err != nil {
				return fmt.Errorf("new logger: %v", err)
			}

			var keystore keystore.Service
			if c.config.GetString(optionNameDataDir) == "" {
				keystore = memkeystore.New()
				logger.Warning("data directory not provided, keys are not persisted")
				return nil
			} else {
				keystore = filekeystore.New(filepath.Join(c.config.GetString(optionNameDataDir), "keys"))
			}

			var password string
			if p := c.config.GetString(optionNamePassword); p != "" {
				password = p
			} else if pf := c.config.GetString(optionNamePasswordFile); pf != "" {
				b, err := ioutil.ReadFile(pf)
				if err != nil {
					return err
				}
				password = string(bytes.Trim(b, "\n"))
			} else {
				exists, err := keystore.Exists("penguin")
				if err != nil {
					return err
				}
				if exists {
					password, err = terminalPromptPassword(cmd, c.passwordReader, "Password")
					if err != nil {
						return err
					}

					penguinPrivateKey, _, err := keystore.Key("penguin", password)
					if err != nil {
						return fmt.Errorf("penguin key: %w", err)
					}

					tempBytes := penguinPrivateKey.D.Bytes()
					var privKeyBytes [32]byte
					copy(privKeyBytes[32-len(tempBytes):], tempBytes)
					privKeyWif, err := xwcfmt.HexKeyToWifKey(hex.EncodeToString(privKeyBytes[:]))
					if err != nil {
						return err
					}

					var key ecdsa.PrivateKey
					key.D = big.NewInt(0).SetBytes(privKeyBytes[:])
					signer := crypto.NewDefaultSigner(penguinPrivateKey)
					publicKey := &penguinPrivateKey.PublicKey

					nodeAddress, err := crypto.NewOverlayAddress(*publicKey, uint64(property.CHAIN_ID_NUM))
					if err != nil {
						return err
					}

					overlayXwcAddress, err := signer.XwcAddress()
					if err != nil {
						return err
					}
					xwcAddr, _ := xwcfmt.HexAddrToXwcAddr(hex.EncodeToString(overlayXwcAddress[:]))

					logger.Info("********************************************************************")
					logger.Infof("!!! PrivateKey: %s !!!", privKeyWif)
					logger.Infof("!!! Xwc Account Address: %s !!!", xwcAddr)
					logger.Infof("!!! Penguin Node Address: %s !!!", nodeAddress)
					logger.Infof("!!! Please backup your PrivateKey, and Do not tell it to anyone else !!!")
					logger.Info("********************************************************************")

				} else {
					return errors.New("penguin private key file not existed, maybe you do not set the option --data-dir")
				}
			}

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	c.setAllFlags(cmd)
	c.root.AddCommand(cmd)
	return nil
}
