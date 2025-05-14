package main

import (
	"context"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ListenAddress    string `env:"LISTEN_ADDRESS,notEmpty" envDefault:":8080"`
	RedirectToLatest bool   `env:"REDIRECT_TO_LATEST"      envDefault:"true"`

	S3Endpoint string `env:"S3_ENDPOINT,notEmpty"`
	S3Region   string `env:"S3_REGION"`
	S3Bucket   string `env:"S3_BUCKET,notEmpty"`

	UpdateCron      string `env:"UPDATE_CRON"         envDefault:"0 8 * * 1-6"`
	UpdateAuthKey   string `env:"UPDATE_AUTH_KEY"`
	UpdateURL       string `env:"UPDATE_URL,notEmpty"`
	UpdateUserAgent string `env:"UPDATE_USER_AGENT"`

	UpdateLimitRequests int           `env:"UPDATE_LIMIT_REQUESTS,notEmpty" envDefault:"2"`
	UpdateLimitWindow   time.Duration `env:"UPDATE_LIMIT_WINDOW,notEmpty"   envDefault:"1m"`

	GetLimitRequests int           `env:"GET_LIMIT_REQUESTS,notEmpty" envDefault:"5"`
	GetLimitWindow   time.Duration `env:"GET_LIMIT_WINDOW,notEmpty"   envDefault:"10s"`
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
