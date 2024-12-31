package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbvehbq/go-metrics-harvester/internal/grpc"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric/service"
	"github.com/nbvehbq/go-metrics-harvester/internal/server"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage/memory"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage/postgres"
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

	cfg, err := server.NewConfig()

	if err != nil {
		log.Fatal(err, "Load config")
	}

	if errInit := logger.Initialize(cfg.LogLevel); errInit != nil {
		log.Fatal(errInit, "initialize logger")
	}

	ctx, done := context.WithCancel(context.Background())
	runner, ctx := errgroup.WithContext(ctx)

	var db service.Repository
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

	service := service.NewService(db)
	grpcServer, err := grpc.NewGrpc(ctx, runner, service, cfg)
	if err != nil {
		log.Fatal(err, "create grpc server")
	}
	if err := grpcServer.Run(ctx); err != nil {
		log.Fatal(err, "run grpc server")
	}

	httpServer, err := server.NewServer(runner, service, cfg)
	if err != nil {
		log.Fatal(err, "create http server")
	}
	httpServer.Run(ctx)

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

		<-stop
		done()

		nctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		grpcServer.Shutdown(nctx)

		if err := httpServer.Shutdown(nctx); err != nil {
			log.Fatal(err)
		}
	}()

	if err := runner.Wait(); err != nil {
		log.Printf("exit reason: %s \n", err)
	}
}
