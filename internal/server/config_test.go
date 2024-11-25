package server

import (
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
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

func Test_ConfigFile(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	want := CfgFile{
		Address:       "localhost:9080",
		Restore:       true,
		StoreInterval: "11s",
		StoreFile:     "",
		DatabaseDSN:   "",
		CryptoKey:     "",
	}

	file, err := createConfigFile(&want)
	assert.NoError(t, err)

	t.Setenv("CONFIG", file)

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, want.Address, cfg.Address)
	si, _ := time.ParseDuration(want.StoreInterval)
	assert.Equal(t, int64(si.Seconds()), cfg.StoreInterval)
}

func createConfigFile(cfg *CfgFile) (string, error) {
	buf, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp("", "test")
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Write(buf)
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}
