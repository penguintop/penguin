// Copyright 2020 The Penguin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/penguintop/penguin/pkg/file"
	"github.com/penguintop/penguin/pkg/file/loadsave"
	"github.com/penguintop/penguin/pkg/jsonhttp"
	"github.com/penguintop/penguin/pkg/logging"
	"github.com/penguintop/penguin/pkg/manifest"
	"github.com/penguintop/penguin/pkg/sctx"
	"github.com/penguintop/penguin/pkg/storage"
	"github.com/penguintop/penguin/pkg/penguin"
	"github.com/penguintop/penguin/pkg/tags"
	"github.com/penguintop/penguin/pkg/tracing"
)

// dirUploadHandler uploads a directory supplied as a tar in a HTTP request
func (s *server) dirUploadHandler(w http.ResponseWriter, r *http.Request, storer storage.Storer) {
	logger := tracing.NewLoggerWithTraceID(r.Context(), s.logger)
	if r.Body == http.NoBody {
		logger.Error("Pen upload dir: request has no body")
		jsonhttp.BadRequest(w, errInvalidRequest)
		return
	}
	contentType := r.Header.Get(contentTypeHeader)
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		logger.Errorf("Pen upload dir: invalid content-type")
		logger.Debugf("Pen upload dir: invalid content-type err: %v", err)
		jsonhttp.BadRequest(w, errInvalidContentType)
		return
	}

	var dReader dirReader
	switch mediaType {
	case contentTypeTar:
		dReader = &tarReader{r: tar.NewReader(r.Body), logger: s.logger}
	case multiPartFormData:
		dReader = &multipartReader{r: multipart.NewReader(r.Body, params["boundary"])}
	default:
		logger.Error("Pen upload dir: invalid content-type for directory upload")
		jsonhttp.BadRequest(w, errInvalidContentType)
		return
	}
	defer r.Body.Close()

	tag, created, err := s.getOrCreateTag(r.Header.Get(PenguinTagHeader))
	if err != nil {
		logger.Debugf("Pen upload dir: get or create tag: %v", err)
		logger.Error("Pen upload dir: get or create tag")
		jsonhttp.InternalServerError(w, nil)
		return
	}

	// Add the tag to the context
	ctx := sctx.SetTag(r.Context(), tag)

	reference, err := storeDir(
		ctx,
		requestEncrypt(r),
		dReader,
		s.logger,
		requestPipelineFn(storer, r),
		loadsave.New(storer, requestModePut(r), requestEncrypt(r)),
		r.Header.Get(PenguinIndexDocumentHeader),
		r.Header.Get(PenguinErrorDocumentHeader),
		tag,
		created,
	)
	if err != nil {
		logger.Debugf("Pen upload dir: store dir err: %v", err)
		logger.Errorf("Pen upload dir: store dir")
		jsonhttp.InternalServerError(w, errDirectoryStore)
		return
	}
	if created {
		_, err = tag.DoneSplit(reference)
		if err != nil {
			logger.Debugf("Pen upload dir: done split: %v", err)
			logger.Error("Pen upload dir: done split failed")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}

	if strings.ToLower(r.Header.Get(PenguinPinHeader)) == "true" {
		if err := s.pinning.CreatePin(r.Context(), reference, false); err != nil {
			logger.Debugf("Pen upload dir: creation of pin for %q failed: %v", reference, err)
			logger.Error("Pen upload dir: creation of pin failed")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}

	w.Header().Set(PenguinTagHeader, fmt.Sprint(tag.Uid))
	jsonhttp.Created(w, penUploadResponse{
		Reference: reference,
	})
}

// storeDir stores all files recursively contained in the directory given as a tar/multipart
// it returns the hash for the uploaded manifest corresponding to the uploaded dir
func storeDir(
	ctx context.Context,
	encrypt bool,
	reader dirReader,
	log logging.Logger,
	p pipelineFunc,
	ls file.LoadSaver,
	indexFilename,
	errorFilename string,
	tag *tags.Tag,
	tagCreated bool,
) (penguin.Address, error) {
	logger := tracing.NewLoggerWithTraceID(ctx, log)

	dirManifest, err := manifest.NewDefaultManifest(ls, encrypt)
	if err != nil {
		return penguin.ZeroAddress, err
	}

	if indexFilename != "" && strings.ContainsRune(indexFilename, '/') {
		return penguin.ZeroAddress, fmt.Errorf("index document suffix must not include slash character")
	}

	filesAdded := 0

	// Iterate through the files in the supplied tar
	for {
		fileInfo, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return penguin.ZeroAddress, fmt.Errorf("read tar stream: %w", err)
		}

		if !tagCreated {
			// Only in the case when tag is sent via header (i.e. not created by this request)
			// for each file
			if estimatedTotalChunks := calculateNumberOfChunks(fileInfo.Size, encrypt); estimatedTotalChunks > 0 {
				err = tag.IncN(tags.TotalChunks, estimatedTotalChunks)
				if err != nil {
					return penguin.ZeroAddress, fmt.Errorf("increment tag: %w", err)
				}
			}
		}

		fileReference, err := p(ctx, fileInfo.Reader)
		if err != nil {
			return penguin.ZeroAddress, fmt.Errorf("store dir file: %w", err)
		}
		logger.Tracef("Uploaded dir file %v with reference %v", fileInfo.Path, fileReference)

		fileMtdt := map[string]string{
			manifest.EntryMetadataContentTypeKey: fileInfo.ContentType,
			manifest.EntryMetadataFilenameKey:    fileInfo.Name,
		}
		// Add file entry to dir manifest
		err = dirManifest.Add(ctx, fileInfo.Path, manifest.NewEntry(fileReference, fileMtdt))
		if err != nil {
			return penguin.ZeroAddress, fmt.Errorf("add to manifest: %w", err)
		}

		filesAdded++
	}

	// Check if files were uploaded through the manifest
	if filesAdded == 0 {
		return penguin.ZeroAddress, fmt.Errorf("no files in tar")
	}

	// Store website information
	if indexFilename != "" || errorFilename != "" {
		metadata := map[string]string{}
		if indexFilename != "" {
			metadata[manifest.WebsiteIndexDocumentSuffixKey] = indexFilename
		}
		if errorFilename != "" {
			metadata[manifest.WebsiteErrorDocumentPathKey] = errorFilename
		}
		rootManifestEntry := manifest.NewEntry(penguin.ZeroAddress, metadata)
		err = dirManifest.Add(ctx, manifest.RootPath, rootManifestEntry)
		if err != nil {
			return penguin.ZeroAddress, fmt.Errorf("add to manifest: %w", err)
		}
	}

	storeSizeFn := []manifest.StoreSizeFunc{}
	if !tagCreated {
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

	// Save manifest
	manifestReference, err := dirManifest.Store(ctx, storeSizeFn...)
	if err != nil {
		return penguin.ZeroAddress, fmt.Errorf("store manifest: %w", err)
	}
	logger.Tracef("Finished uploaded dir with reference %v", manifestReference)

	return manifestReference, nil
}

type FileInfo struct {
	Path        string
	Name        string
	ContentType string
	Size        int64
	Reader      io.Reader
}

type dirReader interface {
	Next() (*FileInfo, error)
}

type tarReader struct {
	r      *tar.Reader
	logger logging.Logger
}

func (t *tarReader) Next() (*FileInfo, error) {
	for {
		fileHeader, err := t.r.Next()
		if err != nil {
			return nil, err
		}

		fileName := fileHeader.FileInfo().Name()
		contentType := mime.TypeByExtension(filepath.Ext(fileHeader.Name))
		fileSize := fileHeader.FileInfo().Size()
		filePath := filepath.Clean(fileHeader.Name)

		if filePath == "." {
			t.logger.Warning("Skipping file upload empty path")
			continue
		}
		if runtime.GOOS == "windows" {
			// Always use Unix path separator
			filePath = filepath.ToSlash(filePath)
		}
		// only store regular files
		if !fileHeader.FileInfo().Mode().IsRegular() {
			t.logger.Warningf("Skipping file upload for %s as it is not a regular file", filePath)
			continue
		}

		return &FileInfo{
			Path:        filePath,
			Name:        fileName,
			ContentType: contentType,
			Size:        fileSize,
			Reader:      t.r,
		}, nil
	}
}

// multipart reader returns files added as a multipart form. We will ensure all the
// part headers are passed correctly
type multipartReader struct {
	r *multipart.Reader
}

func (m *multipartReader) Next() (*FileInfo, error) {
	part, err := m.r.NextPart()
	if err != nil {
		return nil, err
	}

	fileName := part.FileName()
	if fileName == "" {
		fileName = part.FormName()
	}
	if fileName == "" {
		return nil, errors.New("filename missing")
	}

	contentType := part.Header.Get(contentTypeHeader)
	if contentType == "" {
		return nil, errors.New("content-type missing")
	}

	contentLength := part.Header.Get("Content-Length")
	if contentLength == "" {
		return nil, errors.New("content-length missing")
	}
	fileSize, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return nil, errors.New("invalid file size")
	}

	if filepath.Dir(fileName) != "." {
		return nil, errors.New("multipart upload supports only single directory")
	}

	return &FileInfo{
		Path:        fileName,
		Name:        fileName,
		ContentType: contentType,
		Size:        fileSize,
		Reader:      part,
	}, nil
}
