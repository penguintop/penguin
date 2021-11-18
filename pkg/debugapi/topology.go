// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package debugapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/penguintop/penguin/pkg/jsonhttp"
)

func (s *Service) topologyHandler(w http.ResponseWriter, r *http.Request) {
	params := s.topologyDriver.Snapshot()

	params.LightNodes = s.lightNodes.PeerInfo()

	b, err := json.Marshal(params)
	if err != nil {
		s.logger.Errorf("Topology marshal to json: %v", err)
		jsonhttp.InternalServerError(w, err)
		return
	}
	w.Header().Set("Content-Type", jsonhttp.DefaultContentTypeHeader)
	_, _ = io.Copy(w, bytes.NewBuffer(b))
}
