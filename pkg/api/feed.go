// Copyright 2021 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/penguintop/penguin/pkg/feeds"
	"github.com/penguintop/penguin/pkg/file/loadsave"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/manifest"
	"github.com/penguintop/penguin/pkg/soc"
    "github.com/penguintop/penguin/pkg/penguin"
	"github.com/gorilla/mux"
)

const (
	feedMetadataEntryOwner = "penguin-feed-owner"
	feedMetadataEntryTopic = "penguin-feed-topic"
	feedMetadataEntryType  = "penguin-feed-type"
)

var errInvalidFeedUpdate = errors.New("invalid feed update")

type feedReferenceResponse struct {
	Reference penguin.Address `json:"reference"`
}

func (s *server) feedGetHandler(w http.ResponseWriter, r *http.Request) {
	owner, err := hex.DecodeString(mux.Vars(r)["owner"])
	if err != nil {
		s.logger.Debugf("feed get: decode owner: %v", err)
		s.logger.Error("feed get: bad owner")
		jsonhttp.BadRequest(w, "bad owner")
		return
	}

	topic, err := hex.DecodeString(mux.Vars(r)["topic"])
	if err != nil {
		s.logger.Debugf("feed get: decode topic: %v", err)
		s.logger.Error("feed get: bad topic")
		jsonhttp.BadRequest(w, "bad topic")
		return
	}

	var at int64
	atStr := r.URL.Query().Get("at")
	if atStr != "" {
		at, err = strconv.ParseInt(atStr, 10, 64)
		if err != nil {
			s.logger.Debugf("feed get: decode at: %v", err)
			s.logger.Error("feed get: bad at")
			jsonhttp.BadRequest(w, "bad at")
			return
		}
	} else {
		at = time.Now().Unix()
	}

	f := feeds.New(topic, common.BytesToAddress(owner))
	lookup, err := s.feedFactory.NewLookup(feeds.Sequence, f)
	if err != nil {
		s.logger.Debugf("feed get: new lookup: %v", err)
		s.logger.Error("feed get: new lookup")
		jsonhttp.InternalServerError(w, "new lookup")
		return
	}

	ch, cur, next, err := lookup.At(r.Context(), at, 0)
	if err != nil {
		s.logger.Debugf("feed get: lookup: %v", err)
		s.logger.Error("feed get: lookup error")
		jsonhttp.NotFound(w, "lookup failed")
		return
	}

	// KLUDGE: if a feed was never updated, the chunk will be nil
	if ch == nil {
		s.logger.Debugf("feed get: no update found: %v", err)
		s.logger.Error("feed get: no update found")
		jsonhttp.NotFound(w, "lookup failed")
		return
	}

	ref, _, err := parseFeedUpdate(ch)
	if err != nil {
		s.logger.Debugf("feed get: parse update: %v", err)
		s.logger.Error("feed get: parse update")
		jsonhttp.InternalServerError(w, "parse update")
		return
	}

	curBytes, err := cur.MarshalBinary()
	if err != nil {
		s.logger.Debugf("feed get: marshal current index: %v", err)
		s.logger.Error("feed get: marshal index")
		jsonhttp.InternalServerError(w, "marshal index")
		return
	}

	nextBytes, err := next.MarshalBinary()
	if err != nil {
		s.logger.Debugf("feed get: marshal next index: %v", err)
		s.logger.Error("feed get: marshal index")
		jsonhttp.InternalServerError(w, "marshal index")
		return
	}

	w.Header().Set(PenguinFeedIndexHeader, hex.EncodeToString(curBytes))
	w.Header().Set(PenguinFeedIndexNextHeader, hex.EncodeToString(nextBytes))
	w.Header().Set("Access-Control-Expose-Headers", fmt.Sprintf("%s, %s", PenguinFeedIndexHeader, PenguinFeedIndexNextHeader))

	jsonhttp.OK(w, feedReferenceResponse{Reference: ref})
}

func (s *server) feedPostHandler(w http.ResponseWriter, r *http.Request) {
	owner, err := hex.DecodeString(mux.Vars(r)["owner"])
	if err != nil {
		s.logger.Debugf("feed put: decode owner: %v", err)
		s.logger.Error("feed put: bad owner")
		jsonhttp.BadRequest(w, "bad owner")
		return
	}

	topic, err := hex.DecodeString(mux.Vars(r)["topic"])
	if err != nil {
		s.logger.Debugf("feed put: decode topic: %v", err)
		s.logger.Error("feed put: bad topic")
		jsonhttp.BadRequest(w, "bad topic")
		return
	}

	batch, err := requestPostageBatchId(r)
	if err != nil {
		s.logger.Debugf("feed put: postage batch id: %v", err)
		s.logger.Error("feed put: postage batch id")
		jsonhttp.BadRequest(w, "invalid postage batch id")
		return
	}

	putter, err := newStamperPutter(s.storer, s.post, s.signer, batch)
	if err != nil {
		s.logger.Debugf("feed put: putter: %v", err)
		s.logger.Error("feed put: putter")
		jsonhttp.BadRequest(w, nil)
		return
	}

	l := loadsave.New(putter, requestModePut(r), false)
	feedManifest, err := manifest.NewDefaultManifest(l, false)
	if err != nil {
		s.logger.Debugf("feed put: new manifest: %v", err)
		s.logger.Error("feed put: new manifest")
		jsonhttp.InternalServerError(w, "create manifest")
		return
	}

	meta := map[string]string{
		feedMetadataEntryOwner: hex.EncodeToString(owner),
		feedMetadataEntryTopic: hex.EncodeToString(topic),
		feedMetadataEntryType:  feeds.Sequence.String(), // only sequence allowed for now
	}

	emptyAddr := make([]byte, 32)

	// a feed manifest stores the metadata at the root "/" path
	err = feedManifest.Add(r.Context(), "/", manifest.NewEntry(penguin.NewAddress(emptyAddr), meta))
	if err != nil {
		s.logger.Debugf("feed post: add manifest entry: %v", err)
		s.logger.Error("feed post: add manifest entry")
		jsonhttp.InternalServerError(w, nil)
		return
	}
	ref, err := feedManifest.Store(r.Context())
	if err != nil {
		s.logger.Debugf("feed post: store manifest: %v", err)
		s.logger.Error("feed post: store manifest")
		jsonhttp.InternalServerError(w, nil)
		return
	}

	if strings.ToLower(r.Header.Get(PenguinPinHeader)) == "true" {
		if err := s.pinning.CreatePin(r.Context(), ref, false); err != nil {
			s.logger.Debugf("feed post: creation of pin for %q failed: %v", ref, err)
			s.logger.Error("feed post: creation of pin failed")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}

	jsonhttp.Created(w, feedReferenceResponse{Reference: ref})
}

func parseFeedUpdate(ch penguin.Chunk) (penguin.Address, int64, error) {
	s, err := soc.FromChunk(ch)
	if err != nil {
		return penguin.ZeroAddress, 0, fmt.Errorf("soc unmarshal: %w", err)
	}

	update := s.WrappedChunk().Data()
	// split the timestamp and reference
	// possible values right now:
	// unencrypted ref: span+timestamp+ref => 8+8+32=48
	// encrypted ref: span+timestamp+ref+decryptKey => 8+8+64=80
	if len(update) != 48 && len(update) != 80 {
		return penguin.ZeroAddress, 0, errInvalidFeedUpdate
	}
	ts := binary.BigEndian.Uint64(update[8:16])
	ref := penguin.NewAddress(update[16:])
	return ref, int64(ts), nil
}
