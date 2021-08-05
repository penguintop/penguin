// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package p2p provides the peer-to-peer abstractions used
// across different protocols in Pen.
package p2p

import (
	"context"
	"io"
	"time"

	"github.com/penguintop/penguin/pkg/pen"
    "github.com/penguintop/penguin/pkg/penguin"
	ma "github.com/multiformats/go-multiaddr"
)

// Service provides methods to handle p2p Peers and Protocols.
type Service interface {
	AddProtocol(ProtocolSpec) error
	// Connect to a peer but do not notify topology about the established connection.
	Connect(ctx context.Context, addr ma.Multiaddr) (address *pen.Address, err error)
	Disconnecter
	Peers() []Peer
	BlocklistedPeers() ([]Peer, error)
	Addresses() ([]ma.Multiaddr, error)
	SetPickyNotifier(PickyNotifier)
	Halter
}

type Disconnecter interface {
	Disconnect(overlay penguin.Address) error
	// Blocklist will disconnect a peer and put it on a blocklist (blocking in & out connections) for provided duration
	// duration 0 is treated as an infinite duration
	Blocklist(overlay penguin.Address, duration time.Duration) error
}

type Halter interface {
	// Halt new incoming connections while shutting down
	Halt()
}

// PickyNotifer can decide whether a peer should be picked
type PickyNotifier interface {
	Pick(Peer) bool
	Notifier
}

type Notifier interface {
	Connected(context.Context, Peer) error
	Disconnected(Peer)
	Announce(context.Context, penguin.Address, bool) error
}

// DebugService extends the Service with method used for debugging.
type DebugService interface {
	Service
	SetWelcomeMessage(val string) error
	GetWelcomeMessage() string
}

// Streamer is able to create a new Stream.
type Streamer interface {
	NewStream(ctx context.Context, address penguin.Address, h Headers, protocol, version, stream string) (Stream, error)
}

type StreamerDisconnecter interface {
	Streamer
	Disconnecter
}

// Stream represent a bidirectional data Stream.
type Stream interface {
	io.ReadWriter
	io.Closer
	ResponseHeaders() Headers
	Headers() Headers
	FullClose() error
	Reset() error
}

// ProtocolSpec defines a collection of Stream specifications with handlers.
type ProtocolSpec struct {
	Name          string
	Version       string
	StreamSpecs   []StreamSpec
	ConnectIn     func(context.Context, Peer) error
	ConnectOut    func(context.Context, Peer) error
	DisconnectIn  func(Peer) error
	DisconnectOut func(Peer) error
}

// StreamSpec defines a Stream handling within the protocol.
type StreamSpec struct {
	Name    string
	Handler HandlerFunc
	Headler HeadlerFunc
}

// Peer holds information about a Peer.
type Peer struct {
	Address  penguin.Address `json:"address"`
	FullNode bool            `json:"fullNode"`
}

// HandlerFunc handles a received Stream from a Peer.
type HandlerFunc func(context.Context, Peer, Stream) error

// HandlerMiddleware decorates a HandlerFunc by returning a new one.
type HandlerMiddleware func(HandlerFunc) HandlerFunc

// HeadlerFunc is returning response headers based on the received request
// headers.
type HeadlerFunc func(Headers, penguin.Address) Headers

// Headers represents a collection of p2p header key value pairs.
type Headers map[string][]byte

// Common header names.
const (
	HeaderNameTracingSpanContext = "tracing-span-context"
)

// NewPenguinStreamName constructs a libp2p compatible stream name out of
// protocol name and version and stream name.
func NewPenguinStreamName(protocol, version, stream string) string {
	return "/penguin/" + protocol + "/" + version + "/" + stream
}
