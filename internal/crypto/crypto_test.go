package crypto

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecryptMiddleware(t *testing.T) {
	type want struct {
		code     int
		hasError bool
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "wrong sertificate",
			want: want{
				code:     http.StatusForbidden,
				hasError: true,
			},
		},
		{
			name: "correct sertificate",
			want: want{
				code:     http.StatusOK,
				hasError: false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			pub, key, err := GenerateCert()
			assert.NoError(t, err)

			if test.want.hasError {
				pub, _, err = GenerateCert()
				assert.NoError(t, err)
			}

			publicKeyBlock, _ := pem.Decode(pub)
			publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
			assert.NoError(t, err)

			body, err := EncryptOAEP(sha256.New(), publicKey.(*rsa.PublicKey), []byte("Test body"), nil)
			assert.NoError(t, err)

			req := httptest.NewRequest("GET", "http://testing", bytes.NewBuffer(body))
			rec := httptest.NewRecorder()

			handler := WithDecrypt(key)(nextHandler)
			handler.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}
