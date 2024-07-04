package config

import (
	"flag"
	"net/url"
)

type Agent struct {
	ReportInterval int64
	PollInterval   int64
}

type Config struct {
	// DSN is server address eg http://localhost:8080
	DSN   string
	Agent *Agent
}

func NewConfig() (*Config, error) {
	var dsn string
	var agent Agent

	flag.StringVar(&dsn, "a", "http://localhost:8080", "server address eg http://localhost:8080")
	flag.Int64Var(&agent.ReportInterval, "r", 10, "send report interval default 10 seconds")
	flag.Int64Var(&agent.PollInterval, "p", 2, "request metric poll interval default 2 seconds")
	flag.Parse()

	u, err := url.Parse(dsn)
	if err!= nil {
		return nil, err
	}

	if u.Scheme == "localhost" || u.Scheme == "127.0.0.1" {
		dsn = "http://" + dsn
	}

	c := &Config{
		DSN:   dsn,
		Agent: &agent,
	}

	return c, nil
}
