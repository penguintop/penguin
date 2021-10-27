// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/postage"
	"github.com/penguintop/penguin/pkg/pss"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	writeDeadline   = 4 * time.Second // write deadline. should be smaller than the shutdown timeout on api close
	targetMaxLength = 2               // max target length in bytes, in order to prevent grieving by excess computation
)

func (s *server) pssPostHandler(w http.ResponseWriter, r *http.Request) {
	topicVar := mux.Vars(r)["topic"]
	topic := pss.NewTopic(topicVar)

	targetsVar := mux.Vars(r)["targets"]
	var targets pss.Targets
	tgts := strings.Split(targetsVar, ",")

	for _, v := range tgts {
		target, err := hex.DecodeString(v)
		if err != nil || len(target) > targetMaxLength {
			s.logger.Debugf("Pss send: bad targets: %v", err)
			s.logger.Error("Pss send: bad targets")
			jsonhttp.BadRequest(w, nil)
			return
		}
		targets = append(targets, target)
	}

	recipientQueryString := r.URL.Query().Get("recipient")
	var recipient *ecdsa.PublicKey
	if recipientQueryString == "" {
		// Use topic-based encryption
		privkey := crypto.Secp256k1PrivateKeyFromBytes(topic[:])
		recipient = &privkey.PublicKey
	} else {
		var err error
		recipient, err = pss.ParseRecipient(recipientQueryString)
		if err != nil {
			s.logger.Debugf("Pss recipient: %v", err)
			s.logger.Error("Pss recipient")
			jsonhttp.BadRequest(w, nil)
			return
		}
	}

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.logger.Debugf("Pss read payload: %v", err)
		s.logger.Error("Pss read payload")
		jsonhttp.InternalServerError(w, nil)
		return
	}
	batch, err := requestPostageBatchId(r)
	if err != nil {
		s.logger.Debugf("Pss: postage batch id: %v", err)
		s.logger.Error("Pss: postage batch id")
		jsonhttp.BadRequest(w, "invalid postage batch id")
		return
	}
	i, err := s.post.GetStampIssuer(batch)
	if err != nil {
		s.logger.Debugf("Pss: postage batch issuer: %v", err)
		s.logger.Error("Pss: postage batch issue")
		jsonhttp.BadRequest(w, "postage stamp issuer")
		return
	}
	stamper := postage.NewStamper(i, s.signer)

	err = s.pss.Send(r.Context(), topic, payload, stamper, recipient, targets)
	if err != nil {
		s.logger.Debugf("Pss send payload: %v. topic: %s", err, topicVar)
		s.logger.Error("Pss send payload")
		jsonhttp.InternalServerError(w, nil)
		return
	}

	jsonhttp.Created(w, nil)
}

func (s *server) pssWsHandler(w http.ResponseWriter, r *http.Request) {

	upgrader := websocket.Upgrader{
		ReadBufferSize:  penguin.ChunkSize,
		WriteBufferSize: penguin.ChunkSize,
		CheckOrigin:     s.checkOrigin,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Debugf("Pss ws: upgrade: %v", err)
		s.logger.Error("Pss ws: cannot upgrade")
		jsonhttp.InternalServerError(w, nil)
		return
	}

	t := mux.Vars(r)["topic"]
	s.wsWg.Add(1)
	go s.pumpWs(conn, t)
}

func (s *server) pumpWs(conn *websocket.Conn, t string) {
	defer s.wsWg.Done()

	var (
		dataC  = make(chan []byte)
		gone   = make(chan struct{})
		topic  = pss.NewTopic(t)
		ticker = time.NewTicker(s.WsPingPeriod)
		err    error
	)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	cleanup := s.pss.Register(topic, func(_ context.Context, m []byte) {
		dataC <- m
	})

	defer cleanup()

	conn.SetCloseHandler(func(code int, text string) error {
		s.logger.Debugf("Pss handler: client gone. code %d message %s", code, text)
		close(gone)
		return nil
	})

	for {
		select {
		case b := <-dataC:
			err = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err != nil {
				s.logger.Debugf("Pss set write deadline: %v", err)
				return
			}

			err = conn.WriteMessage(websocket.BinaryMessage, b)
			if err != nil {
				s.logger.Debugf("Pss write to websocket: %v", err)
				return
			}

		case <-s.quit:
			// shutdown
			err = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err != nil {
				s.logger.Debugf("Pss set write deadline: %v", err)
				return
			}
			err = conn.WriteMessage(websocket.CloseMessage, []byte{})
			if err != nil {
				s.logger.Debugf("Pss write close message: %v", err)
			}
			return
		case <-gone:
			// client gone
			return
		case <-ticker.C:
			err = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err != nil {
				s.logger.Debugf("Pss set write deadline: %v", err)
				return
			}
			if err = conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// Error encountered while pinging client, client probably gone.
				return
			}
		}
	}
}
