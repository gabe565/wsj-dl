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
	RedirectToLatest bool `env:"REDIRECT_TO_LATEST"      envDefault:"true"`

	// S3-compatible API endpoint.
	S3Endpoint string `env:"S3_ENDPOINT,notEmpty"`
	// S3 region.
	S3Region string `env:"S3_REGION"`
	// S3 bucket name.
	S3Bucket string `env:"S3_BUCKET,notEmpty"`

	// Checks for an updated PDF on startup.
	UpdateOnStartup bool `env:"UPDATE_ON_STARTUP"   envDefault:"true"`
	// Configures the update cron interval. Leave blank to disable.
	UpdateCron string `env:"UPDATE_CRON"         envDefault:"0 8 * * 1-6"`
	// Authorization key for the `/api/update` endpoint. Leave blank to disable this endpoint.
	UpdateAuthKey string `env:"UPDATE_AUTH_KEY"`
	// URL to fetch PDFs from.
	UpdateURL string `env:"UPDATE_URL,notEmpty"`
	// User agent to use when fetching a new PDF. Will be loaded from https://github.com/jnrbsn/user-agents if empty.
	UpdateUserAgent string `env:"UPDATE_USER_AGENT"`

	// HTTP rate limit requests.
	LimitRequests int `env:"LIMIT_REQUESTS,notEmpty" envDefault:"5"`
	// HTTP rate limit window.
	LimitWindow time.Duration `env:"LIMIT_WINDOW,notEmpty"   envDefault:"10s"`
}

func Load() (*Config, error) {
	c, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}

	if c.UpdateUserAgent == "" {
		var err error
		c.UpdateUserAgent, err = LoadUserAgent(context.TODO())
		if err != nil {
			return nil, err
		}
	}

	return &c, nil
}
