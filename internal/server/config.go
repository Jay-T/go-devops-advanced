package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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
  -r bool Restore data from file (default true)
  -t string Trusted subnet
`

const (
	defaultAddress       string        = "localhost:8080"
	defaultStoreInterval time.Duration = time.Duration(300 * time.Second)
	defaultStoreFile     string        = "/tmp/devops-metrics-db.json"
	defaultRestore       bool          = false
	defaultDBAddress     string        = ""
	defaultCryptoKey     string        = ""
	defaultKey           string        = ""
	defaultConfig        string        = ""
	defaultTrustedSubnet string        = ""
)

// Config structure. Used for application configuration.
type Config struct {
	Address       string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	StoreFile     string        `env:"STORE_FILE"`
	Restore       bool          `env:"RESTORE"`
	Key           string        `env:"KEY"`
	DBAddress     string        `env:"DATABASE_DSN"`
	CryptoKey     string        `env:"CRYPTO_KEY"`
	ConfigFile    string        `env:"CONFIG"`
	TrustedSubnet string        `env:"TRUSTED_SUBNET"`
}

type ConfigFile struct {
	Address       string        `json:"address"`
	StoreInterval time.Duration `json:"store_interval"`
	StoreFile     string        `json:"store_file"`
	Restore       bool          `json:"restore"`
	DBAddress     string        `json:"database_dsn"`
	CryptoKey     string        `json:"crypto_key"`
	TrustedSubnet string        `json:"trusted_subnet"`
}

func (config *ConfigFile) UnmarshalJSON(b []byte) error {
	type MyTypeAlias ConfigFile

	unmarshalledJSON := &struct {
		*MyTypeAlias
		StoreInterval string `json:"store_interval"`
	}{
		MyTypeAlias: (*MyTypeAlias)(config),
	}
	err := json.Unmarshal(b, &unmarshalledJSON)
	if err != nil {
		return err
	}

	config.StoreInterval, err = time.ParseDuration(unmarshalledJSON.StoreInterval)
	if err != nil {
		return err
	}

	return nil
}

func loadConfigFromFile(c *Config) error {
	if c.ConfigFile == "" {
		return nil
	}

	log.Printf("Loading config from file '%s'", c.ConfigFile)
	fileBytes, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	var cfgFromFile ConfigFile
	err = json.Unmarshal(fileBytes, &cfgFromFile)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	if c.Address == defaultAddress && cfgFromFile.Address != "" {
		c.Address = cfgFromFile.Address
	}

	if c.StoreInterval == defaultStoreInterval && cfgFromFile.StoreInterval != 0 {
		c.StoreInterval = cfgFromFile.StoreInterval
	}

	if c.StoreFile == defaultStoreFile && cfgFromFile.StoreFile != "" {
		c.StoreFile = cfgFromFile.StoreFile
	}

	if !c.Restore && cfgFromFile.Restore {
		c.Restore = cfgFromFile.Restore
	}

	if c.DBAddress == defaultDBAddress && cfgFromFile.DBAddress != "" {
		c.DBAddress = cfgFromFile.DBAddress
	}

	if c.TrustedSubnet == defaultTrustedSubnet && cfgFromFile.TrustedSubnet != "" {
		c.TrustedSubnet = cfgFromFile.TrustedSubnet
	}

	if c.CryptoKey == defaultCryptoKey && cfgFromFile.CryptoKey != "" {
		c.CryptoKey = cfgFromFile.CryptoKey
	}

	return nil
}

// RewriteConfigWithEnvs rewrites ENV values if the similiar flag is specified during application launch.
func GetConfig() (*Config, error) {
	c := &Config{}

	flag.StringVar(&c.Address, "a", defaultAddress, "Socket to listen on")
	flag.DurationVar(&c.StoreInterval, "i", defaultStoreInterval, "Save data interval")
	flag.StringVar(&c.StoreFile, "f", defaultStoreFile, "File for saving data")
	flag.BoolVar(&c.Restore, "r", defaultRestore, "Restore data from file")
	flag.StringVar(&c.DBAddress, "d", defaultDBAddress, "Database address")
	flag.StringVar(&c.CryptoKey, "crypto-key", defaultCryptoKey, "Path to private key")
	flag.StringVar(&c.Key, "k", defaultKey, "Encryption key")
	flag.StringVar(&c.TrustedSubnet, "t", defaultTrustedSubnet, "Trusted subnet")
	flag.StringVar(&c.ConfigFile, "config", defaultConfig, "Config file name")
	flag.StringVar(&c.ConfigFile, "c", defaultConfig, "Config file name")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	err := env.Parse(c)
	if err != nil {
		log.Fatal(err)
	}

	err = loadConfigFromFile(c)
	if err != nil {
		log.Fatal(err)
	}

	return c, nil
}
