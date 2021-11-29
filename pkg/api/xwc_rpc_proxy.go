// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"context"
	"encoding/json"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/rpc"
	"io/ioutil"
	"net/http"
	"time"
)

const RPC_TIME_OUT = 5 * time.Second

type xwcRequest struct {
	Id     uint64   `json:"id,omitempty"`
	Method string   `json:"method,omitempty"`
	Params []string `json:"params,omitempty"`
}

func (s *server) xwcRpcProxyHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || len(body) < 1 {
		if jsonhttp.HandleBodyReadError(err, w) {
			return
		}
		s.logger.Debugf("XWC RPC proxy: read request body error: %v", err)
		s.logger.Error("XWC RPC proxy: read request body error")
		jsonhttp.InternalServerError(w, "cannot read request")
		return
	}

	req := xwcRequest{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		s.logger.Debugf("XWC RPC proxy: unmarshal tag name error: %v", err)
		s.logger.Errorf("XWC RPC proxy: unmarshal tag name error")
		jsonhttp.InternalServerError(w, "error unmarshaling request")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), RPC_TIME_OUT)
	defer cancel()

	var result *rpc.JsonrpcMessage
	result, err = s.swapBackend.RawCall(ctx, req.Id, req)
	if err != nil {
		s.logger.Debugf("XWC RPC proxy: call rpc error: %v", err)
		s.logger.Errorf("XWC RPC proxy: call rpc error")
		jsonhttp.InternalServerError(w, err)
		return
	}

	s.logger.Debug("xwc request result", result.String())

	jsonhttp.OK(w, result)
}
