// Copyright 2021 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/penguintop/penguin/pkg/api"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/jsonhttp/jsonhttptest"
	"github.com/penguintop/penguin/pkg/logging"
	pinning "github.com/penguintop/penguin/pkg/pinning/mock"
	mockpost "github.com/penguintop/penguin/pkg/postage/mock"
	statestore "github.com/penguintop/penguin/pkg/statestore/mock"
	"github.com/penguintop/penguin/pkg/storage/mock"
	testingc "github.com/penguintop/penguin/pkg/storage/testing"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/penguintop/penguin/pkg/tags"
	"github.com/penguintop/penguin/pkg/traversal"
)

func checkPinHandlers(t *testing.T, client *http.Client, rootHash string) {
	t.Helper()

	const pinsBasePath = "/pins"

	var (
		pinsReferencePath        = pinsBasePath + "/" + rootHash
		pinsInvalidReferencePath = pinsBasePath + "/" + "838d0a193ecd1152d1bb1432d5ecc02398533b2494889e23b8bd5ace30ac2zzz"
		pinsUnknownReferencePath = pinsBasePath + "/" + "838d0a193ecd1152d1bb1432d5ecc02398533b2494889e23b8bd5ace30ac2ccc"
	)

	jsonhttptest.Request(t, client, http.MethodGet, pinsInvalidReferencePath, http.StatusBadRequest)

	jsonhttptest.Request(t, client, http.MethodGet, pinsUnknownReferencePath, http.StatusNotFound,
		jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
			Message: http.StatusText(http.StatusNotFound),
			Code:    http.StatusNotFound,
		}),
	)

	jsonhttptest.Request(t, client, http.MethodPost, pinsReferencePath, http.StatusCreated,
		jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
			Message: http.StatusText(http.StatusCreated),
			Code:    http.StatusCreated,
		}),
	)

	jsonhttptest.Request(t, client, http.MethodGet, pinsReferencePath, http.StatusOK,
		jsonhttptest.WithExpectedJSONResponse(struct {
			Reference penguin.Address `json:"reference"`
		}{
			Reference: penguin.MustParseHexAddress(rootHash),
		}),
	)

	jsonhttptest.Request(t, client, http.MethodGet, pinsBasePath, http.StatusOK,
		jsonhttptest.WithExpectedJSONResponse(struct {
			References []penguin.Address `json:"references"`
		}{
			References: []penguin.Address{penguin.MustParseHexAddress(rootHash)},
		}),
	)

	jsonhttptest.Request(t, client, http.MethodDelete, pinsReferencePath, http.StatusOK)

	jsonhttptest.Request(t, client, http.MethodGet, pinsReferencePath, http.StatusNotFound,
		jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
			Message: http.StatusText(http.StatusNotFound),
			Code:    http.StatusNotFound,
		}),
	)
}

func TestPinHandlers(t *testing.T) {
	var (
		storerMock   = mock.NewStorer()
		client, _, _ = newTestServer(t, testServerOptions{
			Storer:    storerMock,
			Traversal: traversal.New(storerMock),
			Tags:      tags.NewTags(statestore.NewStateStore(), logging.New(ioutil.Discard, 0)),
			Pinning:   pinning.NewServiceMock(),
			Logger:    logging.New(ioutil.Discard, 5),
			Post:      mockpost.New(mockpost.WithAcceptAll()),
		})
	)

	t.Run("bytes", func(t *testing.T) {
		const rootHash = "838d0a193ecd1152d1bb1432d5ecc02398533b2494889e23b8bd5ace30ac2aeb"
		jsonhttptest.Request(t, client, http.MethodPost, "/bytes", http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.PenguinPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(strings.NewReader("this is a simple text")),
			jsonhttptest.WithExpectedJSONResponse(api.PenUploadResponse{
				Reference: penguin.MustParseHexAddress(rootHash),
			}),
		)
		checkPinHandlers(t, client, rootHash)
	})

	t.Run("pen", func(t *testing.T) {
		tarReader := tarFiles(t, []f{{
			data: []byte("<h1>Penguin"),
			name: "index.html",
			dir:  "",
		}})
		rootHash := "9e178dbd1ed4b748379e25144e28dfb29c07a4b5114896ef454480115a56b237"
		jsonhttptest.Request(t, client, http.MethodPost, "/pen", http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.PenguinPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(tarReader),
			jsonhttptest.WithRequestHeader("Content-Type", api.ContentTypeTar),
			jsonhttptest.WithRequestHeader(api.PenguinCollectionHeader, "True"),
			jsonhttptest.WithExpectedJSONResponse(api.PenUploadResponse{
				Reference: penguin.MustParseHexAddress(rootHash),
			}),
		)
		checkPinHandlers(t, client, rootHash)

		rootHash = "dd13a5a6cc9db3ef514d645e6719178dbfb1a90b49b9262cafce35b0d27cf245"
		jsonhttptest.Request(t, client, http.MethodPost, "/pen?name=somefile.txt", http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.PenguinPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestHeader("Content-Type", "text/plain"),
			jsonhttptest.WithRequestBody(strings.NewReader("this is a simple text")),
			jsonhttptest.WithExpectedJSONResponse(api.PenUploadResponse{
				Reference: penguin.MustParseHexAddress(rootHash),
			}),
		)
		checkPinHandlers(t, client, rootHash)
	})

	t.Run("chunk", func(t *testing.T) {
		var (
			chunk    = testingc.GenerateTestRandomChunk()
			rootHash = chunk.Address().String()
		)
		jsonhttptest.Request(t, client, http.MethodPost, "/chunks", http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.PenguinPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(bytes.NewReader(chunk.Data())),
			jsonhttptest.WithExpectedJSONResponse(api.ChunkAddressResponse{
				Reference: chunk.Address(),
			}),
		)
		checkPinHandlers(t, client, rootHash)
	})
}
