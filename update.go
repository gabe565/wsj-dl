package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
)

//nolint:gochecknoglobals
var latest atomic.Pointer[string]

func update(ctx context.Context, conf *Config, s3 *minio.Client) (string, error) {
	u, err := url.Parse(conf.UpdateURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", conf.UpdateUserAgent)

	filename := filepath.Base(u.String())
	flatFilename := filename
	var exists bool

	client := &http.Client{
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			u = req.URL
			filename = filepath.Base(u.String())
			flatFilename = filename

			if d, err := getDate(filename); err == nil {
				ext := filepath.Ext(filename)
				filename = d.Format("2006/01/02") + ext
				flatFilename = d.Format("2006-01-02") + ext
			}

			if _, err := s3.StatObject(ctx, conf.S3Bucket, filename, minio.StatObjectOptions{}); err == nil {
				exists = true
				return http.ErrUseLastResponse
			}

			return nil
		},
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		if exists {
			slog.Info("File already exists", "filename", flatFilename, "url", u.String())
			return flatFilename, nil
		}
		return "", fmt.Errorf("%w: %s", ErrUpstream, res.Status)
	}

	_, err = s3.PutObject(ctx, conf.S3Bucket, filename, res.Body, res.ContentLength, minio.PutObjectOptions{
		ContentType:        res.Header.Get("Content-Type"),
		ContentDisposition: "attachment; filename=" + flatFilename,
	})
	if err != nil {
		return "", err
	}

	slog.Info("Loaded file", "filename", flatFilename, "url", u.String())
	latest.Store(&flatFilename)

	return flatFilename, nil
}

func updateHandler(conf *Config, s3 *minio.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != conf.UpdateAuthKey {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		name, err := update(r.Context(), conf, s3)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "../"+name, http.StatusTemporaryRedirect)
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

func findLatest(ctx context.Context, conf *Config, s3 *minio.Client) (string, error) {
	// Fast path for today
	now := time.Now()
	if _, err := s3.StatObject(ctx, conf.S3Bucket, now.Format("2006/01/02.pdf"),
		minio.StatObjectOptions{},
	); err == nil {
		return now.Format("2006-01-02.pdf"), nil
	}

	// Fast path for yesterday
	now = now.AddDate(0, 0, -1)
	if _, err := s3.StatObject(ctx, conf.S3Bucket, now.Format("2006/01/02.pdf"),
		minio.StatObjectOptions{},
	); err == nil {
		return now.Format("2006-01-02.pdf"), nil
	}

	// Slow path
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var latestName string
	var latest time.Time
	for item := range s3.ListObjectsIter(ctx, conf.S3Bucket, minio.ListObjectsOptions{
		Prefix:    "20",
		Recursive: true,
	}) {
		if item.Err != nil {
			return "", item.Err
		}

		d, err := time.Parse("2006/01/02.pdf", item.Key)
		if err != nil {
			continue
		}

		if d.After(latest) {
			latestName = item.Key
			latest = d
		}
	}

	latestName = strings.ReplaceAll(latestName, "/", "-")
	return latestName, nil
}
