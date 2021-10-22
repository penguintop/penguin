// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"

	"github.com/penguintop/penguin/pkg/feeds"
	"github.com/penguintop/penguin/pkg/file/joiner"
	"github.com/penguintop/penguin/pkg/file/loadsave"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/manifest"
	"github.com/penguintop/penguin/pkg/sctx"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/penguintop/penguin/pkg/tags"
	"github.com/penguintop/penguin/pkg/tracing"
	"github.com/ethersphere/langos"
)

func (s *server) penUploadHandler(w http.ResponseWriter, r *http.Request) {
	logger := tracing.NewLoggerWithTraceID(r.Context(), s.logger)

	contentType := r.Header.Get(contentTypeHeader)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		logger.Debugf("Pen upload: parse content type header %q: %v", contentType, err)
		logger.Errorf("Pen upload: parse content type header %q", contentType)
		jsonhttp.BadRequest(w, errInvalidContentType)
		return
	}

	// get Penguin-Postage-Batch-Id from http header
	batch, err := requestPostageBatchId(r)
	if err != nil {
		logger.Debugf("Pen upload: postage batch id: %v", err)
		logger.Error("Pen upload: postage batch id")
		jsonhttp.BadRequest(w, "invalid postage batch id")
		return
	}

	putter, err := newStamperPutter(s.storer, s.post, s.signer, batch)
	if err != nil {
		logger.Debugf("Pen upload: putter: %v", err)
		logger.Error("Pen upload: putter")
		jsonhttp.BadRequest(w, nil)
		return
	}

	isDir := r.Header.Get(PenguinCollectionHeader)
	if strings.ToLower(isDir) == "true" || mediaType == multiPartFormData {
		s.dirUploadHandler(w, r, putter)
		return
	}
	s.fileUploadHandler(w, r, putter)
}

// penUploadResponse will be returned when HTTP requests to upload a file successfully.
type penUploadResponse struct {
	Reference penguin.Address `json:"reference"`
}

