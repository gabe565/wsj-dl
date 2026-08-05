package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	authKey = "test-key"
	// paperPath redirects to a dated filename on the fake upstream.
	paperPath = "/todaysPaper"
	// issueKey and issueBody are what paperPath resolves to.
	issueKey  = "2026/08/05.pdf"
	issueBody = "/2026-08-05.pdf\n"
)

// fakeS3 records the object keys it receives via PUT.
type fakeS3 struct {
	mu   sync.Mutex
	keys []string
}

func (f *fakeS3) put(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append(f.keys, key)
}

func (f *fakeS3) Keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.keys
}

// newUpload wires the handler up to a fake upstream and a fake S3 backend, returning the
// handler, the recorded S3 keys, and the upstream base URL.
//
// The upstream redirects /todaysPaper to a dated filename, mirroring the real download flow.
func newUpload(t *testing.T) (http.HandlerFunc, *fakeS3, string) {
	t.Helper()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == paperPath:
			http.Redirect(w, r, "/files/a1b2-issue-8-5-2026.pdf", http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/files/"):
			_, _ = w.Write([]byte("%PDF-1.4 fake"))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	t.Cleanup(upstream.Close)

	keys := &fakeS3{}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			// Path is /<bucket>/<key>.
			_, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
			keys.put(key)
			w.Header().Set("ETag", `"abc123"`)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(backend.Close)

	u, err := url.Parse(backend.URL)
	require.NoError(t, err)

	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4("key", "secret", ""),
		Region: "us-east-1",
	})
	require.NoError(t, err)

	conf := &Config{
		S3Bucket:        "test-bucket",
		UploadAuthKey:   authKey,
		UploadUserAgent: "test-agent",
	}

	return uploadHandler(conf, client), keys, upstream.URL
}

func TestUploadHandler(t *testing.T) {
	tests := []struct {
		name string
		auth string
		// path is resolved against the fake upstream. rawURL is sent verbatim instead.
		path     string
		rawURL   string
		date     string
		wantCode int
		wantBody string
		wantKeys []string
	}{
		{
			name: "follows redirect and stores dated key", auth: authKey, path: paperPath,
			wantCode: http.StatusOK, wantBody: issueBody, wantKeys: []string{issueKey},
		},
		{
			name: "direct url", auth: authKey, path: "/files/a1b2-issue-8-5-2026.pdf",
			wantCode: http.StatusOK, wantBody: issueBody, wantKeys: []string{issueKey},
		},
		{
			name: "date param overrides unparseable filename", auth: authKey,
			path: "/files/paper.pdf", date: "2026-03-01",
			wantCode: http.StatusOK, wantBody: "/2026-03-01.pdf\n", wantKeys: []string{"2026/03/01.pdf"},
		},
		{
			name: "unparseable filename without date", auth: authKey, path: "/files/paper.pdf",
			wantCode: http.StatusBadRequest,
		},
		{
			name: "bad date param", auth: authKey, path: paperPath, date: "8-5-2026",
			wantCode: http.StatusBadRequest,
		},
		{
			name: "wrong auth key", auth: "nope", path: paperPath,
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "no auth key", path: paperPath,
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "missing url", auth: authKey,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "non-http scheme", auth: authKey, rawURL: "file:///etc/passwd",
			wantCode: http.StatusBadRequest,
		},
		{
			name: "upstream error", auth: authKey, path: "/gone",
			wantCode: http.StatusBadGateway,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, keys, upstream := newUpload(t)

			q := make(url.Values)
			switch {
			case tt.rawURL != "":
				q.Set("url", tt.rawURL)
			case tt.path != "":
				q.Set("url", upstream+tt.path)
			}
			if tt.date != "" {
				q.Set("date", tt.date)
			}

			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/upload?"+q.Encode(), nil)
			if tt.auth != "" {
				r.Header.Set("Authorization", tt.auth)
			}
			w := httptest.NewRecorder()

			handler(w, r)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
			assert.Equal(t, tt.wantKeys, keys.Keys())
		})
	}
}

func TestUploadHandler_post(t *testing.T) {
	handler, keys, upstream := newUpload(t)

	body := url.Values{"url": {upstream + paperPath}}.Encode()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/upload", strings.NewReader(body))
	r.Header.Set("Authorization", authKey)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, issueBody, w.Body.String())
	assert.Equal(t, []string{issueKey}, keys.Keys())
}

func TestStoreLatest(t *testing.T) {
	t.Cleanup(func() { latest.Store(nil) })

	newer := NewIssueFromDate(time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), ".pdf")
	older := NewIssueFromDate(time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC), ".pdf")

	latest.Store(nil)
	storeLatest(older)
	assert.Equal(t, older, latest.Load())

	storeLatest(newer)
	assert.Equal(t, newer, latest.Load(), "should advance to a newer issue")

	storeLatest(older)
	assert.Equal(t, newer, latest.Load(), "should not regress to an older issue")
}
