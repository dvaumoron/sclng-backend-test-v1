package main

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	Port          int           `envconfig:"PORT" default:"5000"`
	EventApiUrl   string        `envconfig:"GITHUB_EVENT_API_URL" default:"https://api.github.com/events"`
	EventPageSize int           `envconfig:"GITHUB_EVENT_API_PAGE_SIZE" default:"100"` // 100 item per page is the max allowed by the API
	Refresh       time.Duration `envconfig:"REFRESH" default:"5m"`
	MaxCall       int           `envconfig:"MAX_CALL" default:"90"`               // github API accept 100 concurrent requests
	AccessToken   string        `envconfig:"GITHUB_ACCESS_TOKEN" required:"true"` // without it the API limit is 60 requests per hour
}

func newConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to build config from env")
	}
	return &cfg, nil
}
