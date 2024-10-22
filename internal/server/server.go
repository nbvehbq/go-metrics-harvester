package server

import (
	"context"
	"io"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddle "github.com/go-chi/chi/v5/middleware"
	"github.com/nbvehbq/go-metrics-harvester/internal/compress"
	"github.com/nbvehbq/go-metrics-harvester/internal/hash"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/middleware"
	"go.uber.org/zap"
)

// Repository is a metrics repository interface
type Repository interface {
	Set(context.Context, metric.Metric) error
	Get(context.Context, string) (metric.Metric, bool)
	List(context.Context) ([]metric.Metric, error)
	Persist(context.Context, io.Writer) error
	Ping(context.Context) error
	Update(context.Context, []metric.Metric) error
}

// Server is a metrics server
type Server struct {
	srv             *http.Server
	storage         Repository
	storeInterval   int64
	fileStoragePath string
	databaseDSN     string
}

// NewServer creates a new server
func NewServer(storage Repository, cfg *Config) (*Server, error) {
	mux := chi.NewRouter()

	s := &Server{
		srv:             &http.Server{Addr: cfg.Address, Handler: mux},
		storage:         storage,
		storeInterval:   cfg.StoreInterval,
		fileStoragePath: cfg.FileStoragePath,
		databaseDSN:     cfg.DatabaseDSN,
	}

	mdw := []middleware.Middleware{
		hash.WithHash(cfg.Key),
		compress.WithGzip,
		logger.WithLogging,
	}

	mux.Get(`/`, middleware.Combine(s.listMetricHandler, mdw...))
	mux.Get(`/ping`, logger.WithLogging(s.pingDBHandler))
	mux.Post(`/update/`, middleware.Combine(s.updateHandlerJSON, mdw...))
	mux.Post(`/updates/`, middleware.Combine(s.updatesHandlerJSON, mdw...))
	mux.Post(`/value/`, middleware.Combine(s.getMetricHandlerJSON, mdw...))
	mux.Get(`/value/{type}/{name}`, logger.WithLogging(s.getMetricHandler))
	mux.Post(`/update/{type}/{name}/{value}`, logger.WithLogging(s.updateHandler))

	mux.Mount("/debug", chimiddle.Profiler())

	return s, nil
}

// Run runs the server
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

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info("Server stoped.")
	return s.srv.Shutdown(ctx)
}
