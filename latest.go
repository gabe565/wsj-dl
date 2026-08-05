package main

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
)

//nolint:gochecknoglobals
var latest atomic.Pointer[Issue]

// storeLatest sets issue as the latest issue, unless a newer one is already stored.
func storeLatest(issue *Issue) {
	for {
		curr := latest.Load()
		if curr != nil && !issue.Date.After(curr.Date) {
			return
		}
		if latest.CompareAndSwap(curr, issue) {
			return
		}
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
