// Copyright 2021 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package node

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/statestore/leveldb"
	"github.com/penguintop/penguin/pkg/statestore/mock"
	"github.com/penguintop/penguin/pkg/storage"
    "github.com/penguintop/penguin/pkg/penguin"
)

// InitStateStore will initialize the stateStore with the given path to the
// data directory. When given an empty directory path, the function will instead
// initialize an in-memory state store that will not be persisted.
func InitStateStore(log logging.Logger, dataDir string) (ret storage.StateStorer, err error) {
	if dataDir == "" {
		ret = mock.NewStateStore()
		log.Warning("using in-mem state store, no node state will be persisted")
		return ret, nil
	}
	return leveldb.NewStateStore(filepath.Join(dataDir, "statestore"), log)
}

const overlayKey = "overlay"

// CheckOverlayWithStore checks the overlay is the same as stored in the statestore
func CheckOverlayWithStore(overlay penguin.Address, storer storage.StateStorer) error {
	var storedOverlay penguin.Address
	err := storer.Get(overlayKey, &storedOverlay)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		return storer.Put(overlayKey, overlay)
	}

	if !storedOverlay.Equal(overlay) {
		return fmt.Errorf("overlay address changed. was %s before but now is %s", storedOverlay, overlay)
	}
	return nil
}
