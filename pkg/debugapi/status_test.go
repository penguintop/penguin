// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package debugapi_test

import (
	pen "github.com/penguintop/penguin"
	"net/http"
	"testing"

	"github.com/penguintop/penguin/pkg/debugapi"
	"github.com/penguintop/penguin/pkg/jsonhttp/jsonhttptest"
)

func TestHealth(t *testing.T) {
	testServer := newTestServer(t, testServerOptions{})

	jsonhttptest.Request(t, testServer.Client, http.MethodGet, "/health", http.StatusOK,
		jsonhttptest.WithExpectedJSONResponse(debugapi.StatusResponse{
			Status:  "ok",
			Version: pen.Version,
		}),
	)
}

func TestReadiness(t *testing.T) {
	testServer := newTestServer(t, testServerOptions{})

	jsonhttptest.Request(t, testServer.Client, http.MethodGet, "/readiness", http.StatusOK,
		jsonhttptest.WithExpectedJSONResponse(debugapi.StatusResponse{
			Status:  "ok",
			Version: pen.Version,
		}),
	)
}
