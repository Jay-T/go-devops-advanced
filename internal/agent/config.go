package agent

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
  -a string Address for sending data to (default "localhost:8080")
  -crypto-key string Path to public key
  -k string Encryption key (default "testkey")
  -p duration Metric poll interval (default 2s)
  -r duration Metric report to server interval (default 10s)
`

// Config structure. Used for application configuration.
type Config struct {
	Address        string        `env:"ADDRESS" json:"address"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" json:"report_interval"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" json:"poll_interval"`
	Key            string        `env:"KEY"`
	CryptoKey      string        `env:"CRYPTO_KEY" json:"crypto_key"`
	ConfigFile     string        `env:"CONFIG"`
}

func (config *Config) UnmarshalJSON(b []byte) error {
	type MyTypeAlias Config

	unmarshalledJSON := &struct {
		*MyTypeAlias
		ReportInterval interface{} `json:"report_interval"`
		PollInterval   interface{} `json:"poll_interval"`
	}{
		MyTypeAlias: (*MyTypeAlias)(config),
	}
	err := json.Unmarshal(b, &unmarshalledJSON)
	if err != nil {
		return err
	}

	switch value := unmarshalledJSON.ReportInterval.(type) {
	case float64:
		config.ReportInterval = time.Duration(value) * time.Second
	case string:
		config.ReportInterval, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid duration: %#v", unmarshalledJSON)
	}

	switch value := unmarshalledJSON.PollInterval.(type) {
	case float64:
		config.PollInterval = time.Duration(value) * time.Second
	case string:
		config.PollInterval, err = time.ParseDuration(value)
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
		return nil, err
	}

	err = json.Unmarshal(fileBytes, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// RewriteConfigWithEnvs rewrites ENV values if the similiar flag is specified during application launch.
func GetConfig() (*Config, error) {
	c := &Config{
		Address:        "localhost:8080",
		ReportInterval: time.Duration(10 * time.Second),
		PollInterval:   time.Duration(2 * time.Second),
		CryptoKey:      "",
	}

	c, err := PreFlagArgParse(c)
	if err != nil {
		return nil, err
	}

	flag.StringVar(&c.Address, "a", c.Address, "Address for sending data to")
	flag.DurationVar(&c.ReportInterval, "r", c.ReportInterval, "Metric report to server interval")
	flag.DurationVar(&c.PollInterval, "p", c.PollInterval, "Metric poll interval")
	flag.StringVar(&c.CryptoKey, "crypto-key", c.CryptoKey, "Path to public key")
	flag.StringVar(&c.Key, "k", "testkey", "Encryption key")
	flag.StringVar(&c.ConfigFile, "config", "", "Config file name")
	flag.StringVar(&c.ConfigFile, "c", "", "Config file name")
	flag.Parse()
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
