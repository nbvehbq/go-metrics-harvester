package httpclient

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/nbvehbq/go-metrics-harvester/internal/agent"
	"github.com/nbvehbq/go-metrics-harvester/internal/crypto"
	"github.com/nbvehbq/go-metrics-harvester/internal/hash"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/pkg/retry"
	"github.com/pkg/errors"
)

type HTTPClient struct {
	client    http.Client
	publicKey []byte
	address   string
	key       string
}

func NewHTTPClient(cfg *agent.Config) (*HTTPClient, error) {
	var buf []byte
	var err error
	if cfg.CryptoKey != "" {
		buf, err = os.ReadFile(cfg.CryptoKey)
		if err != nil {
			return nil, errors.Wrap(err, "open public key filename")
		}
	}

	return &HTTPClient{
		client:    http.Client{},
		address:   cfg.Address,
		key:       cfg.Key,
		publicKey: buf,
	}, nil
}

func (h *HTTPClient) Publish(_ context.Context, m []metric.Metric) error {
	buf, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	if h.publicKey != nil {
		publicKeyBlock, _ := pem.Decode(h.publicKey)
		publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
		if err != nil {
			return errors.Wrap(err, "parse public key")
		}

		buf, err = crypto.EncryptOAEP(sha256.New(), publicKey.(*rsa.PublicKey), buf, nil)
		if err != nil {
			return errors.Wrap(err, "encrypt body")
		}
	}

	buf, err = compress(buf)
	if err != nil {
		return errors.Wrap(err, "compress")
	}

	err = retry.Do(func() (err error) {
		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/updates/", h.address),
			bytes.NewReader(buf))
		if err != nil {
			return errors.Wrap(err, "new request")
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept-Encoding", "gzip")
		req.Header.Add("Content-Encoding", "gzip")

		addr, err := realIP()
		if err != nil {
			return errors.Wrap(err, "get ip address")
		}
		req.Header.Add("X-Real-IP", addr)

		if h.key != "" {
			sign := hash.Hash([]byte(h.key), buf)
			req.Header.Add(hash.HashHeaderKey, base64.StdEncoding.EncodeToString(sign))
		}

		res, err := h.client.Do(req)
		if err != nil {
			return errors.Wrap(err, "send request")
		}
		defer res.Body.Close()

		return
	})

	if err != nil {
		return errors.Wrap(err, "error while publish")
	}

	return nil
}

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	_, err := w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to buffer: %v", err)
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}

	return b.Bytes(), nil
}

func realIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}

	return "127.0.0.1", nil
}
