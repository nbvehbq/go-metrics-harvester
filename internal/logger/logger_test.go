package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	type args struct {
		level string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "initialize logger with no error",
			args:    args{level: "info"},
			wantErr: false,
		},
		{
			name:    "initialize logger with error",
			args:    args{level: "bad"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.args.level)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

type testLoggerWriter struct {
	*httptest.ResponseRecorder
}

func TestWithLogging(t *testing.T) {

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := w.(*loggingResponseWriter)
		if !ok {
			t.Errorf("loggingResponseWriter is unavailable on the writer.")
		}
	})

	r := httptest.NewRequest("GET", "/", nil)
	w := testLoggerWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}

	handler := WithLogging(testHandler)
	handler.ServeHTTP(w, r)
}
