package main

import (
	"context"
	"time"

	"github.com/caarlos0/env/v11"
)

//go:generate go tool envdoc -output config.md
type Config struct {
	// The address to listen for HTTP requests on.
	ListenAddress string `env:"LISTEN_ADDRESS,notEmpty" envDefault:":8080"`
	// Redirect requests to `/` to the latest PDF.
	RedirectToLatest bool `env:"REDIRECT_TO_LATEST" envDefault:"true"`

	// S3-compatible API endpoint.
	S3Endpoint string `env:"S3_ENDPOINT,notEmpty"`
	// S3 region.
	S3Region string `env:"S3_REGION"`
	// S3 bucket name.
	S3Bucket string `env:"S3_BUCKET,notEmpty"`

	// Authorization key for the `/api/upload` endpoint.
	UploadAuthKey string `env:"UPLOAD_AUTH_KEY,notEmpty"`
	// User agent to use when fetching a new PDF. Will be loaded from https://github.com/jnrbsn/user-agents if empty.
	UploadUserAgent string `env:"UPLOAD_USER_AGENT"`

	// CIDR ranges of reverse proxies whose X-Forwarded-For headers are trusted
	TrustedProxies []string `env:"TRUSTED_PROXIES"`
	// HTTP rate limit requests.
	LimitRequests int `env:"LIMIT_REQUESTS,notEmpty" envDefault:"30"`
	// HTTP rate limit window.
	LimitWindow time.Duration `env:"LIMIT_WINDOW,notEmpty" envDefault:"15s"`
}

func Load() (*Config, error) {
	c, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}

	if c.UploadUserAgent == "" {
		var err error
		c.UploadUserAgent, err = LoadUserAgent(context.TODO())
		if err != nil {
			return nil, err
		}
	}

	return &c, nil
}
