package grpclient

import (
	"context"

	"github.com/nbvehbq/go-metrics-harvester/internal/agent"
	"github.com/nbvehbq/go-metrics-harvester/internal/hash"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	metricsv1 "github.com/nbvehbq/go-metrics-harvester/pkg/contract/gen/metrics"
	"github.com/nbvehbq/go-metrics-harvester/pkg/retry"
)

type GRPClient struct {
	client  metricsv1.MetricServiceClient
	address string
	key     string
}

func NewGRPClient(cfg *agent.Config) (*GRPClient, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GRPClient{
		client:  metricsv1.NewMetricServiceClient(conn),
		address: cfg.Address,
		key:     cfg.Key,
	}, nil
}

func (h *GRPClient) Publish(ctx context.Context, m []metric.Metric) error {
	list := make([]*metricsv1.Metric, 0, len(m))
	for _, v := range m {
		list = append(list, &metricsv1.Metric{
			Id:    v.ID,
			MType: v.MType,
			Delta: v.Delta,
			Value: v.Value,
		})
	}
	message := metricsv1.UpdateRequest{Metric: list}

	var sign []byte
	if h.key != "" {
		buf, err := proto.Marshal(&message)
		if err != nil {
			return errors.Wrap(err, "marshal message")
		}
		sign = hash.Hash([]byte(h.key), buf)
	}

	err := retry.Do(func() (err error) {
		if sign != nil {
			ctx = metadata.AppendToOutgoingContext(ctx, hash.HashHeaderKey, string(sign))
		}
		_, err = h.client.Update(ctx, &message, grpc.UseCompressor(gzip.Name))
		if err != nil {
			return errors.Wrap(err, "send request")
		}

		return
	})

	if err != nil {
		return errors.Wrap(err, "error while publish")
	}

	return nil
}
