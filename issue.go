package main

import (
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"
)

var ErrInvalidFilename = errors.New("invalid filename")

func NewIssueFromUpstream(p string) (*Issue, error) {
	ext := path.Ext(p)
	p = path.Base(strings.TrimSuffix(p, ext))
	log := slog.With("filename", p)

	_, p, ok := strings.Cut(p, "-")
	if !ok {
		log.Warn("Filename missing random prefix")
		return nil, fmt.Errorf("%w: %s", ErrInvalidFilename, "missing random prefix")
	}

	_, p, ok = strings.Cut(p, "-")
	if !ok {
		log.Warn("Filename missing non-random prefix")
		return nil, fmt.Errorf("%w: %s", ErrInvalidFilename, "missing non-random prefix")
	}

	d, err := time.Parse("1-2-2006", p)
	if err != nil {
		log.Warn("Failed to parse filename date", "error", err)
		return nil, fmt.Errorf("%w: %w", ErrInvalidFilename, err)
	}

	return &Issue{Date: d, Ext: ext}, nil
}

func NewIssueFromPath(p string) (*Issue, error) {
	ext := path.Ext(p)
	p = path.Base(strings.TrimSuffix(p, ext))

	d, err := time.Parse("2006-01-02", p)
	if err != nil {
		return nil, err
	}

	return &Issue{Date: d, Ext: ext}, nil
}

func NewIssueFromDate(date time.Time, ext string) *Issue {
	return &Issue{Date: date, Ext: ext}
}

type Issue struct {
	Date time.Time
	Ext  string
}

func (i Issue) FullPath() string {
	return i.Date.Format("2006/01/02") + i.Ext
}

func (i Issue) ShortPath() string {
	return i.Date.Format("2006-01-02") + i.Ext
}

func (i Issue) String() string {
	return i.ShortPath()
}
