// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p_test

import (
	"testing"

	"github.com/penguintop/penguin/pkg/p2p"
)

func TestNewPenguinStreamName(t *testing.T) {
	want := "/penguin/hive/1.2.0/peers"
	got := p2p.NewPenguinStreamName("hive", "1.2.0", "peers")

	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
