package server

import (
	"reflect"
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		want    *Config
		wantErr bool
	}{
		{
			name: "default config",
			want: &Config{
				Address:         "localhost:8080",
				LogLevel:        "info",
				StoreInterval:   300,
				FileStoragePath: "/tmp",
				Restore:         true,
				DatabaseDSN:     "",
				Key:             "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("FILE_STORAGE_PATH", "/tmp")
			got, err := NewConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
