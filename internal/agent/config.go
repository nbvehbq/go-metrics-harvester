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
	defaultLogLevel       = "info"
	defaultRateLimit      = 1024
)

// Config is an agent configuration
type Config struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int64  `env:"REPORT_INTERVAL"`
	PollInterval   int64  `env:"POLL_INTERVAL"`
	LogLevel       string `env:"LOG_LEVEL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
	CryptoKey      string `env:"CRYPTO_KEY"`
}

// NewConfig returns a new config
func NewConfig() (*Config, error) {
	cfg := &Config{
		Address:        defaultAddress,
		ReportInterval: defaultReportInterval,
		PollInterval:   defaultPollInterval,
		LogLevel:       defaultLogLevel,
	}

	flag.StringVar(&cfg.Address, "a", defaultAddress, "server address eg http://localhost:8080")
	flag.Int64Var(&cfg.ReportInterval, "r", 10, "send report interval default 10 seconds")
	flag.Int64Var(&cfg.PollInterval, "p", 2, "request metric poll interval default 2 seconds")
	flag.StringVar(&cfg.Key, "k", "", "secret key")
	flag.IntVar(&cfg.RateLimit, "l", defaultRateLimit, "requests limit default 1024")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "public key")
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
