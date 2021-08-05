// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package multiresolver

import "github.com/penguintop/penguin/pkg/logging"

func GetLogger(mr *MultiResolver) logging.Logger {
	return mr.logger
}

func GetCfgs(mr *MultiResolver) []ConnectionConfig {
	return mr.cfgs
}
