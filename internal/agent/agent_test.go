package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	xhash "hash"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nbvehbq/go-metrics-harvester/internal/agent/mocks"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func ptr[T any](v T) *T { return &v }

func Test_requestMetrics(t *testing.T) {
	type args struct {
		m *metric.Metrics
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "request metrics",
			args: args{m: metric.NewMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestMetrics(tt.args.m)
			assert.Contains(t, tt.args.m.Metrics, "Alloc")
			assert.Contains(t, tt.args.m.Metrics, "BuckHashSys")
			assert.Contains(t, tt.args.m.Metrics, "Frees")
			assert.Contains(t, tt.args.m.Metrics, "GCCPUFraction")
			assert.Contains(t, tt.args.m.Metrics, "GCSys")
			assert.Contains(t, tt.args.m.Metrics, "HeapAlloc")
			assert.Contains(t, tt.args.m.Metrics, "HeapIdle")
			assert.Contains(t, tt.args.m.Metrics, "HeapInuse")
			assert.Contains(t, tt.args.m.Metrics, "HeapObjects")
			assert.Contains(t, tt.args.m.Metrics, "HeapReleased")
			assert.Contains(t, tt.args.m.Metrics, "HeapSys")
			assert.Contains(t, tt.args.m.Metrics, "LastGC")
			assert.Contains(t, tt.args.m.Metrics, "Lookups")
			assert.Contains(t, tt.args.m.Metrics, "MCacheInuse")
			assert.Contains(t, tt.args.m.Metrics, "MCacheSys")
			assert.Contains(t, tt.args.m.Metrics, "MSpanInuse")
			assert.Contains(t, tt.args.m.Metrics, "MSpanSys")
			assert.Contains(t, tt.args.m.Metrics, "Mallocs")
			assert.Contains(t, tt.args.m.Metrics, "NextGC")
			assert.Contains(t, tt.args.m.Metrics, "NumForcedGC")
			assert.Contains(t, tt.args.m.Metrics, "NumGC")
			assert.Contains(t, tt.args.m.Metrics, "OtherSys")
			assert.Contains(t, tt.args.m.Metrics, "PauseTotalNs")
			assert.Contains(t, tt.args.m.Metrics, "StackInuse")
			assert.Contains(t, tt.args.m.Metrics, "StackSys")
			assert.Contains(t, tt.args.m.Metrics, "Sys")
			assert.Contains(t, tt.args.m.Metrics, "TotalAlloc")
		})
	}
}

func Test_requestMemoryMetrics(t *testing.T) {
	type args struct {
		m *metric.Metrics
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "request memory metrics",
			args: args{m: metric.NewMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requestMemoryMetrics(context.Background(), tt.args.m)
			assert.NoError(t, err)
			assert.Contains(t, tt.args.m.Metrics, "TotalMemory")
			assert.Contains(t, tt.args.m.Metrics, "FreeMemory")
		})
	}
}

func Test_commpress(t *testing.T) {
	payload := []byte("hello world")
	b, err := compress(payload)
	assert.NoError(t, err)
	assert.NotEqual(t, payload, b)
}

func Test_publishMetrics(t *testing.T) {
	type want struct {
		wantError bool
	}
	tests := []struct {
		name string
		cfg  *Config
		want want
	}{
		{
			name: " success publish metrics",
			cfg: &Config{
				Address:  "localhost:8080",
				LogLevel: "debug",
			},
			want: want{wantError: false},
		},
		{
			name: " fail publish metrics",
			cfg: &Config{
				Address:  "localhost:8080",
				LogLevel: "debug",
			},
			want: want{wantError: true},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockHttpClient(ctrl)

	// prepare metrics
	mt := metric.NewMetrics()
	requestMetrics(mt)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := http.StatusOK
			if tt.want.wantError {
				status = http.StatusBadRequest
			}
			m.EXPECT().Do(gomock.Any()).Return(&http.Response{
				Body:       io.NopCloser(bytes.NewBufferString("test")),
				StatusCode: status,
			}, nil)
			a := &Agent{
				cfg:       tt.cfg,
				runner:    &errgroup.Group{},
				client:    m,
				publicKey: nil,
			}
			err := a.publishMetrics(mt)
			if tt.want.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_publishMetrics_crypt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockHttpClient(ctrl)

	// prepare metrics
	mt := metric.NewMetrics()
	requestMetrics(mt)

	cert, key, err := generateCert()
	assert.NoError(t, err)

	var req *http.Request
	m.EXPECT().
		Do(gomock.Any()).
		DoAndReturn(
			func(arg *http.Request) (*http.Response, error) {
				req = arg
				return &http.Response{
					Body:       io.NopCloser(bytes.NewBufferString("test")),
					StatusCode: http.StatusOK,
				}, nil
			},
		)

	a := &Agent{
		cfg: &Config{
			Address:  "localhost:8080",
			LogLevel: "debug",
		},
		runner:    &errgroup.Group{},
		client:    m,
		publicKey: cert,
	}
	err = a.publishMetrics(mt)
	assert.NoError(t, err)

	// decompress body
	z, err := gzip.NewReader(req.Body)
	assert.NoError(t, err)
	var resB bytes.Buffer
	_, err = resB.ReadFrom(z)
	assert.NoError(t, err)
	body := resB.Bytes()

	// decrypt
	plain, err := decrypt(body, key)
	assert.NoError(t, err)

	var list []metric.Metric
	err = json.Unmarshal(plain, &list)
	assert.NoError(t, err)
}

func generateCert() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
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

	return certPEM.Bytes(), privateKeyPEM.Bytes(), nil
}

func decrypt(buf, key []byte) ([]byte, error) {
	privateKeyBlock, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return decryptOAEP(sha256.New(), nil, privateKey, buf, nil)
}

func decryptOAEP(hash xhash.Hash, random io.Reader, private *rsa.PrivateKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)
	step := private.PublicKey.Size()
	var decryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		decryptedBlockBytes, err := rsa.DecryptOAEP(hash, random, private, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}

	return decryptedBytes, nil
}
