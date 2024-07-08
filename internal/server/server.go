package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
)

type Repository interface {
	Set(value metric.Metric) metric.Metric
	Get(key string) (metric.Metric, bool)
	List() []metric.Metric
}

type Server struct {
	srv     *http.Server
	storage Repository
}

func NewServer(storage Repository, cfg *Config) *Server {
	mux := chi.NewRouter()

	s := &Server{
		srv:     &http.Server{Addr: cfg.Address, Handler: mux},
		storage: storage,
	}

	mux.Get("/", s.listMetricHandler)
	mux.Get("/value/{type}/{name}", s.getMetricHandler)
	mux.Post(`/update/{type}/{name}/{value}`, s.updateHandler)

	return s
}

func (s *Server) Run() error {
	log.Printf("Server started.")
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Server stoped.")
	return s.srv.Shutdown(ctx)
}

func (s *Server) listMetricHandler(res http.ResponseWriter, req *http.Request) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Metrics list</title>
</head>
<body>
  <ol>
    %s
  </ol>
</body>
</html>
	`

	list := s.storage.List()
	li := make([]string, len(list))
	for i, v := range list {
		var value string
		switch v.Type {
		case metric.Counter:
			value = strconv.FormatFloat(v.Value.(float64), 'f', -1, 64)
		case metric.Gauge:
			value = fmt.Sprintf("%f", v.Value)
		}
		li[i] = fmt.Sprintf("<li>%s: %s</li>", v.Name, value)
	}

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(res, html, strings.Join(li, "\n"))
}

func (s *Server) getMetricHandler(res http.ResponseWriter, req *http.Request) {
	mtype := chi.URLParam(req, "type")
	mname := chi.URLParam(req, "name")

	_, ok := metric.AllowedMetricName[mtype]
	if !ok {
		http.Error(res, "not found", http.StatusNotFound)
		return
	}

	m, ok := s.storage.Get(mname)
	if !ok {
		http.Error(res, "not found", http.StatusNotFound)
		return
	}

	switch v := m.Value.(type) {
	case int64:
		res.Write([]byte(fmt.Sprintf("%d", v)))
	case float64:
		res.Write([]byte(strconv.FormatFloat(v, 'f', -1, 64)))
	default:
		res.Write([]byte(fmt.Sprintf("%v", v)))
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
}

func (s *Server) updateHandler(res http.ResponseWriter, req *http.Request) {
	mtype := chi.URLParam(req, "type")
	mname := chi.URLParam(req, "name")
	mvalue := chi.URLParam(req, "value")

	//check metric name
	if mname == "" {
		http.Error(res, "not found", http.StatusNotFound)
		return
	}

	// check metric type
	validate, ok := metric.AllowedMetricName[mtype]
	if !ok {
		http.Error(res, "bad request (type)", http.StatusBadRequest)
		return
	}

	// check metric value
	if !validate(mvalue) {
		http.Error(res, "bad request (value)", http.StatusBadRequest)
		return
	}

	s.storage.Set(metric.Metric{Name: mname, Type: mtype, Value: mvalue})

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
}
