package metrics

import (
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	metricsv1 "github.com/nbvehbq/go-metrics-harvester/pkg/contract/gen/metrics"
	"google.golang.org/grpc"
)

type serverAPI struct {
	metricsv1.UnimplementedMetricServiceServer
	service metric.MetricService
}

func Register(server *grpc.Server, srv metric.MetricService) {
	metricsv1.RegisterMetricServiceServer(server, &serverAPI{service: srv})
}
