// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package api provides the functionality of the Pen
// client-facing HTTP API.
package api

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/penguintop/penguin/pkg/xwcclient"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/penguintop/penguin/pkg/crypto"
	"github.com/penguintop/penguin/pkg/feeds"
	"github.com/penguintop/penguin/pkg/file/pipeline/builder"
	"github.com/penguintop/penguin/pkg/logging"
	m "github.com/penguintop/penguin/pkg/metrics"
	"github.com/penguintop/penguin/pkg/pinning"
	"github.com/penguintop/penguin/pkg/postage"
	"github.com/penguintop/penguin/pkg/postage/postagecontract"
	"github.com/penguintop/penguin/pkg/pss"
	"github.com/penguintop/penguin/pkg/resolver"
	"github.com/penguintop/penguin/pkg/steward"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/penguintop/penguin/pkg/tags"
	"github.com/penguintop/penguin/pkg/tracing"
	"github.com/penguintop/penguin/pkg/traversal"
)

const (
	PenguinPinHeader            = "Penguin-Pin"
	PenguinTagHeader            = "Penguin-Tag"
	PenguinEncryptHeader        = "Penguin-Encrypt"
	PenguinIndexDocumentHeader  = "Penguin-Index-Document"
	PenguinErrorDocumentHeader  = "Penguin-Error-Document"
	PenguinFeedIndexHeader      = "Penguin-Feed-Index"
	PenguinFeedIndexNextHeader  = "Penguin-Feed-Index-Next"
	PenguinCollectionHeader     = "Penguin-Collection"
	PenguinPostageBatchIdHeader = "Penguin-Postage-Batch-Id"
)

// The size of buffer used for prefetching content with Langos.
// Warning: This value influences the number of chunk requests and chunker join goroutines
// per file request.
// Recommended value is 8 or 16 times the io.Copy default buffer value which is 32kB, depending
// on the file size. Use lookaheadBufferSize() to get the correct buffer size for the request.
const (
	smallFileBufferSize = 8 * 32 * 1024
	largeFileBufferSize = 16 * 32 * 1024

	largeBufferFilesizeThreshold = 10 * 1000000 // ten megs
)

const (
	contentTypeHeader = "Content-Type"
	multiPartFormData = "multipart/form-data"
	contentTypeTar    = "application/x-tar"
)

var (
	errInvalidNameOrAddress = errors.New("invalid name or pen address")
	errNoResolver           = errors.New("no resolver connected")
	errInvalidRequest       = errors.New("could not validate request")
	errInvalidContentType   = errors.New("invalid content-type")
	errDirectoryStore       = errors.New("could not store directory")
	errFileStore            = errors.New("could not store file")
	errInvalidPostageBatch  = errors.New("invalid postage batch id")
)

// Service is the API service interface.
type Service interface {
	http.Handler
	m.Collector
	io.Closer
}

type server struct {
	tags            *tags.Tags
	storer          storage.Storer
	resolver        resolver.Interface
	pss             pss.Interface
	traversal       traversal.Traverser
	pinning         pinning.Interface
	steward         steward.Reuploader
	logger          logging.Logger
	tracer          *tracing.Tracer
	feedFactory     feeds.Factory
	signer          crypto.Signer
	post            postage.Service
	postageContract postagecontract.Interface
	Options
	http.Handler
	metrics metrics

	wsWg sync.WaitGroup // wait for all websockets to close on exit
	quit chan struct{}

	// xwc rpc related
	swapBackend  *xwcclient.Client
}

type Options struct {
	CORSAllowedOrigins []string
	GatewayMode        bool
	WsPingPeriod       time.Duration
}

const (
	// TargetsRecoveryHeader defines the Header for Recovery targets in Global Pinning
	TargetsRecoveryHeader = "penguin-recovery-targets"
)

