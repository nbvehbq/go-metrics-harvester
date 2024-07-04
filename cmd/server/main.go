package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbvehbq/go-metrics-harvester/internal/config"
	"github.com/nbvehbq/go-metrics-harvester/internal/server"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		panic(err)
	}

	db := storage.NewMemStorage()
	server := server.NewServer(db, cfg)

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

		<-stop

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			panic(err)
		}
	}()

	if err := server.Run(); err != nil {
		panic(err)
	}
}
