package main

import (
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewS3(conf *Config) (*minio.Client, error) {
	u, err := url.Parse(conf.S3Endpoint)
	if err != nil {
		return nil, err
	}

	opts := &minio.Options{
		Creds: credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvAWS{},
			&credentials.IAM{},
		}),
		Secure: u.Scheme == "https",
		Region: conf.S3Region,
	}

	return minio.New(u.Host, opts)
}
