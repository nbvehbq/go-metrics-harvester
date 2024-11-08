package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/server"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage/memory"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage/postgres"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	greetings := "Build version: %s\nBuild date: %s\nBuild commit: %s\n\n"
	fmt.Printf(greetings, buildVersion, buildDate, buildCommit)

	cfg, err := server.NewConfig()

	if err != nil {
		log.Fatal(err, "Load config")
	}

	if errInit := logger.Initialize(cfg.LogLevel); errInit != nil {
		log.Fatal(errInit, "initialize logger")
	}

	ctx, cancel := context.WithCancel(context.Background())

	var db server.Repository
	if cfg.DatabaseDSN == "" {
		db = memory.NewMemStorage()
	} else {
		db, err = postgres.NewStorage(ctx, cfg.DatabaseDSN)
		if err != nil {
			log.Fatal(err, "connect to db")
		}
	}

	if cfg.Restore {
		file, errOpen := os.OpenFile(cfg.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
		if errOpen != nil {
			log.Fatal(errOpen, "open storage file")
		}
		fi, errOpen := file.Stat()
		if errOpen != nil {
			log.Fatal(errOpen, "could not obtain stat")
		}

		if fi.Size() > 0 {
			if cfg.DatabaseDSN == "" {
				db, errOpen = memory.NewFrom(file)
				if errOpen != nil {
					log.Fatal(errOpen, " restor storage from file")
				}
			} else {
				db, errOpen = postgres.NewFrom(ctx, file, cfg.DatabaseDSN)
				if errOpen != nil {
					log.Fatal(errOpen, " restor storage from file")
				}
			}
		}
	}

	server, err := server.NewServer(db, cfg)
	if err != nil {
		log.Fatal(err, "create server")
	}

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
