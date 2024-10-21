package hash

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHashMiddleware(t *testing.T) {
	type want struct {
		code int
	}
	tests := []struct {
		name string
		hash string
		want want
	}{
		{
			name: "wrong hash",
			hash: "wrong hash",
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "correct hash",
			hash: base64.StdEncoding.EncodeToString(Hash([]byte(HashHeaderKey), []byte("Test body"))),
			want: want{
				code: http.StatusOK,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "http://testing", bytes.NewBuffer([]byte("Test body")))
			req.Header.Set(HashHeaderKey, test.hash)
			rec := httptest.NewRecorder()

			handler := WithHash(HashHeaderKey)(nextHandler)
			handler.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			if resp.StatusCode != test.want.code {
				t.Errorf("wrong status code: want %d, got %d", test.want.code, resp.StatusCode)
			}

		})
	}
}
