package compress

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const compressHeader = "gzip"

func TestCommpress_header(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Test"))
	})

	handler := WithGzip(nextHandler)

	req := httptest.NewRequest("GET", "http://testing", nil)
	req.Header.Set("Accept-Encoding", compressHeader)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Content-Encoding") != compressHeader {
		t.Error("wrong header value")
	}
}
