package server

import (
	"flag"
	"strings"

	"github.com/caarlos0/env/v11"
)

const (
	defaultAddress  = "localhost:8080"
	defaultLogLevel = "info"
	logUsage        = "log level supported levels (info, warn, debug, error, panic, fatal)"
	addresUsage     = "server address eg localhost:8080"
)

type Config struct {
	Address  string `env:"ADDRESS"`
	LogLevel string `env:"LOG_LEVEL"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{
		Address:  defaultAddress,
		LogLevel: defaultLogLevel,
	}

	flag.StringVar(&cfg.Address, "a", defaultAddress, addresUsage)
	flag.StringVar(&cfg.LogLevel, "l", defaultLogLevel, logUsage)
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if strings.HasPrefix(cfg.Address, "http://") {
		cfg.Address = strings.Replace(cfg.Address, "http://", "", -1)
	}

	return cfg, nil
}
