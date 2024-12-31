package grpc

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	ilog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/nbvehbq/go-metrics-harvester/internal/grpc/metrics"
	"github.com/nbvehbq/go-metrics-harvester/internal/hash"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	srv "github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/server"
	"github.com/nbvehbq/go-metrics-harvester/internal/subnet"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

type Grpc struct {
	server  *grpc.Server
	service srv.MetricService
	runner  *errgroup.Group
	cfg     *server.Config
}

func NewGrpc(ctx context.Context, runner *errgroup.Group, metric srv.MetricService, cfg *server.Config) (*Grpc, error) {
	opts := []ilog.Option{
		ilog.WithLogOnEvents(ilog.StartCall, ilog.FinishCall),
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			ilog.UnaryServerInterceptor(InterceptorLogger(logger.Log), opts...),
			hash.UnaryServerInterceptor(cfg.Key),
			subnet.UnaryServerInterceptor(cfg.TrustedSubnet),
		),
	)
	metrics.Register(server, metric)

	return &Grpc{server: server, runner: runner, cfg: cfg, service: metric}, nil
}

func (g *Grpc) Run(ctx context.Context) error {
	host, port, err := net.SplitHostPort(g.cfg.Address)
	if err != nil {
		return errors.Wrap(err, "parse address")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return errors.Wrap(err, "parse port")
	}

	g.runner.Go(func() error {
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, portNum+1))
		if err != nil {
			return errors.Wrap(err, "listen")
		}

		if err := g.server.Serve(l); err != nil {
			return err
		}

		return nil
	})

	storeInterval := time.Second * time.Duration(g.cfg.StoreInterval)

	if g.cfg.StoreInterval > 0 {
		g.runner.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(storeInterval):
					if err := g.service.SaveToFile(ctx, g.cfg.FileStoragePath); err != nil {
						logger.Log.Error("save error", zap.Error(err))
					}
				}
			}
		})
	}

	return nil
}

func (g *Grpc) Shutdown(_ context.Context) {
	g.server.GracefulStop()
}

func InterceptorLogger(l *zap.Logger) ilog.Logger {
	return ilog.LoggerFunc(func(ctx context.Context, lvl ilog.Level, msg string, fields ...any) {
		f := make([]zap.Field, 0, len(fields)/2)

		for i := 0; i < len(fields); i += 2 {
			key := fields[i]
			value := fields[i+1]

			switch v := value.(type) {
			case string:
				f = append(f, zap.String(key.(string), v))
			case int:
				f = append(f, zap.Int(key.(string), v))
			case bool:
				f = append(f, zap.Bool(key.(string), v))
			default:
				f = append(f, zap.Any(key.(string), v))
			}
		}

		logger := l.WithOptions(zap.AddCallerSkip(1)).With(f...)

		switch lvl {
		case ilog.LevelDebug:
			logger.Debug(msg)
		case ilog.LevelInfo:
			logger.Info(msg)
		case ilog.LevelWarn:
			logger.Warn(msg)
		case ilog.LevelError:
			logger.Error(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}