// New will create and initialize a new API service.
func New(tags *tags.Tags, storer storage.Storer, resolver resolver.Interface, pss pss.Interface,
	traversalService traversal.Traverser, pinning pinning.Interface, feedFactory feeds.Factory,
	post postage.Service, postageContract postagecontract.Interface, steward steward.Reuploader,
	signer crypto.Signer, swapBackend *xwcclient.Client, logger logging.Logger, tracer *tracing.Tracer, o Options) Service {
	s := &server{
		tags:            tags,
		storer:          storer,
		resolver:        resolver,
		pss:             pss,
		traversal:       traversalService,
		pinning:         pinning,
		feedFactory:     feedFactory,
		post:            post,
		postageContract: postageContract,
		steward:         steward,
		signer:          signer,
		swapBackend: 	 swapBackend,
		Options:         o,
		logger:          logger,
		tracer:          tracer,
		metrics:         newMetrics(),
		quit:            make(chan struct{}),
	}

	s.setupRouting()

	return s
}

// Close hangs up running websockets on shutdown.
func (s *server) Close() error {
	s.logger.Info("API shutting down")
	close(s.quit)

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.wsWg.Wait()
	}()

	select {
		case <-done:
		case <-time.After(1 * time.Second):
			return errors.New("API shutting down with open websockets")
	}

	return nil
}

// getOrCreateTag attempts to get the tag if an id is supplied, and returns an error if it does not exist.
// If no id is supplied, it will attempt to create a new tag with a generated name and return it.
func (s *server) getOrCreateTag(tagUid string) (*tags.Tag, bool, error) {
	// If tag ID is not supplied, create a new tag
	if tagUid == "" {
		tag, err := s.tags.Create(0)
		if err != nil {
			return nil, false, fmt.Errorf("cannot create tag: %w", err)
		}
		return tag, true, nil
	}
	t, err := s.getTag(tagUid)
	return t, false, err
}

func (s *server) getTag(tagUid string) (*tags.Tag, error) {
	uid, err := strconv.Atoi(tagUid)
	if err != nil {
		return nil, fmt.Errorf("cannot parse taguid: %w", err)
	}
	return s.tags.Get(uint32(uid))
}

func (s *server) resolveNameOrAddress(str string) (penguin.Address, error) {
	log := s.logger

	// Try and parse the name as a pen address.
	addr, err := penguin.ParseHexAddress(str)
	if err == nil {
		log.Tracef("Name resolve: valid pen address %q", str)
		return addr, nil
	}

	// If resolver is not available, return an error.
	if s.resolver == nil {
		return penguin.ZeroAddress, errNoResolver
	}

	// Try and resolve the name with the provided resolver.
	log.Debugf("Name resolve: attempting to resolve %s to pen address", str)
	addr, err = s.resolver.Resolve(str)
	if err == nil {
		log.Tracef("Name resolve: resolved name %s to %s", str, addr)
		return addr, nil
	}

	return penguin.ZeroAddress, fmt.Errorf("%w: %v", errInvalidNameOrAddress, err)
}

// requestModePut returns the expected storage.ModePut for this request based on the request headers.
func requestModePut(r *http.Request) storage.ModePut {
	if h := strings.ToLower(r.Header.Get(PenguinPinHeader)); h == "true" {
		return storage.ModePutUploadPin
	}
	return storage.ModePutUpload
}

func requestEncrypt(r *http.Request) bool {
	return strings.ToLower(r.Header.Get(PenguinEncryptHeader)) == "true"
}

func requestPostageBatchId(r *http.Request) ([]byte, error) {
	if h := strings.ToLower(r.Header.Get(PenguinPostageBatchIdHeader)); h != "" {
		if len(h) != 64 {
			return nil, errInvalidPostageBatch
		}
		b, err := hex.DecodeString(h)
		if err != nil {
			return nil, errInvalidPostageBatch
		}
		return b, nil
	}

	return nil, errInvalidPostageBatch
}

