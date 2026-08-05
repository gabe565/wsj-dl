package main

import (
	"crypto/subtle"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/minio/minio-go/v7"
)

// defaultExt is used when the download URL has no file extension.
const defaultExt = ".pdf"

// uploadHandler downloads the PDF at the `url` param and stores it in S3.
//
// The issue date is parsed from the filename of the final URL after redirects. An optional
// `date` param in YYYY-MM-DD format overrides it for URLs that don't follow that format.
func uploadHandler(conf *Config, s3 *minio.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare(
			[]byte(r.Header.Get("Authorization")), []byte(conf.UploadAuthKey),
		) != 1 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		src := r.FormValue("url")
		if src == "" {
			handleHTTPError(w, "Missing url", http.StatusBadRequest)
			return
		}

		u, err := url.Parse(src)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			handleHTTPError(w, "URL scheme must be http or https", http.StatusBadRequest)
			return
		}

		var date time.Time
		if v := r.FormValue("date"); v != "" {
			if date, err = time.Parse(time.DateOnly, v); err != nil {
				handleHTTPError(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.Header.Set("User-Agent", conf.UploadUserAgent)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
		}()

		if res.StatusCode != http.StatusOK {
			handleHTTPError(w, fmt.Errorf("%w: %s", ErrUpstream, res.Status).Error(), http.StatusBadGateway)
			return
		}

		// Request is the last request in the redirect chain, so its URL holds the real filename.
		u = res.Request.URL

		var issue *Issue
		if date.IsZero() {
			if issue, err = NewIssueFromUpstream(u.Path); err != nil {
				handleHTTPError(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			ext := path.Ext(u.Path)
			if ext == "" {
				ext = defaultExt
			}
			issue = NewIssueFromDate(date, ext)
		}

		_, err = s3.PutObject(r.Context(), conf.S3Bucket, issue.FullPath(), res.Body, res.ContentLength,
			minio.PutObjectOptions{
				ContentType:        res.Header.Get("Content-Type"),
				ContentDisposition: "attachment; filename=" + issue.ShortPath(),
			},
		)
		if err != nil {
			handleHTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("Loaded file", "filename", issue, "url", u.String())
		storeLatest(issue)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "/"+issue.ShortPath()+"\n")
	}
}
