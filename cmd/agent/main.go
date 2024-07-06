package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nbvehbq/go-metrics-harvester/internal/agent"
	"golang.org/x/sync/errgroup"
)

func main() {
	cfg, err := agent.NewConfig()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner, ctx := errgroup.WithContext(ctx)

	agent := agent.NewAgent(runner, cfg)
	agent.Run(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	cancel()

	runner.Wait()
	log.Println("Agent stoped.")
}
