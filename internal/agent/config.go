package agent

import (
	"flag"
	"net/url"

	"github.com/caarlos0/env/v11"
)

const (
	defaultAddress        = "http://localhost:8080"
	defaultReportInterval = 10
	defaultPollInterval   = 2
)

type Config struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int64  `env:"REPORT_INTERVAL"`
	PollInterval   int64  `env:"POLL_INTERVAL"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{
		Address:        defaultAddress,
		ReportInterval: defaultReportInterval,
		PollInterval:   defaultPollInterval,
	}

	flag.StringVar(&cfg.Address, "a", defaultAddress, "server address eg http://localhost:8080")
	flag.Int64Var(&cfg.ReportInterval, "r", 10, "send report interval default 10 seconds")
	flag.Int64Var(&cfg.PollInterval, "p", 2, "request metric poll interval default 2 seconds")
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	u, err := url.Parse(cfg.Address)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "localhost" || u.Scheme == "127.0.0.1" {
		cfg.Address = "http://" + cfg.Address
	}

	return cfg, nil
}
