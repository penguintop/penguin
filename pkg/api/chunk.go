// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/penguintop/penguin/pkg/cac"
	"github.com/penguintop/penguin/pkg/netstore"

	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/sctx"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/penguintop/penguin/pkg/tags"
	"github.com/gorilla/mux"
)

type chunkAddressResponse struct {
	Reference penguin.Address `json:"reference"`
}

func (s *server) chunkUploadHandler(w http.ResponseWriter, r *http.Request) {
	var (
		tag *tags.Tag
		ctx = r.Context()
		err error
	)

	if h := r.Header.Get(PenguinTagHeader); h != "" {
		tag, err = s.getTag(h)
		if err != nil {
			s.logger.Debugf("Chunk upload: get tag: %v", err)
			s.logger.Error("Chunk upload: get tag")
			jsonhttp.BadRequest(w, "cannot get tag")
			return

		}

		// Add the tag to the context if it exists
		ctx = sctx.SetTag(r.Context(), tag)

		// Increase the StateSplit here since we dont have a splitter for the file upload
		err = tag.Inc(tags.StateSplit)
		if err != nil {
			s.logger.Debugf("Chunk upload: increment tag: %v", err)
			s.logger.Error("Chunk upload: increment tag")
			jsonhttp.InternalServerError(w, "increment tag")
			return
		}
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if jsonhttp.HandleBodyReadError(err, w) {
			return
		}
		s.logger.Debugf("Chunk upload: read chunk data error: %v", err)
		s.logger.Error("Chunk upload: read chunk data error")
		jsonhttp.InternalServerError(w, "cannot read chunk data")
		return
	}

	if len(data) < penguin.SpanSize {
		s.logger.Debug("Chunk upload: not enough data")
		s.logger.Error("Chunk upload: data length")
		jsonhttp.BadRequest(w, "data length")
		return
	}

	chunk, err := cac.NewWithDataSpan(data)
	if err != nil {
		s.logger.Debugf("Chunk upload: create chunk error: %v", err)
		s.logger.Error("Chunk upload: create chunk error")
		jsonhttp.InternalServerError(w, "create chunk error")
		return
	}

	batch, err := requestPostageBatchId(r)
	if err != nil {
		s.logger.Debugf("Chunk upload: postage batch id: %v", err)
		s.logger.Error("Chunk upload: postage batch id")
		jsonhttp.BadRequest(w, "invalid postage batch id")
		return
	}

	putter, err := newStamperPutter(s.storer, s.post, s.signer, batch)
	if err != nil {
		s.logger.Debugf("Chunk upload: putter:%v", err)
		s.logger.Error("Chunk upload: putter")
		jsonhttp.BadRequest(w, nil)
		return
	}

	seen, err := putter.Put(ctx, requestModePut(r), chunk)
	if err != nil {
		s.logger.Debugf("Chunk upload: chunk write error: %v, addr %s", err, chunk.Address())
		s.logger.Error("Chunk upload: chunk write error")
		jsonhttp.BadRequest(w, "chunk write error")
		return
	} else if len(seen) > 0 && seen[0] && tag != nil {
		err := tag.Inc(tags.StateSeen)
		if err != nil {
			s.logger.Debugf("Chunk upload: increment tag", err)
			s.logger.Error("Chunk upload: increment tag")
			jsonhttp.BadRequest(w, "increment tag")
			return
		}
	}

	if tag != nil {
		// Indicate that the chunk is stored
		err = tag.Inc(tags.StateStored)
		if err != nil {
			s.logger.Debugf("Chunk upload: increment tag", err)
			s.logger.Error("Chunk upload: increment tag")
			jsonhttp.InternalServerError(w, "increment tag")
			return
		}
		w.Header().Set(PenguinTagHeader, fmt.Sprint(tag.Uid))
	}

	if strings.ToLower(r.Header.Get(PenguinPinHeader)) == "true" {
		if err := s.pinning.CreatePin(ctx, chunk.Address(), false); err != nil {
			s.logger.Debugf("Chunk upload: creation of pin for %q failed: %v", chunk.Address(), err)
			s.logger.Error("Chunk upload: creation of pin failed")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}

	w.Header().Set("Access-Control-Expose-Headers", PenguinTagHeader)
	jsonhttp.Created(w, chunkAddressResponse{Reference: chunk.Address()})
}

func (s *server) chunkGetHandler(w http.ResponseWriter, r *http.Request) {
	targets := r.URL.Query().Get("targets")
	if targets != "" {
		r = r.WithContext(sctx.SetTargets(r.Context(), targets))
	}

	nameOrHex := mux.Vars(r)["addr"]
	ctx := r.Context()

	address, err := s.resolveNameOrAddress(nameOrHex)
	if err != nil {
		s.logger.Debugf("Chunk: parse chunk address %s: %v", nameOrHex, err)
		s.logger.Error("Chunk: parse chunk address error")
		jsonhttp.NotFound(w, nil)
		return
	}

	chunk, err := s.storer.Get(ctx, storage.ModeGetRequest, address)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			s.logger.Tracef("Chunk: chunk not found. addr %s", address)
			jsonhttp.NotFound(w, "chunk not found")
			return

		}
		if errors.Is(err, netstore.ErrRecoveryAttempt) {
			s.logger.Tracef("Chunk: chunk recovery initiated. addr %s", address)
			jsonhttp.Accepted(w, "chunk recovery initiated. retry after sometime.")
			return
		}
		s.logger.Debugf("Chunk: chunk read error: %v ,addr %s", err, address)
		s.logger.Error("Chunk: chunk read error")
		jsonhttp.InternalServerError(w, "chunk read error")
		return
	}
	w.Header().Set("Content-Type", "binary/octet-stream")
	if targets != "" {
		w.Header().Set(TargetsRecoveryHeader, targets)
	}
	_, _ = io.Copy(w, bytes.NewReader(chunk.Data()))
}
