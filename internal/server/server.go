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

// var metrics = make(map[string]Metric)

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

// Config structure. Used for application configuration.
type Config struct {
	Address       string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	StoreFile     string        `env:"STORE_FILE"`
	Restore       bool          `env:"RESTORE"`
	Key           string        `env:"KEY"`
	DBAddress     string        `env:"DATABASE_DSN"`
}

// RewriteConfigWithEnvs rewrites ENV values if the similiar flag is specified during application launch.
func GetConfig() (*Config, error) {
	c := &Config{}

	flag.StringVar(&c.Address, "a", "localhost:8080", "Socket to listen on")
	flag.DurationVar(&c.StoreInterval, "i", time.Duration(300*time.Second), "Save data interval")
	flag.StringVar(&c.StoreFile, "f", "/tmp/devops-metrics-db.json", "File for saving data")
	flag.BoolVar(&c.Restore, "r", true, "Restore data from file")
	flag.StringVar(&c.Key, "k", "", "Encryption key")
	flag.StringVar(&c.DBAddress, "d", "", "Database address")

	flag.Parse()

	err := env.Parse(c)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return c, nil
}

// Service structure. Holds application config and db connector.
type Service struct {
	Cfg     *Config
	DB      *sql.DB
	Metrics map[string]Metric
}

// NewService returns Service with config parsed from flags or ENV vars.
func NewService(ctx context.Context) (*Service, error) {
	var s Service
	s.Metrics = map[string]Metric{}

	cfg, err := GetConfig()
	if err != nil {
		log.Fatal("Error while getting config.", err.Error())
	}

	s.Cfg = cfg
	if s.Cfg.DBAddress != "" {
		db, err := s.NewServiceDB(ctx)
		if err != nil {
			log.Print("Error during DB connection.")
			log.Fatal(err)
		}
		s.DB = db
	}

	if s.Cfg.Restore {
		if s.DB != nil {
			log.Printf("Restoring metrics from DB")
			if err := s.RestoreMetricFromDB(ctx); err != nil {
				log.Println(err)
			}
		} else if s.Cfg.StoreFile != "" {
			log.Printf("Restoring metrics from file '%s'", s.Cfg.StoreFile)
			if err := s.RestoreMetricFromFile(); err != nil {
				log.Println(err)
			}
		}
	}

	if s.Cfg.StoreFile != "" && s.Cfg.StoreInterval > time.Duration(0) {
		log.Printf("Saving results to file with interval %s", s.Cfg.StoreInterval)
		go s.StartRecordInterval(ctx)
	}

	return &s, nil
}

func (s Service) saveMetric(ctx context.Context, m *Metric) {
	switch m.MType {
	case counter:
		if s.Metrics[m.ID].Delta == nil {
			s.Metrics[m.ID] = *m
		} else {
			*s.Metrics[m.ID].Delta += *m.Delta
		}
	case gauge:
		s.Metrics[m.ID] = *m
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
	consumer.ReadEvents(s.Metrics)
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

	for _, metric := range s.Metrics {
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
			log.Println("Context has been canceled successfully.")
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
	r.Get("/value/*", s.GetMetricOldHandler)
	r.Get("/", s.GetAllMetricHandler)
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
func (s Service) StopServer(ctx context.Context, cancel context.CancelFunc) {
	log.Println("Received a SIGINT! Stopping application")
	if s.DB != nil {
		log.Println("Saving data to DB.")
		err := s.SaveMetricToDB(ctx)
		if err != nil {
			log.Print(err.Error())
		}
		err = s.DB.Close()
		if err != nil {
			log.Print(err.Error())
		}
		log.Println("DB connection is closed.")
	} else {
		s.SaveMetricToFile()
	}

	cancel()
	log.Println("Canceled all goroutines.")
	os.Exit(1)
}
