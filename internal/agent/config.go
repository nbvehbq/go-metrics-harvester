package agent

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
)

type Protocol string

const (
	HTTPProtocol Protocol = "http"
	GRPCProtocol Protocol = "grpc"
)

const (
	defaultAddress        = "localhost:8080"
	defaultReportInterval = 10
	defaultPollInterval   = 2
	defaultLogLevel       = "info"
	defaultRateLimit      = 1024
	defaultProtocol       = HTTPProtocol
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
	ConfigFile     string `env:"CONFIG"`
	Protocol       string `env:"PROTOCOL"`
}

type CfgFile struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
	Protocol       string `json:"protocol"`
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
	flag.StringVar(&cfg.ConfigFile, "c", "", "json file holding configuration")
	flag.StringVar(&cfg.Protocol, "protocol", string(defaultProtocol), "protocol to comunicate with server")
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
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

		ri, err := time.ParseDuration(fileCfg.ReportInterval)
		if err != nil {
			return nil, err
		}
		pi, err := time.ParseDuration(fileCfg.PollInterval)
		if err != nil {
			return nil, err
		}

		cfg.Address = fileCfg.Address
		cfg.ReportInterval = int64(ri.Seconds())
		cfg.PollInterval = int64(pi.Seconds())
		cfg.CryptoKey = fileCfg.CryptoKey
	}

	u, err := url.Parse(cfg.Address)
	if err != nil {
		return nil, err
	}

	if cfg.Protocol != string(HTTPProtocol) && cfg.Protocol != string(GRPCProtocol) {
		return nil, fmt.Errorf("unknown protocol %s", cfg.Protocol)
	}

	if cfg.Protocol == string(HTTPProtocol) && (u.Scheme == "localhost" || u.Scheme == "127.0.0.1") {
		cfg.Address = "http://" + cfg.Address
	}

	return cfg, nil
}
