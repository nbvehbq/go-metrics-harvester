package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"go.uber.org/zap"
)

func (s *Server) pingDBHandler(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	if err := s.service.Ping(ctx); err != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
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

	ctx := req.Context()
	list, err := s.service.List(ctx)
	if err != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

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
	ctx := req.Context()
	var m metric.Metric

	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}

	_, ok := metric.AllowedMetricType[m.MType]
	if !ok {
		JSONError(res, "not found", http.StatusNotFound)
		return
	}

	value, err := s.service.Get(ctx, m.ID, m.MType)
	if err != nil {
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
	ctx := req.Context()
	var m metric.Metric

	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.service.Set(ctx, m); err != nil {
		switch {
		case errors.Is(err, metric.ErrMetricNotFound):
			JSONError(res, err.Error(), http.StatusNotFound)
		case errors.Is(err, metric.ErrMetricBadType):
			JSONError(res, err.Error(), http.StatusBadRequest)
		default:
			JSONError(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	updated, _ := s.service.Get(ctx, m.ID, m.MType)

	if s.storeInterval == 0 {
		go s.service.SaveToFile(ctx, s.fileStoragePath)
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(res).Encode(updated); err != nil {
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}
}

func (s *Server) getMetricHandler(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	mtype := chi.URLParam(req, "type")
	mname := chi.URLParam(req, "name")

	_, ok := metric.AllowedMetricType[mtype]
	if !ok {
		http.Error(res, "not found", http.StatusNotFound)
		return
	}

	m, err := s.service.Get(ctx, mname, mtype)
	if err != nil {
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
	ctx := req.Context()
	mtype := chi.URLParam(req, "type")
	mname := chi.URLParam(req, "name")
	mvalue := chi.URLParam(req, "value")

	//check metric name
	if mname == "" {
		http.Error(res, "not found", http.StatusNotFound)
		return
	}

	// check metric type
	validate, ok := metric.AllowedMetricType[mtype]
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
		// ошибку не обрабатываем т.к выше вызывали функцию validate
		delta, _ := strconv.ParseInt(mvalue, 10, 64)
		m.Delta = &delta
	} else {
		// ошибку не обрабатываем т.к выше вызывали функцию validate
		value, _ := strconv.ParseFloat(mvalue, 64)
		m.Value = &value
	}

	if err := s.service.Set(ctx, m); err != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
}

func (s *Server) updatesHandlerJSON(res http.ResponseWriter, req *http.Request) {

	var me []metric.Metric
	if err := json.NewDecoder(req.Body).Decode(&me); err != nil {
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}

	for _, m := range me {
		//check metric name
		if m.ID == "" {
			JSONError(res, "not found", http.StatusNotFound)
			return
		}

		// check metric type
		_, ok := metric.AllowedMetricType[m.MType]
		if !ok {
			JSONError(res, "bad request (type)", http.StatusBadRequest)
			return
		}
	}

	ctx := req.Context()
	if err := s.service.Update(ctx, me); err != nil {
		logger.Log.Error("update", zap.Error(err))
		JSONError(res, err.Error(), http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
}

// JSONError sends an error message in JSON format
func JSONError(w http.ResponseWriter, msg string, code int) {
	res := struct {
		Err string `json:"error"`
	}{Err: msg}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(res)
}
