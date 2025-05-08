package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
)

func get(conf *Config, s3 *minio.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := chi.URLParam(r, "*")
		filename = strings.ReplaceAll(filename, "-", "/")

		obj, err := s3.GetObject(r.Context(), conf.S3Bucket, filename, minio.GetObjectOptions{})
		if err != nil {
			handleMinioError(w, err)
			return
		}
		defer func() {
			_ = obj.Close()
		}()

		stat, err := obj.Stat()
		if err != nil {
			handleMinioError(w, err)
			return
		}

		if v := stat.ETag; v != "" {
			w.Header().Set("ETag", strconv.Quote(v))
		}

		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeContent(w, r, filename, stat.LastModified, obj)
	}
}

func handleMinioError(w http.ResponseWriter, err error) {
	if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	} else {
		handleHTTPError(w, err.Error(), http.StatusInternalServerError)
	}
}
