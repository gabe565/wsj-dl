package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

func upload(conf *Config, s3 *minio.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != conf.UploadAuthKey {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		url := r.URL.Query().Get("url")
		if url == "" {
			handleHTTPError(w, "Missing url", http.StatusBadRequest)
			return
		}

		filename := filepath.Base(url)

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("User-Agent", conf.UploadUserAgent)

		client := &http.Client{
			CheckRedirect: func(req *http.Request, _ []*http.Request) error {
				filename = filepath.Base(req.URL.Path)
				return nil
			},
		}

		res, err := client.Do(req)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
		}()

		if res.StatusCode != http.StatusOK {
			handleHTTPError(w, res.Status, res.StatusCode)
			return
		}

		flatFilename := filename
		if d, err := getDate(filename); err == nil {
			ext := filepath.Ext(filename)
			filename = d.Format("2006/01/02") + ext
			flatFilename = d.Format("2006-01-02") + ext
		}

		_, err = s3.PutObject(r.Context(), conf.S3Bucket, filename, res.Body, res.ContentLength, minio.PutObjectOptions{
			ContentType:        res.Header.Get("Content-Type"),
			ContentDisposition: "attachment; filename=" + flatFilename,
		})
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("Loaded file", "filename", filename, "url", url)

		fileURL := *r.URL
		fileURL.Path = flatFilename
		fileURL.RawQuery = ""
		_, _ = io.WriteString(w, fileURL.String()+"\n")
	}
}

var ErrInvalidFilename = errors.New("invalid filename")

func getDate(filename string) (time.Time, error) {
	log := slog.With("filename", filename)

	_, filename, ok := strings.Cut(filename, "-")
	if !ok {
		log.Warn("Filename missing random prefix")
		return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidFilename, "missing random prefix")
	}

	_, filename, ok = strings.Cut(filename, "-")
	if !ok {
		log.Warn("Filename missing non-random prefix")
		return time.Time{}, fmt.Errorf("%w: %s", ErrInvalidFilename, "missing non-random prefix")
	}

	ext := filepath.Ext(filename)
	filename = strings.TrimSuffix(filename, ext)

	d, err := time.Parse("1-2-2006", filename)
	if err != nil {
		log.Warn("Failed to parse filename date", "error", err)
		return time.Time{}, fmt.Errorf("%w: %w", ErrInvalidFilename, err)
	}

	return d, nil
}
