package server

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nbvehbq/go-metrics-harvester/internal/compress"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"go.uber.org/zap"
)

type Repository interface {
	Set(context.Context, metric.Metric) error
	Get(context.Context, string) (metric.Metric, bool)
	List(context.Context) ([]metric.Metric, error)
	Persist(context.Context, io.Writer) error
	Ping(context.Context) error
	Update(context.Context, []metric.Metric) error
}

type Server struct {
	srv             *http.Server
	storage         Repository
	storeInterval   int64
	fileStoragePath string
	databaseDSN     string
}

func NewServer(storage Repository, cfg *Config) (*Server, error) {
	mux := chi.NewRouter()

	s := &Server{
		srv:             &http.Server{Addr: cfg.Address, Handler: mux},
		storage:         storage,
		storeInterval:   cfg.StoreInterval,
		fileStoragePath: cfg.FileStoragePath,
		databaseDSN:     cfg.DatabaseDSN,
	}

	mux.Get("/", logger.WithLogging(compress.WithGzip(s.listMetricHandler)))
	mux.Get("/ping", logger.WithLogging(s.pingDBHandler))
	mux.Post(`/update/`, logger.WithLogging(compress.WithGzip(s.updateHandlerJSON)))
	mux.Post(`/updates/`, logger.WithLogging(compress.WithGzip(s.updatesHandlerJSON)))
	mux.Post(`/value/`, logger.WithLogging(compress.WithGzip(s.getMetricHandlerJSON)))
	mux.Get("/value/{type}/{name}", logger.WithLogging(s.getMetricHandler))
	mux.Post(`/update/{type}/{name}/{value}`, logger.WithLogging(s.updateHandler))

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	logger.Log.Info("Server started.")

	storeInterval := time.Second * time.Duration(s.storeInterval)
	wait := make(chan struct{}, 1)

	if s.storeInterval > 0 {
		go func() {
			defer func() {
				wait <- struct{}{}
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(storeInterval):
					if err := s.saveToFile(ctx); err != nil {
						logger.Log.Error("save error", zap.Error(err))
					}
				}
			}
		}()
	}

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	<-wait

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info("Server stoped.")
	return s.srv.Shutdown(ctx)
}
