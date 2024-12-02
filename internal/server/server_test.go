package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric/mocks"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func intPtr(v int64) *int64 {
	return &v
}

func TestServer_PingHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockMetricService(ctrl)

	w := httptest.NewRecorder()

	m.EXPECT().Ping(gomock.Any()).Return(nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	runner, _ := errgroup.WithContext(req.Context())

	srv, err := NewServer(runner, m, &Config{})
	assert.NoError(t, err)
	srv.pingDBHandler(w, req)

	res := w.Result()
	res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestServer_updatesServerJSON(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name string
		body []byte
		want want
	}{
		{
			name: "bad body",
			body: []byte("bad body"),
			want: want{
				code:        400,
				contentType: "application/json",
			},
		},
		{
			name: "bad metric name",
			body: []byte(`[{"id":"","type":"counter","delta":42}]`),
			want: want{
				code:        404,
				contentType: "application/json",
			},
		},
		{
			name: "wrong type",
			body: []byte(`[{"id":"test","type":"histogram","delta":42}]`),
			want: want{
				code:        400,
				contentType: "application/json",
			},
		}, {
			name: "unmarshal error",
			body: []byte(`[{"id":"test","type":"gauge","delta":42.0}]`),
			want: want{
				code:        400,
				contentType: "application/json",
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockMetricService(ctrl)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			m.EXPECT().
				Set(gomock.Any(), gomock.Any()).
				Return(nil).
				AnyTimes()

			req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewBuffer(test.body))
			runner, _ := errgroup.WithContext(req.Context())
			srv, err := NewServer(runner, m, &Config{})
			assert.NoError(t, err)
			srv.updatesHandlerJSON(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestServer_updateHandlerJSON(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name string
		body []byte
		want want
	}{
		{
			name: "counter test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{"type":"counter","id":"test","delta":1}`),
		},
		{
			name: "gauge test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "gauge",
				"id":  "test",
				"value": 2.34
			}`),
		},
		{
			name: "empty type",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "",
				"id":  "test",
				"value": 2.34
			}`),
		},
		{
			name: "empty metric name",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "gauge",
				"id":  "",
				"value": 1
			}`),
		},
		{
			name: "invalid counter",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{"type": "counter", "id": "test", "delta": 3}`),
		},
		{
			name: "invalid gauge",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "gauge",
				"id":  "test",
				"value": 10
			}`),
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockMetricService(ctrl)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			switch test.want.code {
			case 200:
				m.EXPECT().
					Set(gomock.Any(), gomock.Any()).
					Return(nil)
			case 404:
				m.EXPECT().
					Set(gomock.Any(), gomock.Any()).
					Return(metric.ErrMetricNotFound)
			case 400:
				m.EXPECT().
					Set(gomock.Any(), gomock.Any()).
					Return(metric.ErrMetricBadType)
			default:
				m.EXPECT().
					Set(gomock.Any(), gomock.Any()).
					Return(errors.New("new error"))
			}

			m.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&metric.Metric{}, nil).AnyTimes()
			m.EXPECT().SaveToFile(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewBuffer(test.body))
			runner, _ := errgroup.WithContext(req.Context())
			srv, err := NewServer(runner, m, &Config{})
			assert.NoError(t, err)
			srv.updateHandlerJSON(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestServer_getMetricHandlerJSON(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
		metric      metric.Metric
		res         error
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "wrong type",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				metric:      metric.Metric{ID: "test", MType: metric.Gauge, Delta: intPtr(42)},
				res:         errors.New("not found"),
			},
		},
		{
			name: "epsent metric",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				metric:      metric.Metric{ID: "test", MType: metric.Counter, Delta: intPtr(42)},
				res:         errors.New("not found"),
			},
		},
		{
			name: "return metric",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				metric:      metric.Metric{ID: "test3", MType: metric.Counter, Delta: intPtr(42)},
				res:         nil,
			},
		},
	}

	ctrl := gomock.NewController(t)
	m := mocks.NewMockMetricService(ctrl)
	defer ctrl.Finish()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			err := json.NewEncoder(&body).Encode(test.want.metric)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/value", &body)
			w := httptest.NewRecorder()

			m.EXPECT().
				Get(gomock.Any(), test.want.metric.ID, test.want.metric.MType).
				Return(&test.want.metric, test.want.res)

			runner, _ := errgroup.WithContext(req.Context())
			srv, err := NewServer(runner, m, &Config{})
			assert.NoError(t, err)
			srv.getMetricHandlerJSON(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestServer_listMetricHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name         string
		storageError bool
		want         want
	}{
		{
			name:         "list metrics",
			storageError: false,
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/html; charset=utf-8",
			},
		},
		{
			name:         "storage error",
			storageError: true,
			want: want{
				code:        500,
				response:    `{"status":"error"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockMetricService(ctrl)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.storageError {
				m.EXPECT().List(gomock.Any()).Return([]metric.Metric{}, errors.New("storage error"))
			} else {
				m.EXPECT().List(gomock.Any()).Return([]metric.Metric{}, nil)
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			runner, _ := errgroup.WithContext(req.Context())
			srv, err := NewServer(runner, m, &Config{})
			assert.NoError(t, err)

			srv.listMetricHandler(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestServer_getMetricHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
		metric      metric.Metric
		res         error
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "wrong type",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
				metric:      metric.Metric{ID: "test", MType: metric.Gauge, Delta: intPtr(42)},
				res:         errors.New("not found"),
			},
		},
		{
			name: "epsent metric",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
				metric:      metric.Metric{ID: "test", MType: metric.Counter, Delta: intPtr(42)},
				res:         errors.New("not found"),
			},
		},
		{
			name: "return metric",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
				metric:      metric.Metric{ID: "test3", MType: metric.Counter, Delta: intPtr(42)},
				res:         nil,
			},
		},
	}

	ctrl := gomock.NewController(t)
	m := mocks.NewMockMetricService(ctrl)
	defer ctrl.Finish()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/value/%s/%s/", test.want.metric.MType, test.want.metric.ID),
				nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("type", test.want.metric.MType)
			rctx.URLParams.Add("name", test.want.metric.ID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			m.EXPECT().
				Get(gomock.Any(), test.want.metric.ID, test.want.metric.MType).
				Return(&test.want.metric, test.want.res)

			runner, _ := errgroup.WithContext(req.Context())
			srv, err := NewServer(runner, m, &Config{})
			assert.NoError(t, err)
			srv.getMetricHandler(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestServer_updateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	type arg struct {
		mtype string
		name  string
		value string
	}
	tests := []struct {
		name string
		arg  arg
		want want
	}{
		{
			name: "wrong type",
			arg:  arg{name: "test", mtype: "histogram", value: "42"},
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "empty name",
			arg:  arg{name: "", mtype: "counter", value: "42"},
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "wrong value",
			arg:  arg{name: "test", mtype: "counter", value: "42.42"},
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "success update counter",
			arg:  arg{name: "test", mtype: "counter", value: "42"},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "success update gauge",
			arg:  arg{name: "test", mtype: "gauge", value: "0.5"},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
		},
	}

	ctrl := gomock.NewController(t)
	m := mocks.NewMockMetricService(ctrl)
	defer ctrl.Finish()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/value/%s/%s/%s", test.arg.mtype, test.arg.name, test.arg.value),
				nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("type", test.arg.mtype)
			rctx.URLParams.Add("name", test.arg.name)
			rctx.URLParams.Add("value", test.arg.value)
			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			m.EXPECT().
				Set(ctx, gomock.Any()).
				Return(nil).
				AnyTimes()

			runner, _ := errgroup.WithContext(req.Context())
			srv, err := NewServer(runner, m, &Config{})
			assert.NoError(t, err)
			srv.updateHandler(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func generateCert() (string, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", nil, err
	}

	certBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", nil, err
	}

	var certPEM bytes.Buffer
	pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	var privateKeyPEM bytes.Buffer
	pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	file, err := os.CreateTemp("", "test")
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	file.Write(privateKeyPEM.Bytes())

	return file.Name(), certPEM.Bytes(), nil
}
