package server

import (
	"encoding/json"
	"flag"
	"net"
	"os"
	"strings"
	"time"

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

type CfgFile struct {
	Address       string `json:"address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	TrustedSubnet string `json:"trusted_subnet"`
}

// Config is a server configuration
type Config struct {
	Address         string `env:"ADDRESS"`
	LogLevel        string `env:"LOG_LEVEL"`
	StoreInterval   int64  `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
	CryptoKey       string `env:"CRYPTO_KEY"`
	ConfigFile      string `env:"CONFIG"`
	TrustedSubnet   string `env:"TRUSTED_SUBNET"`
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
	flag.StringVar(&cfg.Key, "k", "", "secret key")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "secret assymetric key")
	flag.StringVar(&cfg.ConfigFile, "c", "", "json file holding configuration")
	flag.StringVar(&cfg.TrustedSubnet, "t", "", "trust subnet CIDR notation")
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if cfg.TrustedSubnet != "" {
		_, _, err := net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			return nil, err
		}
	}

	if cfg.ConfigFile != "" {
		file, err := os.Open(cfg.ConfigFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		var fileCfg CfgFile
		if err := json.NewDecoder(file).Decode(&fileCfg); err != nil {
			return nil, err
		}

		si, err := time.ParseDuration(fileCfg.StoreInterval)
		if err != nil {
			return nil, err
		}

		cfg.Address = fileCfg.Address
		cfg.Restore = fileCfg.Restore
		cfg.StoreInterval = int64(si.Seconds())
		cfg.FileStoragePath = fileCfg.StoreFile
		cfg.DatabaseDSN = fileCfg.DatabaseDSN
		cfg.CryptoKey = fileCfg.CryptoKey
	}

	if strings.HasPrefix(cfg.Address, "http://") {
		cfg.Address = strings.Replace(cfg.Address, "http://", "", -1)
	}

	if cfg.Restore && cfg.FileStoragePath == "" {
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
