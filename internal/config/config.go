package config

import (
	"flag"
	"net/url"
)

type Agent struct {
	DSN            string
	ReportInterval int64
	PollInterval   int64
}

type Server struct {
	DSN string
}

type Config struct {
	Agent  *Agent
	Server *Server
}

func NewConfig() (*Config, error) {
	var server Server
	var agent Agent
	var dsn string

	flag.StringVar(&dsn, "a", "localhost:8080", "server address eg http://localhost:8080")
	flag.Int64Var(&agent.ReportInterval, "r", 10, "send report interval default 10 seconds")
	flag.Int64Var(&agent.PollInterval, "p", 2, "request metric poll interval default 2 seconds")
	flag.Parse()

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	server.DSN = dsn
	if u.Scheme == "localhost" || u.Scheme == "127.0.0.1" {
		dsn = "http://" + dsn
	}
	agent.DSN = dsn

	c := &Config{
		Server: &server,
		Agent:  &agent,
	}

	return c, nil
}