func (s *server) newTracingHandler(spanName string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := s.tracer.WithContextFromHTTPHeaders(r.Context(), r.Header)
			if err != nil && !errors.Is(err, tracing.ErrContextNotFound) {
				s.logger.Debugf("span '%s': extract tracing context: %v", spanName, err)
				// ignore
			}

			span, _, ctx := s.tracer.StartSpanFromContext(ctx, spanName, s.logger)
			defer span.Finish()

			err = s.tracer.AddContextHTTPHeader(ctx, r.Header)
			if err != nil {
				s.logger.Debugf("span '%s': inject tracing context: %v", spanName, err)
				// ignore
			}

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func lookaheadBufferSize(size int64) int {
	if size <= largeBufferFilesizeThreshold {
		return smallFileBufferSize
	}
	return largeFileBufferSize
}

// checkOrigin returns true if the origin is not set or is equal to the request host.
func (s *server) checkOrigin(r *http.Request) bool {
	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	hosts := append(s.CORSAllowedOrigins, scheme+"://"+r.Host)
	for _, v := range hosts {
		if equalASCIIFold(origin[0], v) || v == "*" {
			return true
		}
	}

	return false
}

// equalASCIIFold returns true if s is equal to t with ASCII case folding as
// defined in RFC 4790.
func equalASCIIFold(s, t string) bool {
	for s != "" && t != "" {
		sr, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		tr, size := utf8.DecodeRuneInString(t)
		t = t[size:]
		if sr == tr {
			continue
		}
		if 'A' <= sr && sr <= 'Z' {
			sr = sr + 'a' - 'A'
		}
		if 'A' <= tr && tr <= 'Z' {
			tr = tr + 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return s == t
}

type stamperPutter struct {
	storage.Storer
	stamper postage.Stamper
}

func newStamperPutter(s storage.Storer, post postage.Service, signer crypto.Signer, batch []byte) (storage.Storer, error) {
	i, err := post.GetStampIssuer(batch)
	if err != nil {
		return nil, fmt.Errorf("stamp issuer: %w", err)
	}

	stamper := postage.NewStamper(i, signer)
	return &stamperPutter{Storer: s, stamper: stamper}, nil
}

func (p *stamperPutter) Put(ctx context.Context, mode storage.ModePut, chs ...penguin.Chunk) (exists []bool, err error) {
	var (
		ctp []penguin.Chunk
		idx []int
	)
	exists = make([]bool, len(chs))

	for i, c := range chs {
		has, err := p.Storer.Has(ctx, c.Address())
		if err != nil {
			return nil, err
		}
		if has || containsChunk(c.Address(), chs[:i]...) {
			exists[i] = true
			continue
		}
		stamp, err := p.stamper.Stamp(c.Address())
		if err != nil {
			return nil, err
		}
		chs[i] = c.WithStamp(stamp)
		ctp = append(ctp, chs[i])
		idx = append(idx, i)
	}

	exists2, err := p.Storer.Put(ctx, mode, ctp...)
	if err != nil {
		return nil, err
	}
	for i, v := range idx {
		exists[v] = exists2[i]
	}
	return exists, nil
}

type pipelineFunc func(context.Context, io.Reader) (penguin.Address, error)

func requestPipelineFn(s storage.Putter, r *http.Request) pipelineFunc {
	mode, encrypt := requestModePut(r), requestEncrypt(r)
	return func(ctx context.Context, r io.Reader) (penguin.Address, error) {
		pipe := builder.NewPipelineBuilder(ctx, s, mode, encrypt)
		return builder.FeedPipeline(ctx, pipe, r)
	}
}

// calculateNumberOfChunks calculates the number of chunks in an arbitrary
// content length.
func calculateNumberOfChunks(contentLength int64, isEncrypted bool) int64 {
	if contentLength <= penguin.ChunkSize {
		return 1
	}
	branchingFactor := penguin.Branches
	if isEncrypted {
		branchingFactor = penguin.EncryptedBranches
	}

	dataChunks := math.Ceil(float64(contentLength) / float64(penguin.ChunkSize))
	totalChunks := dataChunks
	intermediate := dataChunks / float64(branchingFactor)

	for intermediate > 1 {
		totalChunks += math.Ceil(intermediate)
		intermediate = intermediate / float64(branchingFactor)
	}

	return int64(totalChunks) + 1
}

func requestCalculateNumberOfChunks(r *http.Request) int64 {
	if !strings.Contains(r.Header.Get(contentTypeHeader), "multipart") && r.ContentLength > 0 {
		return calculateNumberOfChunks(r.ContentLength, requestEncrypt(r))
	}
	return 0
}

// containsChunk returns true if the chunk with a specific address
// is present in the provided chunk slice.
func containsChunk(addr penguin.Address, chs ...penguin.Chunk) bool {
	for _, c := range chs {
		if addr.Equal(c.Address()) {
			return true
		}
	}
	return false
}
