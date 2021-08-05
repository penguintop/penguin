// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package breaker

import "time"

func SetTimeNow(f func() time.Time) {
	timeNow = f
}
