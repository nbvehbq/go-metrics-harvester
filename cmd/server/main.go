package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/server"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage"
)

func main() {
	cfg, err := server.NewConfig()

	if err != nil {
		log.Fatal(err, "Load config")
	}

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatal(err, "initialize logger")
	}

	db := storage.NewMemStorage()
	if cfg.Restore {
		file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err, "open storage file")
		}
		fi, err := file.Stat()
		if err != nil {
			log.Fatal(err, "could not obtain stat")
		}

		if fi.Size() > 0 {
			db, err = storage.NewFrom(file)
			if err != nil {
				log.Fatal(err, " restor storage from file")
			}
		}
	}

	server, err := server.NewServer(db, cfg)
	if err != nil {
		log.Fatal(err, "create server")
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

		<-stop

		nctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		if err := server.Shutdown(nctx); err != nil {
			log.Fatal(err)
		}
	}()

	if err := server.Run(ctx); err != nil {
		panic(err)
	}
}
