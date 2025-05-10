package main

import (
	"context"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ListenAddress    string `env:"LISTEN_ADDRESS,notEmpty"  envDefault:":8080"`
	UploadAuthKey    string `env:"UPLOAD_AUTH_KEY,notEmpty"`
	UploadUserAgent  string `env:"UPLOAD_USER_AGENT"`
	RedirectToLatest bool   `env:"REDIRECT_TO_LATEST"       envDefault:"true"`

	S3Endpoint string `env:"S3_ENDPOINT,notEmpty"`
	S3Region   string `env:"S3_REGION"`
	S3Bucket   string `env:"S3_BUCKET,notEmpty"`

	UploadLimitRequests int           `env:"UPLOAD_LIMIT_REQUESTS,notEmpty" envDefault:"2"`
	UploadLimitWindow   time.Duration `env:"UPLOAD_LIMIT_WINDOW,notEmpty"   envDefault:"1m"`

	GetLimitRequests int           `env:"GET_LIMIT_REQUESTS,notEmpty" envDefault:"5"`
	GetLimitWindow   time.Duration `env:"GET_LIMIT_WINDOW,notEmpty"   envDefault:"10s"`
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
