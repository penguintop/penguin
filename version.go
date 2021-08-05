// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pen

var (
	version = "0.0.1" // manually set semantic version number
	commit  string    // automatically set git commit hash

	Version = func() string {
		if commit != "" {
			return version + "-" + commit
		}
		return version + "-dev"
	}()
)
