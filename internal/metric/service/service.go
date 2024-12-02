package service

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
)

type Repository interface {
	Set(context.Context, metric.Metric) error
	Get(context.Context, string) (metric.Metric, bool)
	List(context.Context) ([]metric.Metric, error)
	Persist(context.Context, io.Writer) error
	Ping(context.Context) error
	Update(context.Context, []metric.Metric) error
}

type Service struct {
	storage Repository
}

func NewService(storage Repository) *Service {
	return &Service{storage: storage}
}

func (s *Service) List(ctx context.Context) ([]metric.Metric, error) {
	return s.storage.List(ctx)
}

func (s *Service) Get(ctx context.Context, ID, MType string) (*metric.Metric, error) {
	_, ok := metric.AllowedMetricType[MType]
	if !ok {
		return nil, errors.New("type not alowed")
	}

	value, ok := s.storage.Get(ctx, ID)
	if !ok {
		return nil, errors.New("not found")
	}

	if value.MType != MType {
		return nil, errors.New("not found")
	}

	return &value, nil
}

func (s *Service) Update(ctx context.Context, me []metric.Metric) error {
	for _, m := range me {
		//check metric name
		if m.ID == "" {
			return metric.ErrMetricNotFound
		}

		// check metric type
		_, ok := metric.AllowedMetricType[m.MType]
		if !ok {
			return metric.ErrMetricBadType
		}
	}

	return s.storage.Update(ctx, me)
}

func (s *Service) Ping(ctx context.Context) error {
	return s.storage.Ping(ctx)
}

func (s *Service) Set(ctx context.Context, m metric.Metric) error {
	//check metric name
	if m.ID == "" {
		return metric.ErrMetricNotFound
	}

	// check metric type
	_, ok := metric.AllowedMetricType[m.MType]
	if !ok {
		return metric.ErrMetricBadType
	}

	return s.storage.Set(ctx, m)
}

func (s *Service) SaveToFile(ctx context.Context, path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	if err := s.storage.Persist(ctx, file); err != nil {
		return err
	}

	return nil
}
