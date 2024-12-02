package metrics

import (
	"context"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	metricsv1 "github.com/nbvehbq/go-metrics-harvester/pkg/contract/gen/metrics"
)

func (s *serverAPI) List(ctx context.Context, _ *metricsv1.ListRequest) (*metricsv1.ListResponse, error) {
	list, err := s.service.List(ctx)
	if err != nil {
		return nil, internalError(err)
	}

	res := make([]*metricsv1.Metric, 0, len(list))
	for _, v := range list {
		res = append(res, &metricsv1.Metric{
			Id:    v.ID,
			MType: v.MType,
			Delta: v.Delta,
			Value: v.Value,
		})
	}

	return &metricsv1.ListResponse{Metric: res}, nil
}

func (s *serverAPI) Update(ctx context.Context, in *metricsv1.UpdateRequest) (*metricsv1.UpdateResponse, error) {
	m := make([]metric.Metric, 0, len(in.Metric))
	for _, v := range in.Metric {
		m = append(m, metric.Metric{
			ID:    v.Id,
			MType: v.MType,
			Delta: v.Delta,
			Value: v.Value,
		})
	}
	if err := s.service.Update(ctx, m); err != nil {
		return nil, internalError(err)
	}

	return &metricsv1.UpdateResponse{}, nil
}

func (s *serverAPI) Value(ctx context.Context, in *metricsv1.ValueRequest) (*metricsv1.ValueResponse, error) {
	value, err := s.service.Get(ctx, in.Id, in.Type)
	if err != nil {
		return nil, argumentError(err)
	}

	return &metricsv1.ValueResponse{
		Metric: &metricsv1.Metric{
			Id:    value.ID,
			MType: value.MType,
			Delta: value.Delta,
			Value: value.Value,
		},
	}, nil
}
