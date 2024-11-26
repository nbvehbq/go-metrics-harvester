package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/nbvehbq/go-metrics-harvester/internal/crypto"
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

	return crypto.DecryptOAEP(sha256.New(), privateKey, buf, nil)
}

func TestAgent_publishMetrics(t *testing.T) {
	type fields struct {
		cfg       *Config
		runner    *errgroup.Group
		client    HTTPClient
		publicKey []byte
	}
	type args struct {
		m *metric.Metrics
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{
				cfg:       tt.fields.cfg,
				runner:    tt.fields.runner,
				client:    tt.fields.client,
				publicKey: tt.fields.publicKey,
			}
			if err := a.publishMetrics(tt.args.m); (err != nil) != tt.wantErr {
				t.Errorf("Agent.publishMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
