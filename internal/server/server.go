package server

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddle "github.com/go-chi/chi/v5/middleware"
	"github.com/nbvehbq/go-metrics-harvester/internal/compress"
	"github.com/nbvehbq/go-metrics-harvester/internal/crypto"
	"github.com/nbvehbq/go-metrics-harvester/internal/hash"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/middleware"
	"github.com/nbvehbq/go-metrics-harvester/internal/subnet"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Server is a metrics server
type Server struct {
	srv             *http.Server
	runner          *errgroup.Group
	service         metric.MetricService
	storeInterval   int64
	fileStoragePath string
}

// NewServer creates a new server
func NewServer(runner *errgroup.Group, service metric.MetricService, cfg *Config) (*Server, error) {
	var (
		buf []byte
		err error
	)
	if cfg.CryptoKey != "" {
		buf, err = os.ReadFile(cfg.CryptoKey)
		if err != nil {
			return nil, errors.Wrap(err, "open private key filename")
		}
	}

	mux := chi.NewRouter()

	s := &Server{
		srv:             &http.Server{Addr: cfg.Address, Handler: mux},
		runner:          runner,
		service:         service,
		storeInterval:   cfg.StoreInterval,
		fileStoragePath: cfg.FileStoragePath,
	}

	mdw := []middleware.Middleware{
		hash.WithHash(cfg.Key),
		compress.WithGzip,
		logger.WithLogging,
	}

	updatesMdw := append(
		mdw,
		subnet.WithTructedSubnets(cfg.TrustedSubnet),
		crypto.WithDecrypt(buf),
	)

	mux.Get(`/`, middleware.Combine(s.listMetricHandler, mdw...))
	mux.Get(`/ping`, logger.WithLogging(s.pingDBHandler))
	mux.Post(`/update/`, middleware.Combine(s.updateHandlerJSON, mdw...))
	mux.Post(`/updates/`, middleware.Combine(s.updatesHandlerJSON, updatesMdw...))
	mux.Post(`/value/`, middleware.Combine(s.getMetricHandlerJSON, mdw...))
	mux.Get(`/value/{type}/{name}`, logger.WithLogging(s.getMetricHandler))
	mux.Post(`/update/{type}/{name}/{value}`, logger.WithLogging(s.updateHandler))

	mux.Mount("/debug", chimiddle.Profiler())

	return s, nil
}

// Run runs the server
func (s *Server) Run(ctx context.Context) {
	logger.Log.Info("Server started.")

	storeInterval := time.Second * time.Duration(s.storeInterval)

	if s.storeInterval > 0 {
		s.runner.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(storeInterval):
					if err := s.service.SaveToFile(ctx, s.fileStoragePath); err != nil {
						logger.Log.Error("save error", zap.Error(err))
					}
				}
			}
		})
	}

	s.runner.Go(func() error {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info("Server stoped.")
	return s.srv.Shutdown(ctx)
}
