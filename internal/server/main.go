// Application for receiving metrics over WEB and storing in DB.
package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

const (
	gauge   = "gauge"
	counter = "counter"
)

var metrics = make(map[string]Metric)
var dataMap = make(map[string]float64)

// vars for application configuration.
var (
	address       *string
	restore       *bool
	storeInterval *time.Duration
	storeFile     *string
	s             Service
	key           *string
	db            *string
)

// Config structure. Used for application configuration.
type Config struct {
	Address       string        `env:"ADDRESS" envDefault:"127.0.0.1:8080"`
	StoreInterval time.Duration `env:"STORE_INTERVAL" envDefault:"300s"`
	StoreFile     string        `env:"STORE_FILE" envDefault:"/tmp/devops-metrics-db.json"`
	Restore       bool          `env:"RESTORE" envDefault:"true"`
	Key           string        `env:"KEY"`
	DB            string        `env:"DATABASE_DSN"`
}

// Metric struct. Describes metric message format.
type Metric struct {
	ID    string   `json:"id"`              // metric's name
	MType string   `json:"type"`            // parameter taking value of gauge or counter
	Delta *int64   `json:"delta,omitempty"` // metric value in case of MType == counter
	Value *float64 `json:"value,omitempty"` // metric value in case of MType == gauge
	Hash  string   `json:"hash,omitempty"`  // hash value
}

// GetBody parses HTTP request's body and returns Metric.
func GetBody(r *http.Request) (*Metric, error) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		return nil, err
	}
	m := &Metric{}
	err = json.Unmarshal(body, m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Service structure. Holds application config and db connector.
type Service struct {
	Cfg Config
	DB  *sql.DB
}

// NewService returns Service with config parsed from flags or ENV vars.
func NewService() (*Service, error) {
	err := env.Parse(&s.Cfg)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	address = flag.String("a", "localhost:8080", "Socket to listen on")
	restore = flag.Bool("r", true, "Restore data from file")
	storeInterval = flag.Duration("i", time.Duration(300*time.Second), "Save data interval")
	storeFile = flag.String("f", "/tmp/devops-metrics-db.json", "File for saving data")
	key = flag.String("k", "", "Encryption key")
	db = flag.String("d", "", "Database address")
	flag.Parse()
	RewriteConfigWithEnvs(&s)
	return &s, nil
}

func (s Service) saveMetric(ctx context.Context, m *Metric) {
	switch m.MType {
	case counter:
		if metrics[m.ID].Delta == nil {
			metrics[m.ID] = *m
		} else {
			*metrics[m.ID].Delta += *m.Delta
		}
	case gauge:
		metrics[m.ID] = *m
	default:
		log.Printf("Metric type '%s' is not expected. Skipping.", m.MType)
	}
	if s.DB != nil {
		err := s.SaveMetricToDB(ctx)
		if err != nil {
			log.Print(err.Error())
		}
	} else if s.Cfg.StoreFile != "" && s.Cfg.StoreInterval == time.Duration(0) {
		s.SaveMetricToFile()
	}
}

// RestoreMetricFromFile loads metrics from local file during application init.
func (s Service) RestoreMetricFromFile() error {
	flags := os.O_RDONLY | os.O_CREATE
	consumer, err := NewConsumer(s.Cfg.StoreFile, flags)
	if err != nil {
		return err
	}
	consumer.ReadEvents()
	return nil
}

// SaveMetricToFile saves metrics to local file.
func (s *Service) SaveMetricToFile() {
	var MetricList []Metric
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	producer, err := NewProducer(s.Cfg.StoreFile, flags)
	if err != nil {
		log.Fatal(err)
	}

	for _, metric := range metrics {
		MetricList = append(MetricList, metric)
	}
	if err := producer.WriteMetric(&MetricList); err != nil {
		log.Fatal(err)
	}
	producer.Close()
}

// StartRecordInterval preiodically saves metrics.
func (s Service) StartRecordInterval(ctx context.Context) {
	ticker := time.NewTicker(s.Cfg.StoreInterval)
	for {
		select {
		case <-ticker.C:
			if s.DB != nil {
				err := s.SaveMetricToDB(ctx)
				if err != nil {
					log.Print(err.Error())
				}
			} else {
				s.SaveMetricToFile()
			}
		case <-ctx.Done():
			fmt.Println("Context has been canceled successfully.")
			return
		}
	}
}

// StartServer launches HTTP server.
func (s Service) StartServer(ctx context.Context) {
	r := chi.NewRouter()
	r.Use(gzipHandle)
	r.Mount("/debug", middleware.Profiler())
	// old methods
	r.Post("/update/gauge/{metricName}/{metricValue}", s.SetMetricOldHandler(ctx))
	r.Post("/update/counter/{metricName}/{metricValue}", s.SetMetricOldHandler(ctx))
	r.Post("/update/*", NotImplemented)
	r.Post("/update/{metricName}/", NotFound)
	r.Get("/value/*", GetMetricOldHandler)
	r.Get("/", GetAllMetricHandler)
	// new methods
	r.Post("/update/", s.SetMetricHandler(ctx))
	r.Post("/updates/", s.SetMetricListHandler(ctx))
	r.Post("/value/", s.GetMetricHandler)
	r.Get("/ping", s.PingDBHandler)

	srv := &http.Server{
		Addr:    s.Cfg.Address,
		Handler: r,
	}
	srv.SetKeepAlivesEnabled(false)
	log.Printf("Listening socket: %s", s.Cfg.Address)
	log.Fatal(srv.ListenAndServe())
}

// GenerateHash generates sha256 hash for http request's body fields for message validation.
func (s Service) GenerateHash(m *Metric) []byte {
	var data string

	h := hmac.New(sha256.New, []byte(s.Cfg.Key))
	switch m.MType {
	case gauge:
		data = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	case counter:
		data = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	}
	h.Write([]byte(data))
	return h.Sum(nil)
}

// CloseApp closes http application.
func CloseApp() {
	log.Println("SIGINT!")
	os.Exit(1)
}

// RewriteConfigWithEnvs rewrites ENV values if the similiar flag is specified during application launch.
func RewriteConfigWithEnvs(s *Service) {
	if _, present := os.LookupEnv("ADDRESS"); !present {
		s.Cfg.Address = *address
	}
	if _, present := os.LookupEnv("STORE_INTERVAL"); !present {
		s.Cfg.StoreInterval = *storeInterval
	}
	if _, present := os.LookupEnv("STORE_FILE"); !present {
		s.Cfg.StoreFile = *storeFile
	}
	if _, present := os.LookupEnv("RESTORE"); !present {
		s.Cfg.Restore = *restore
	}
	if _, present := os.LookupEnv("KEY"); !present {
		s.Cfg.Key = *key
	}
	if _, present := os.LookupEnv("DATABASE_DSN"); !present {
		s.Cfg.DB = *db
	}
}
