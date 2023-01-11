package agent

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
  -a string Address for sending data to (default "localhost:8080")
  -crypto-key string Path to public key
  -k string Encryption key (default "testkey")
  -p duration Metric poll interval (default 2s)
  -r duration Metric report to server interval (default 10s)
  -intf string Local network interface
`

const (
	defaultAddress        string        = "localhost:8080"
	defaultReportInterval time.Duration = time.Duration(10 * time.Second)
	defaultPollInterval   time.Duration = time.Duration(2 * time.Second)
	defaultCryptoKey      string        = ""
	defaultKey            string        = ""
	defaultLocalInterface string        = ""
)

// Config structure. Used for application configuration.
type Config struct {
	Address        string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	Key            string        `env:"KEY"`
	CryptoKey      string        `env:"CRYPTO_KEY"`
	ConfigFile     string        `env:"CONFIG"`
	LocalInterface string        `env:"LOCAL_INTERFACE"`
	GRPC           bool
}

type ConfigFile struct {
	Address        string        `json:"address"`
	ReportInterval time.Duration `json:"report_interval"`
	PollInterval   time.Duration `json:"poll_interval"`
	CryptoKey      string        `json:"crypto_key"`
	LocalInterface string        `json:"local_interface"`
}

func (config *ConfigFile) UnmarshalJSON(b []byte) error {
	type MyTypeAlias ConfigFile

	unmarshalledJSON := &struct {
		*MyTypeAlias
		ReportInterval string `json:"report_interval"`
		PollInterval   string `json:"poll_interval"`
	}{
		MyTypeAlias: (*MyTypeAlias)(config),
	}
	err := json.Unmarshal(b, &unmarshalledJSON)
	if err != nil {
		return err
	}

	config.ReportInterval, err = time.ParseDuration(unmarshalledJSON.ReportInterval)
	if err != nil {
		return err
	}
	config.PollInterval, err = time.ParseDuration(unmarshalledJSON.PollInterval)
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
		return err
	}

	var cfgFromFile ConfigFile
	err = json.Unmarshal(fileBytes, &cfgFromFile)
	if err != nil {
		return err
	}

	if c.Address == defaultAddress && cfgFromFile.Address != "" {
		c.Address = cfgFromFile.Address
	}

	if c.ReportInterval == defaultReportInterval && cfgFromFile.ReportInterval != 0 {
		c.ReportInterval = cfgFromFile.ReportInterval
	}

	if c.PollInterval == defaultPollInterval && cfgFromFile.PollInterval != 0 {
		c.PollInterval = cfgFromFile.PollInterval
	}

	if c.CryptoKey == defaultCryptoKey && cfgFromFile.CryptoKey != "" {
		c.CryptoKey = cfgFromFile.CryptoKey
	}

	if c.LocalInterface == defaultLocalInterface && cfgFromFile.LocalInterface != "" {
		c.LocalInterface = cfgFromFile.LocalInterface
	}

	return nil
}

// RewriteConfigWithEnvs rewrites ENV values if the similiar flag is specified during application launch.
func GetConfig() (*Config, error) {
	c := &Config{}

	flag.StringVar(&c.Address, "a", defaultAddress, "Address for sending data to")
	flag.DurationVar(&c.ReportInterval, "r", defaultReportInterval, "Metric report to server interval")
	flag.DurationVar(&c.PollInterval, "p", defaultPollInterval, "Metric poll interval")
	flag.StringVar(&c.CryptoKey, "crypto-key", defaultCryptoKey, "Path to public key")
	flag.StringVar(&c.Key, "k", defaultKey, "Encryption key")
	flag.StringVar(&c.LocalInterface, "intf", defaultLocalInterface, "Local network interface")
	flag.StringVar(&c.ConfigFile, "config", "", "Config file name")
	flag.StringVar(&c.ConfigFile, "c", "", "Config file name")
	flag.BoolVar(&c.GRPC, "grpc", false, "Run as gRPC service")
	flag.Parse()
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	err := env.Parse(c)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	err = loadConfigFromFile(c)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return c, nil
}
