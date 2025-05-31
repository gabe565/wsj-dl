package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
)

//nolint:gochecknoglobals
var latest atomic.Pointer[Issue]

func update(ctx context.Context, conf *Config, s3 *minio.Client, force bool) (*Issue, error) {
	if !force {
		if issue := latest.Load(); issue != nil {
			y1, m1, d1 := issue.Date.Date()
			y2, m2, d2 := time.Now().Date()
			if y1 == y2 && m1 == m2 && d1 == d2 {
				slog.Info("Latest issue is already downloaded", "filename", issue)
				return issue, nil
			}
		}
	}

	u, err := url.Parse(conf.UpdateURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", conf.UpdateUserAgent)

	var issue *Issue
	var exists bool

	client := &http.Client{
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			u = req.URL

			issue, err = NewIssueFromUpstream(u.Path)
			if err != nil {
				return err
			}

			if !force {
				if _, err := s3.StatObject(ctx, conf.S3Bucket, issue.FullPath(), minio.StatObjectOptions{}); err == nil {
					exists = true
					return http.ErrUseLastResponse
				}
			}

			return nil
		},
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		if exists {
			slog.Info("File already exists", "filename", issue, "url", u.String())
			latest.Store(issue)
			return issue, nil
		}
		return nil, fmt.Errorf("%w: %s", ErrUpstream, res.Status)
	}

	_, err = s3.PutObject(ctx, conf.S3Bucket, issue.FullPath(), res.Body, res.ContentLength, minio.PutObjectOptions{
		ContentType:        res.Header.Get("Content-Type"),
		ContentDisposition: "attachment; filename=" + issue.ShortPath(),
	})
	if err != nil {
		return nil, err
	}

	slog.Info("Loaded file", "filename", issue, "url", u.String())
	latest.Store(issue)

	return issue, nil
}

func updateHandler(conf *Config, s3 *minio.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != conf.UpdateAuthKey {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		issue, err := update(r.Context(), conf, s3, r.URL.Query().Has("force"))
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "../"+issue.ShortPath(), http.StatusTemporaryRedirect)
	}
}

func findLatest(ctx context.Context, conf *Config, s3 *minio.Client) (*Issue, error) {
	// Fast path for today
	now := time.Now()
	if _, err := s3.StatObject(ctx, conf.S3Bucket, now.Format("2006/01/02.pdf"),
		minio.StatObjectOptions{},
	); err == nil {
		return NewIssueFromDate(now, ".pdf"), nil
	}

	// Fast path for yesterday
	now = now.AddDate(0, 0, -1)
	if _, err := s3.StatObject(ctx, conf.S3Bucket, now.Format("2006/01/02.pdf"),
		minio.StatObjectOptions{},
	); err == nil {
		return NewIssueFromDate(now, ".pdf"), nil
	}

	// Slow path
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var latest time.Time
	for item := range s3.ListObjectsIter(ctx, conf.S3Bucket, minio.ListObjectsOptions{
		Prefix:    "20",
		Recursive: true,
	}) {
		if item.Err != nil {
			return nil, item.Err
		}

		d, err := time.Parse("2006/01/02.pdf", item.Key)
		if err != nil {
			continue
		}

		if d.After(latest) {
			latest = d
		}
	}

	return NewIssueFromDate(latest, ".pdf"), nil
}