// fileUploadHandler uploads the file and its metadata supplied in the file body and
// the headers
func (s *server) fileUploadHandler(w http.ResponseWriter, r *http.Request, storer storage.Storer) {
	logger := tracing.NewLoggerWithTraceID(r.Context(), s.logger)
	var (
		reader   io.Reader
		fileName string
	)

	// Content-Type has already been validated by this time
	contentType := r.Header.Get(contentTypeHeader)

	tag, created, err := s.getOrCreateTag(r.Header.Get(PenguinTagHeader))
	if err != nil {
		logger.Debugf("Pen upload file: get or create tag: %v", err)
		logger.Error("Pen upload file: get or create tag")
		jsonhttp.InternalServerError(w, nil)
		return
	}

	if !created {
		// Only in the case when tag is sent via header (i.e. not created by this request)
		if estimatedTotalChunks := requestCalculateNumberOfChunks(r); estimatedTotalChunks > 0 {
			err = tag.IncN(tags.TotalChunks, estimatedTotalChunks)
			if err != nil {
				s.logger.Debugf("Pen upload file: increment tag: %v", err)
				s.logger.Error("Pen upload file: increment tag")
				jsonhttp.InternalServerError(w, nil)
				return
			}
		}
	}

	// Add the tag to the context
	ctx := sctx.SetTag(r.Context(), tag)

	fileName = r.URL.Query().Get("name")
	reader = r.Body

	p := requestPipelineFn(storer, r)

	// Firstly store the file and get its reference
	fr, err := p(ctx, reader)
	if err != nil {
		logger.Debugf("Pen upload file: file store, file %q: %v", fileName, err)
		logger.Errorf("Pen upload file: file store, file %q", fileName)
		jsonhttp.InternalServerError(w, errFileStore)
		return
	}

	// If the file name is still empty, use the file hash as the file name
	if fileName == "" {
		fileName = fr.String()
	}

	encrypt := requestEncrypt(r)
	l := loadsave.New(storer, requestModePut(r), encrypt)

	m, err := manifest.NewDefaultManifest(l, encrypt)
	if err != nil {
		logger.Debugf("Pen upload file: create manifest, file %q: %v", fileName, err)
		logger.Errorf("Pen upload file: create manifest, file %q", fileName)
		jsonhttp.InternalServerError(w, nil)
		return
	}

	rootMetadata := map[string]string{
		manifest.WebsiteIndexDocumentSuffixKey: fileName,
	}

	err = m.Add(ctx, manifest.RootPath, manifest.NewEntry(penguin.ZeroAddress, rootMetadata))
	if err != nil {
		logger.Debugf("Pen upload file: adding metadata to manifest, file %q: %v", fileName, err)
		logger.Errorf("Pen upload file: adding metadata to manifest, file %q", fileName)
		jsonhttp.InternalServerError(w, nil)
		return
	}

	fileMtdt := map[string]string{
		manifest.EntryMetadataContentTypeKey: contentType,
		manifest.EntryMetadataFilenameKey:    fileName,
	}

	err = m.Add(ctx, fileName, manifest.NewEntry(fr, fileMtdt))
	if err != nil {
		logger.Debugf("Pen upload file: adding file to manifest, file %q: %v", fileName, err)
		logger.Errorf("Pen upload file: adding file to manifest, file %q", fileName)
		jsonhttp.InternalServerError(w, nil)
		return
	}

	logger.Debugf("Uploading file Encrypt: %v Filename: %s Filehash: %s FileMtdt: %v",
		encrypt, fileName, fr.String(), fileMtdt)

	storeSizeFn := []manifest.StoreSizeFunc{}
	if !created {
		// Only in the case when tag is sent via header (i.e. not created by this request)
		// each content that is saved for manifest
		storeSizeFn = append(storeSizeFn, func(dataSize int64) error {
			if estimatedTotalChunks := calculateNumberOfChunks(dataSize, encrypt); estimatedTotalChunks > 0 {
				err = tag.IncN(tags.TotalChunks, estimatedTotalChunks)
				if err != nil {
					return fmt.Errorf("increment tag: %w", err)
				}
			}
			return nil
		})
	}

	manifestReference, err := m.Store(ctx, storeSizeFn...)
	if err != nil {
		logger.Debugf("Pen upload file: manifest store, file %q: %v", fileName, err)
		logger.Errorf("Pen upload file: manifest store, file %q", fileName)
		jsonhttp.InternalServerError(w, nil)
		return
	}
	logger.Debugf("Manifest Reference: %s", manifestReference.String())

	if created {
		_, err = tag.DoneSplit(manifestReference)
		if err != nil {
			logger.Debugf("Pen upload file: done split: %v", err)
			logger.Error("Pen upload file: done split failed")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}

	if strings.ToLower(r.Header.Get(PenguinPinHeader)) == "true" {
		if err := s.pinning.CreatePin(ctx, manifestReference, false); err != nil {
			logger.Debugf("Pen upload file: creation of pin for %q failed: %v", manifestReference, err)
			logger.Error("Pen upload file: creation of pin failed")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}

	w.Header().Set("ETag", fmt.Sprintf("%q", manifestReference.String()))
	w.Header().Set(PenguinTagHeader, fmt.Sprint(tag.Uid))
	w.Header().Set("Access-Control-Expose-Headers", PenguinTagHeader)
	jsonhttp.Created(w, penUploadResponse{
		Reference: manifestReference,
	})
}

func (s *server) penDownloadHandler(w http.ResponseWriter, r *http.Request) {
	logger := tracing.NewLoggerWithTraceID(r.Context(), s.logger)
	ls := loadsave.New(s.storer, storage.ModePutRequest, false)
	feedDereferenced := false

	targets := r.URL.Query().Get("targets")
	if targets != "" {
		r = r.WithContext(sctx.SetTargets(r.Context(), targets))
	}
	ctx := r.Context()

	nameOrHex := mux.Vars(r)["address"]
	pathVar := mux.Vars(r)["path"]
	if strings.HasSuffix(pathVar, "/") {
		pathVar = strings.TrimRight(pathVar, "/")
		// NOTE: leave one slash if there was some
		pathVar += "/"
	}

	address, err := s.resolveNameOrAddress(nameOrHex)
	if err != nil {
		logger.Debugf("Pen download: parse address %s: %v", nameOrHex, err)
		logger.Error("Pen download: parse address")
		jsonhttp.NotFound(w, nil)
		return
	}

FETCH:
	// read manifest entry
	m, err := manifest.NewDefaultManifestReference(
		address,
		ls,
	)
	if err != nil {
		logger.Debugf("Pen download: not manifest %s: %v", address, err)
		logger.Error("Pen download: not manifest")
		jsonhttp.NotFound(w, nil)
		return
	}

	// There's a possible ambiguity here, right now the data which was
	// read can be an entry.Entry or a mantaray feed manifest. Try to
	// unmarshal as mantaray first and possibly resolve the feed, otherwise
	// go on normally.
	if !feedDereferenced {
		if l, err := s.manifestFeed(ctx, m); err == nil {
			//We have a feed manifest here
			ch, cur, _, err := l.At(ctx, time.Now().Unix(), 0)
			if err != nil {
				logger.Debugf("Pen download: feed lookup: %v", err)
				logger.Error("Pen download: feed lookup")
				jsonhttp.NotFound(w, "feed not found")
				return
			}
			if ch == nil {
				logger.Debugf("Pen download: feed lookup: no updates")
				logger.Error("Pen download: feed lookup")
				jsonhttp.NotFound(w, "no update found")
				return
			}
			ref, _, err := parseFeedUpdate(ch)
			if err != nil {
				logger.Debugf("Pen download: parse feed update: %v", err)
				logger.Error("Pen download: parse feed update")
				jsonhttp.InternalServerError(w, "parse feed update")
				return
			}
			address = ref
			feedDereferenced = true
			curBytes, err := cur.MarshalBinary()
			if err != nil {
				s.logger.Debugf("Pen download: marshal feed index: %v", err)
				s.logger.Error("Pen download: marshal index")
				jsonhttp.InternalServerError(w, "marshal index")
				return
			}

			w.Header().Set(PenguinFeedIndexHeader, hex.EncodeToString(curBytes))
			// This header might be overriding others. handle with care. in the future
			// we should implement an append functionality for this specific header,
			// since different parts of handlers might be overriding others' values
			// resulting in inconsistent headers in the response.
			w.Header().Set("Access-Control-Expose-Headers", PenguinFeedIndexHeader)
			goto FETCH
		}
	}

	if pathVar == "" {
		logger.Tracef("Pen download: handle empty path %s", address)

		if indexDocumentSuffixKey, ok := manifestMetadataLoad(ctx, m, manifest.RootPath, manifest.WebsiteIndexDocumentSuffixKey); ok {
			pathWithIndex := path.Join(pathVar, indexDocumentSuffixKey)
			indexDocumentManifestEntry, err := m.Lookup(ctx, pathWithIndex)
			if err == nil {
				// Index document exists
				logger.Debugf("Pen download: serving path: %s", pathWithIndex)

				s.serveManifestEntry(w, r, address, indexDocumentManifestEntry, !feedDereferenced)
				return
			}
		}
	}

	me, err := m.Lookup(ctx, pathVar)
	if err != nil {
		logger.Debugf("Pen download: invalid path %s/%s: %v", address, pathVar, err)
		logger.Error("Pen download: invalid path")

		if errors.Is(err, manifest.ErrNotFound) {

			if !strings.HasPrefix(pathVar, "/") {
				// Check for directory
				dirPath := pathVar + "/"
				exists, err := m.HasPrefix(ctx, dirPath)
				if err == nil && exists {
					// Redirect to directory
					u := r.URL
					u.Path += "/"
					redirectURL := u.String()

					logger.Debugf("Pen download: redirecting to %s: %v", redirectURL, err)

					http.Redirect(w, r, redirectURL, http.StatusPermanentRedirect)
					return
				}
			}

			// Check index suffix path
			if indexDocumentSuffixKey, ok := manifestMetadataLoad(ctx, m, manifest.RootPath, manifest.WebsiteIndexDocumentSuffixKey); ok {
				if !strings.HasSuffix(pathVar, indexDocumentSuffixKey) {
					// Check if path is directory with indexing
					pathWithIndex := path.Join(pathVar, indexDocumentSuffixKey)
					indexDocumentManifestEntry, err := m.Lookup(ctx, pathWithIndex)
					if err == nil {
						// Index document exists
						logger.Debugf("Pen download: serving path: %s", pathWithIndex)

						s.serveManifestEntry(w, r, address, indexDocumentManifestEntry, !feedDereferenced)
						return
					}
				}
			}

			// Check if error document is to be shown
			if errorDocumentPath, ok := manifestMetadataLoad(ctx, m, manifest.RootPath, manifest.WebsiteErrorDocumentPathKey); ok {
				if pathVar != errorDocumentPath {
					errorDocumentManifestEntry, err := m.Lookup(ctx, errorDocumentPath)
					if err == nil {
						// Error document exists
						logger.Debugf("Pen download: serving path: %s", errorDocumentPath)

						s.serveManifestEntry(w, r, address, errorDocumentManifestEntry, !feedDereferenced)
						return
					}
				}
			}

			jsonhttp.NotFound(w, "path address not found")
		} else {
			jsonhttp.NotFound(w, nil)
		}
		return
	}

	// Serve requested path
	s.serveManifestEntry(w, r, address, me, !feedDereferenced)
}

func (s *server) serveManifestEntry(
	w http.ResponseWriter,
	r *http.Request,
	address penguin.Address,
	manifestEntry manifest.Entry,
	etag bool,
) {

	additionalHeaders := http.Header{}
	mtdt := manifestEntry.Metadata()
	if fname, ok := mtdt[manifest.EntryMetadataFilenameKey]; ok {
		additionalHeaders["Content-Disposition"] =
			[]string{fmt.Sprintf("inline; filename=\"%s\"", fname)}
	}
	if mimeType, ok := mtdt[manifest.EntryMetadataContentTypeKey]; ok {
		additionalHeaders["Content-Type"] = []string{mimeType}
	}

	s.downloadHandler(w, r, manifestEntry.Reference(), additionalHeaders, etag)
}

// downloadHandler contains common logic for dowloading Penguin file from API
func (s *server) downloadHandler(w http.ResponseWriter, r *http.Request, reference penguin.Address, additionalHeaders http.Header, etag bool) {
	logger := tracing.NewLoggerWithTraceID(r.Context(), s.logger)
	targets := r.URL.Query().Get("targets")
	if targets != "" {
		r = r.WithContext(sctx.SetTargets(r.Context(), targets))
	}

	reader, l, err := joiner.New(r.Context(), s.storer, reference)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			logger.Debugf("API download: not found %s: %v", reference, err)
			logger.Error("API download: not found")
			jsonhttp.NotFound(w, nil)
			return
		}
		logger.Debugf("API download: unexpected error %s: %v", reference, err)
		logger.Error("API download: unexpected error")
		jsonhttp.InternalServerError(w, nil)
		return
	}

	// Include additional headers
	for name, values := range additionalHeaders {
		w.Header().Set(name, strings.Join(values, "; "))
	}
	if etag {
		w.Header().Set("ETag", fmt.Sprintf("%q", reference))
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", l))
	w.Header().Set("Decompressed-Content-Length", fmt.Sprintf("%d", l))
	w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	if targets != "" {
		w.Header().Set(TargetsRecoveryHeader, targets)
	}
	http.ServeContent(w, r, "", time.Now(), langos.NewBufferedLangos(reader, lookaheadBufferSize(l)))
}

// manifestMetadataLoad returns the value for a key stored in the metadata of
// manifest path, or empty string if no value is present.
// The ok result indicates whether value was found in the metadata.
func manifestMetadataLoad(
	ctx context.Context,
	manifest manifest.Interface,
	path, metadataKey string,
) (string, bool) {
	me, err := manifest.Lookup(ctx, path)
	if err != nil {
		return "", false
	}

	manifestRootMetadata := me.Metadata()
	if val, ok := manifestRootMetadata[metadataKey]; ok {
		return val, ok
	}

	return "", false
}

func (s *server) manifestFeed(
	ctx context.Context,
	m manifest.Interface,
) (feeds.Lookup, error) {
	e, err := m.Lookup(ctx, "/")
	if err != nil {
		return nil, fmt.Errorf("node lookup: %w", err)
	}
	var (
		owner, topic []byte
		t            = new(feeds.Type)
	)
	meta := e.Metadata()
	if e := meta[feedMetadataEntryOwner]; e != "" {
		owner, err = hex.DecodeString(e)
		if err != nil {
			return nil, err
		}
	}
	if e := meta[feedMetadataEntryTopic]; e != "" {
		topic, err = hex.DecodeString(e)
		if err != nil {
			return nil, err
		}
	}
	if e := meta[feedMetadataEntryType]; e != "" {
		err := t.FromString(e)
		if err != nil {
			return nil, err
		}
	}
	if len(owner) == 0 || len(topic) == 0 {
		return nil, fmt.Errorf("node lookup: %s", "feed metadata absent")
	}
	f := feeds.New(topic, common.BytesToAddress(owner))
	return s.feedFactory.NewLookup(*t, f)
}

func (s *server) penPatchHandler(w http.ResponseWriter, r *http.Request) {
	nameOrHex := mux.Vars(r)["address"]
	address, err := s.resolveNameOrAddress(nameOrHex)
	if err != nil {
		s.logger.Debugf("Pen patch: parse address %s: %v", nameOrHex, err)
		s.logger.Error("Pen patch: parse address")
		jsonhttp.NotFound(w, nil)
		return
	}
	err = s.steward.Reupload(r.Context(), address)
	if err != nil {
		s.logger.Debugf("Pen patch: reupload %s: %v", address.String(), err)
		s.logger.Error("Pen patch: reupload")
		jsonhttp.InternalServerError(w, nil)
		return
	}
	jsonhttp.OK(w, nil)
}
