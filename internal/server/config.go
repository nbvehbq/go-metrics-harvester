package server

import (
	"flag"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

const (
	defaultAddress       = "localhost:8080"
	defaultLogLevel      = "info"
	defaultStoreInterval = 300
	defaultRestore       = true

	logUsage           = "log level (default 'info')"
	addresUsage        = "server address (default localhost:8080)"
	storeIntervalUsage = "metrics saving interval (default 300 seconds) 0 means saving synchronously"
	storePathUsage     = "metric db filename path"
	restoreUsage       = "restore metrics at start (default true)"
	databaseUsage      = "database DSN string eg 'postgresql://user:password@localhost:5432/dbname'"
)

type Config struct {
	Address         string `env:"ADDRESS"`
	LogLevel        string `env:"LOG_LEVEL"`
	StoreInterval   int64  `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{
		Address:  defaultAddress,
		LogLevel: defaultLogLevel,
	}

	flag.StringVar(&cfg.Address, "a", defaultAddress, addresUsage)
	flag.StringVar(&cfg.LogLevel, "l", defaultLogLevel, logUsage)
	flag.Int64Var(&cfg.StoreInterval, "i", defaultStoreInterval, storeIntervalUsage)
	flag.StringVar(&cfg.FileStoragePath, "f", "", storePathUsage)
	flag.BoolVar(&cfg.Restore, "r", defaultRestore, restoreUsage)
	flag.StringVar(&cfg.DatabaseDSN, "d", "", databaseUsage)
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if strings.HasPrefix(cfg.Address, "http://") {
		cfg.Address = strings.Replace(cfg.Address, "http://", "", -1)
	}

	if cfg.FileStoragePath == "" {
		f, err := os.CreateTemp("", "metric")
		if err != nil {
			return nil, err
		}
		cfg.FileStoragePath = f.Name()
		if err := f.Close(); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
