package hash

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"hash"
	"io"
	"net/http"
)

const (
	HashHeaderKey = "HashSHA256"
)

func Hash(key, value []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(value)
	return h.Sum(nil)
}

type hashWriter struct {
	w http.ResponseWriter
	h hash.Hash
}

func newHashWriter(w http.ResponseWriter, key string) *hashWriter {
	return &hashWriter{
		w: w,
		h: hmac.New(sha256.New, []byte(key)),
	}
}

func (h *hashWriter) Header() http.Header {
	return h.w.Header()
}

func (h *hashWriter) Write(p []byte) (int, error) {
	h.h.Write(p)
	return h.w.Write(p)
}

func (h *hashWriter) WriteHeader(statusCode int) {
	h.w.WriteHeader(statusCode)
}

func (h *hashWriter) Close() error {
	sign := base64.StdEncoding.EncodeToString(h.h.Sum(nil))
	h.w.Header().Set(HashHeaderKey, sign)

	return nil
}

func WithHash(key string) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			originalWriter := w

			if key != "" {
				hashWriter := newHashWriter(w, key)
				originalWriter = hashWriter
				defer hashWriter.Close()

				sign := r.Header.Get(HashHeaderKey)
				if sign != "" {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						http.Error(w, "can't read body", http.StatusBadRequest)
						return
					}

					bodySign := base64.StdEncoding.EncodeToString(Hash([]byte(key), body))
					if sign != bodySign {
						http.Error(w, "wrong signature", http.StatusBadRequest)
						return
					}

					r.Body = io.NopCloser(bytes.NewBuffer(body))
				}
			}

			h.ServeHTTP(originalWriter, r)
		}
	}
}
