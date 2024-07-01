package server

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
)

type Repository interface {
	Set(value metric.Metric) metric.Metric
	Get(key string) (metric.Metric, bool)
}

type Server struct {
	srv     *http.Server
	Storage Repository
}

func NewServer(storage Repository) *Server {
	mux := http.NewServeMux()

	s := &Server{
		srv: &http.Server{Addr: `:8080`, Handler: mux },
		Storage: storage,
	}

	mux.HandleFunc(`/update/`, s.updateHandler)

	return s
}

func (s *Server) Run() error {
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) updateHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	// check params
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(res, "not found", http.StatusNotFound)
		return
	}

	// check metric type
	validate, ok := metric.AllowedMetricName[parts[2]]
	if !ok {
		http.Error(res, "bad request (type)", http.StatusBadRequest)
		return
	}

	// check metric value
	if !validate(parts[4]) {
		log.Println(parts[1:])
		http.Error(res, "bad request (value)", http.StatusBadRequest)
		return
	}

	s.Storage.Set(metric.Metric{Name: parts[3], Type: parts[2], Value: parts[4]})

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
}
