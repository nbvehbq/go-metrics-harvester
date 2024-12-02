package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nbvehbq/go-metrics-harvester/internal/agent"
	"github.com/nbvehbq/go-metrics-harvester/internal/grpclient"
	"github.com/nbvehbq/go-metrics-harvester/internal/httpclient"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"golang.org/x/sync/errgroup"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	greetings := "Build version: %s\nBuild date: %s\nBuild commit: %s\n\n"
	fmt.Printf(greetings, buildVersion, buildDate, buildCommit)

	cfg, err := agent.NewConfig()
	if err != nil {
		log.Fatal(err, "Load config error")
	}

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatal(err, "initialize logger")
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner, ctx := errgroup.WithContext(ctx)

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

		<-stop
		cancel()
	}()

	var client agent.Publisher
	if cfg.Protocol == string(agent.HTTPProtocol) {
		client, err = httpclient.NewHTTPClient(cfg)
		if err != nil {
			log.Fatal(err, "initialize http client")
		}
	}

	if cfg.Protocol == string(agent.GRPCProtocol) {
		client, err = grpclient.NewGRPClient(cfg)
		if err != nil {
			log.Fatal(err, "initialize grpc client")
		}
	}

	agent, err := agent.NewAgent(runner, cfg, client)
	if err != nil {
		log.Fatal(err, "initialize agent")
	}
	agent.Run(ctx)

	if err := runner.Wait(); err != nil {
		log.Printf("exit reason: %s \n", err)
	}
}
