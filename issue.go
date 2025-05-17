package main

import (
	"errors"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func NewIssueFromPath(p string) (*Issue, error) {
	p = path.Base(p)

	d, err := getDate(p)
	if err != nil {
		return nil, err
	}

	return &Issue{Date: d, Ext: filepath.Ext(p)}, nil
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

var ErrInvalidFilename = errors.New("invalid filename")

func getDate(filename string) (time.Time, error) {
	log := slog.With("filename", filename)

	filename = path.Base(filename)
	ext := filepath.Ext(filename)
	filename = strings.TrimSuffix(filename, ext)

	d, err := time.Parse("2006-01-02", filename)
	if err != nil {
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

		d, err = time.Parse("1-2-2006", filename)
		if err != nil {
			log.Warn("Failed to parse filename date", "error", err)
			return time.Time{}, fmt.Errorf("%w: %w", ErrInvalidFilename, err)
		}
	}

	return d, nil
}
