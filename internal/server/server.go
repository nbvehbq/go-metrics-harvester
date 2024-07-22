package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nbvehbq/go-metrics-harvester/internal/compress"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
)

type Repository interface {
	Set(value metric.Metric)
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

	mux.Get("/", logger.WithLogging(compress.WithGzip(s.listMetricHandler)))
	mux.Post(`/update/`, logger.WithLogging(compress.WithGzip(s.updateHandlerJSON)))
	mux.Post(`/value/`, logger.WithLogging(compress.WithGzip(s.getMetricHandlerJSON)))
	mux.Get("/value/{type}/{name}", logger.WithLogging(s.getMetricHandler))
	mux.Post(`/update/{type}/{name}/{value}`, logger.WithLogging(s.updateHandler))

	return s
}

func (s *Server) Run() error {
	logger.Log.Info("Server started.")
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info("Server stoped.")
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
		switch v.MType {
		case metric.Gauge:
			value = strconv.FormatFloat(*v.Value, 'f', -1, 64)
		case metric.Counter:
			value = fmt.Sprintf("%d", *v.Delta)
		}
		li[i] = fmt.Sprintf("<li>%s: %s</li>", v.ID, value)
	}

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(res, html, strings.Join(li, "\n"))
}

func (s *Server) getMetricHandlerJSON(res http.ResponseWriter, req *http.Request) {
	var m metric.Metric

	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}

	_, ok := metric.AllowedMetricName[m.MType]
	if !ok {
		JSONError(res, "not found", http.StatusNotFound)
		return
	}

	value, ok := s.storage.Get(m.ID)
	if !ok {
		JSONError(res, "not found", http.StatusNotFound)
		return
	}

	if value.MType != m.MType {
		JSONError(res, "not found", http.StatusNotFound)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(res).Encode(value); err != nil {
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}
}

func (s *Server) updateHandlerJSON(res http.ResponseWriter, req *http.Request) {
	var m metric.Metric

	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}

	//check metric name
	if m.ID == "" {
		JSONError(res, "not found", http.StatusNotFound)
		return
	}

	// check metric type
	_, ok := metric.AllowedMetricName[m.MType]
	if !ok {
		JSONError(res, "bad request (type)", http.StatusBadRequest)
		return
	}

	s.storage.Set(m)
	updated, _ := s.storage.Get(m.ID)

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(res).Encode(updated); err != nil {
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}
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

	switch m.MType {
	case metric.Counter:
		res.Write([]byte(fmt.Sprintf("%d", *m.Delta)))
	case metric.Gauge:
		res.Write([]byte(strconv.FormatFloat(*m.Value, 'f', -1, 64)))
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
	m := metric.Metric{ID: mname, MType: mtype}
	if mtype == metric.Counter {
		delta, _ := strconv.ParseInt(mvalue, 10, 64)
		m.Delta = &delta
	} else {
		value, _ := strconv.ParseFloat(mvalue, 64)
		m.Value = &value
	}

	s.storage.Set(m)

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
}

func JSONError(w http.ResponseWriter, msg string, code int) {
	res := struct {
		Err string `json:"error"`
	}{Err: msg}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(res)
}
