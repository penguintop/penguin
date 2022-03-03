// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/sctx"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/penguintop/penguin/pkg/tags"
	"github.com/penguintop/penguin/pkg/tracing"
	"github.com/gorilla/mux"
)

type bytesPostResponse struct {
	Reference penguin.Address `json:"reference"`
}

// bytesUploadHandler handles upload of raw binary data of arbitrary length.
func (s *server) bytesUploadHandler(w http.ResponseWriter, r *http.Request) {
	logger := tracing.NewLoggerWithTraceID(r.Context(), s.logger)

	tag, created, err := s.getOrCreateTag(r.Header.Get(PenguinTagHeader))
	if err != nil {
		logger.Debugf("bytes upload: get or create tag: %v", err)
		logger.Error("bytes upload: get or create tag")
		jsonhttp.InternalServerError(w, "cannot get or create tag")
		return
	}

	if !created {
		// Only in the case when the tag is sent via header (i.e. not created by this request)
		if estimatedTotalChunks := requestCalculateNumberOfChunks(r); estimatedTotalChunks > 0 {
			err = tag.IncN(tags.TotalChunks, estimatedTotalChunks)
			if err != nil {
				s.logger.Debugf("bytes upload: increment tag: %v", err)
				s.logger.Error("bytes upload: increment tag")
				jsonhttp.InternalServerError(w, "increment tag")
				return
			}
		}
	}

	// Add the tag to the context
	ctx := sctx.SetTag(r.Context(), tag)

	batch, err := requestPostageBatchId(r)
	if err != nil {
		logger.Debugf("Bytes upload: postage batch id:%v", err)
		logger.Error("Bytes upload: postage batch id")
		jsonhttp.BadRequest(w, nil)
		return
	}

	putter, err := newStamperPutter(s.storer, s.post, s.signer, batch)
	if err != nil {
		logger.Debugf("Bytes upload: get putter:%v", err)
		logger.Error("Bytes upload: putter")
		jsonhttp.BadRequest(w, nil)
		return
	}

	p := requestPipelineFn(putter, r)
	address, err := p(ctx, r.Body)
	if err != nil {
		logger.Debugf("Bytes upload: split write all: %v", err)
		logger.Error("Bytes upload: split write all")
		jsonhttp.InternalServerError(w, nil)
		return
	}

	if created {
		_, err = tag.DoneSplit(address)
		if err != nil {
			logger.Debugf("Bytes upload: done split: %v", err)
			logger.Error("Bytes upload: done split failed")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}

	if strings.ToLower(r.Header.Get(PenguinPinHeader)) == "true" {
		if err := s.pinning.CreatePin(ctx, address, false); err != nil {
			logger.Debugf("Bytes upload: creation of pin for %q failed: %v", address, err)
			logger.Error("Bytes upload: creation of pin failed")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}

	w.Header().Set(PenguinTagHeader, fmt.Sprint(tag.Uid))
	w.Header().Set("Access-Control-Expose-Headers", PenguinTagHeader)
	jsonhttp.Created(w, bytesPostResponse{
		Reference: address,
	})
}

// bytesGetHandler handles retrieval of raw binary data of arbitrary length.
func (s *server) bytesGetHandler(w http.ResponseWriter, r *http.Request) {
	logger := tracing.NewLoggerWithTraceID(r.Context(), s.logger).Logger
	nameOrHex := mux.Vars(r)["address"]

	address, err := s.resolveNameOrAddress(nameOrHex)
	if err != nil {
		logger.Debugf("bytes: parse address %s: %v", nameOrHex, err)
		logger.Error("bytes: parse address error")
		jsonhttp.NotFound(w, nil)
		return
	}

	additionalHeaders := http.Header{
		"Content-Type": {"application/octet-stream"},
	}

	s.downloadHandler(w, r, address, additionalHeaders, true)
}
