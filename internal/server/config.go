package server

import (
	"flag"
	"strings"

	"github.com/caarlos0/env/v11"
)

const (
	defaultAddress = "localhost:8080"
)

type Config struct {
	Address string `env:"ADDRESS"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{
		Address: defaultAddress,
	}

	flag.StringVar(&cfg.Address, "a", defaultAddress, "server address eg localhost:8080")
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if strings.HasPrefix(cfg.Address, "http://") {
		cfg.Address = strings.Replace(cfg.Address, "http://", "", -1)
	}

	return cfg, nil
}
