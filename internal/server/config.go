package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env"
)

const usage = `Usage of using_flag:

  -c, -config string Path to config file
  -a string Socket to listen on (default "localhost:8080")
  -crypto-key string Path to private key
  -d string Database address
  -f string File for saving data (default "/tmp/devops-metrics-db.json")
  -i duration Save data interval (default 5m0s)
  -k string Encryption key
  -r Restore data from file (default true)
`

// Config structure. Used for application configuration.
type Config struct {
	Address       string        `env:"ADDRESS" json:"address"`
	StoreInterval time.Duration `env:"STORE_INTERVAL" json:"store_interval"`
	StoreFile     string        `env:"STORE_FILE" json:"store_file"`
	Restore       bool          `env:"RESTORE" json:"restore"`
	Key           string        `env:"KEY"`
	DBAddress     string        `env:"DATABASE_DSN" json:"database_dsn"`
	CryptoKey     string        `env:"CRYPTO_KEY" json:"crypto_key"`
	ConfigFile    string        `env:"CONFIG"`
}

func (config *Config) UnmarshalJSON(b []byte) error {
	type MyTypeAlias Config

	unmarshalledJSON := &struct {
		*MyTypeAlias
		StoreInterval interface{} `json:"store_interval"`
	}{
		MyTypeAlias: (*MyTypeAlias)(config),
	}
	err := json.Unmarshal(b, &unmarshalledJSON)
	if err != nil {
		return err
	}

	switch value := unmarshalledJSON.StoreInterval.(type) {
	case float64:
		config.StoreInterval = time.Duration(value) * time.Second
	case string:
		config.StoreInterval, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid duration: %#v", unmarshalledJSON)
	}

	return nil
}

func loadConfigFromFile(c *Config, filename string) (*Config, error) {
	log.Printf("Loading config from file '%s'", filename)
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	err = json.Unmarshal(fileBytes, c)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return c, nil
}

// RewriteConfigWithEnvs rewrites ENV values if the similiar flag is specified during application launch.
func GetConfig() (*Config, error) {
	c := &Config{
		Address:       "localhost:8080",
		StoreInterval: time.Duration(300 * time.Second),
		StoreFile:     "/tmp/devops-metrics-db.json",
		Restore:       true,
		DBAddress:     "",
		CryptoKey:     "",
	}

	c, err := PreFlagArgParse(c)
	if err != nil {
		return nil, err
	}

	flag.StringVar(&c.Address, "a", c.Address, "Socket to listen on")
	flag.DurationVar(&c.StoreInterval, "i", c.StoreInterval, "Save data interval")
	flag.StringVar(&c.StoreFile, "f", c.StoreFile, "File for saving data")
	flag.BoolVar(&c.Restore, "r", c.Restore, "Restore data from file")
	flag.StringVar(&c.DBAddress, "d", c.DBAddress, "Database address")
	flag.StringVar(&c.CryptoKey, "crypto-key", c.CryptoKey, "Path to private key")
	flag.StringVar(&c.Key, "k", "", "Encryption key")
	flag.StringVar(&c.ConfigFile, "config", "", "Config file name")
	flag.StringVar(&c.ConfigFile, "c", "", "Config file name")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	err = env.Parse(c)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return c, nil
}

func PreFlagArgParse(c *Config) (*Config, error) {
	var filename string
	var err error

	for num, i := range os.Args {
		if i != "-c" && i != "--config" && !strings.HasPrefix(i, "-c=") && !strings.HasPrefix(i, "--config=") {
			continue
		}

		if strings.Contains(i, "=") {
			filename = strings.Split(i, "=")[1]
			continue
		}
		if num != len(os.Args)-1 {
			filename = os.Args[num+1]
		}
	}

	for _, i := range os.Environ() {
		if !strings.Contains(i, "CONFIG=") {
			continue
		}
		filename = strings.Split(i, "=")[1]
	}

	if filename != "" {
		c, err = loadConfigFromFile(c, filename)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}
