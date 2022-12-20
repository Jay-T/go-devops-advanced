package server

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		unparsed string
		parsed   *ConfigFile
	}{
		{
			name: "TestOne",
			unparsed: `
			{
				"address": "localhost:8080",
				"restore": false,
				"store_interval": "10s",
				"store_file": "",
				"database_dsn": "postgres://localhost/mydb?sslmode=disable",
				"crypto_key": "/Users/daniliuk-ve/work/go-devops-advanced/internal/server/cert/key.priv"
			} 
			`,
			parsed: &ConfigFile{
				Address:       "localhost:8080",
				Restore:       false,
				StoreInterval: time.Duration(10 * time.Second),
				StoreFile:     "",
				DBAddress:     "postgres://localhost/mydb?sslmode=disable",
				CryptoKey:     "/Users/daniliuk-ve/work/go-devops-advanced/internal/server/cert/key.priv",
			},
		},
		// {
		// 	name: "TestTwo",
		// 	unparsed: `
		// 	{
		// 		"store_interval": 10
		// 	}
		// 	`,
		// 	parsed: &Config{
		// 		StoreInterval: time.Duration(10 * time.Second),
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConfigFile{}
			err := c.UnmarshalJSON([]byte(tt.unparsed))
			if err != nil {
				log.Fatal(err)
			}
			assert.Equal(t, tt.parsed, c)
		})
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		parsed    *Config
		wantError bool
		errorText string
	}{
		{
			name:     "TestOne",
			filename: "test_config.json",
			parsed: &Config{
				Address:       "localhost:8080",
				Restore:       false,
				StoreInterval: time.Duration(10 * time.Second),
				StoreFile:     "/tmp/testfile",
				DBAddress:     "postgres://localhost/mydb?sslmode=disable",
				CryptoKey:     "/Users/daniliuk-ve/work/go-devops-advanced/internal/server/cert/key.priv",
				ConfigFile:    "test_config.json",
			},
			wantError: false,
			errorText: "",
		},
		{
			name:      "TestTwo",
			filename:  "test_bad_config.json",
			parsed:    &Config{},
			wantError: true,
			errorText: `time: invalid duration "sss"`,
		},
		{
			name:      "TestThree",
			filename:  "test_bad_config_doesnt_exist.json",
			parsed:    &Config{},
			wantError: true,
			errorText: `open test_bad_config_doesnt_exist.json: no such file or directory`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Address:       defaultAddress,
				StoreInterval: defaultStoreInterval,
				Restore:       defaultRestore,
				StoreFile:     defaultStoreFile,
				DBAddress:     defaultDBAddress,
				CryptoKey:     defaultCryptoKey,
				ConfigFile:    tt.filename,
			}
			err := loadConfigFromFile(c)
			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorText, err.Error())
			} else {
				assert.Equal(t, tt.parsed, c)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	c := &Config{
		Address:       "localhost:9999",
		StoreInterval: time.Duration(300 * time.Second),
		StoreFile:     "/tmp/devops-metrics-db.json",
		Restore:       false,
		DBAddress:     "",
		CryptoKey:     "",
	}

	err := os.Setenv("ADDRESS", "localhost:9999")
	if err != nil {
		log.Fatal(err)
	}
	err = os.Setenv("RESTORE", "false")
	if err != nil {
		log.Fatal(err)
	}

	cfg, _ := GetConfig()
	assert.Equal(t, c, cfg)

}
