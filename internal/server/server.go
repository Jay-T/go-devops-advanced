// Application for receiving metrics over WEB and storing in DB.
package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

const (
	gauge   = "gauge"
	counter = "counter"
)

// Metric struct. Describes metric message format.
type Metric struct {
	ID    string   `json:"id"`              // metric's name
	MType string   `json:"type"`            // parameter taking value of gauge or counter
	Delta *int64   `json:"delta,omitempty"` // metric value in case of MType == counter
	Value *float64 `json:"value,omitempty"` // metric value in case of MType == gauge
	Hash  string   `json:"hash,omitempty"`  // hash value
}

// Service structure. Holds application config and db connector.
type Service struct {
	Cfg       *Config
	Metrics   map[string]Metric
	Decryptor *Decryptor
}

// NewService returns Service with config parsed from flags or ENV vars.
func NewService(ctx context.Context, cfg *Config, backuper StorageBackuper) (*Service, error) {
	var s Service
	var err error

	s.Metrics = map[string]Metric{}
	s.Cfg = cfg

	if s.Cfg.Restore {
		err = backuper.RestoreMetrics(ctx, s.Metrics)
		if err != nil {
			log.Print("Error during data restoration.")
			return nil, err
		}
	}

	if s.Cfg.StoreFile != "" && s.Cfg.StoreInterval > time.Duration(0) {
		log.Printf("Saving results to storage with interval %s", s.Cfg.StoreInterval)
		go s.StartRecordInterval(ctx, backuper)
	}

	if s.Cfg.CryptoKey != "" {
		s.Decryptor, err = NewDecryptor(s.Cfg.CryptoKey)
		if err != nil {
			return nil, err
		}
		log.Print("Crypto is enabled")
	}

	return &s, nil
}

// GetBody parses HTTP request's body and returns Metric.
func (s *Service) GetBody(r *http.Request) (*Metric, error) {
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

func (s Service) saveMetric(ctx context.Context, backuper StorageBackuper, m *Metric) {
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
	err := backuper.SaveMetric(ctx, s.Metrics)
	if err != nil {
		log.Print(err)
	}
}

// StartRecordInterval preiodically saves metrics.
func (s Service) StartRecordInterval(ctx context.Context, backuper StorageBackuper) {
	ticker := time.NewTicker(s.Cfg.StoreInterval)
	for {
		select {
		case <-ticker.C:
			err := backuper.SaveMetric(ctx, s.Metrics)
			if err != nil {
				log.Print(err.Error())
			}
		case <-ctx.Done():
			log.Println("Context has been canceled successfully.")
			return
		}
	}
}

// StartServer launches HTTP server.
func (s Service) StartServer(ctx context.Context, backuper StorageBackuper) {
	r := chi.NewRouter()
	// middlewares
	r.Use(gzipHandle)
	if s.Decryptor != nil {
		r.Use(s.decryptHandler)
	}
	r.Mount("/debug", middleware.Profiler())
	// old methods
	r.Post("/update/gauge/{metricName}/{metricValue}", s.SetMetricOldHandler(ctx, backuper))
	r.Post("/update/counter/{metricName}/{metricValue}", s.SetMetricOldHandler(ctx, backuper))
	r.Post("/update/*", NotImplemented)
	r.Post("/update/{metricName}/", NotFound)
	r.Get("/value/*", s.GetMetricOldHandler)
	r.Get("/", s.GetAllMetricHandler)
	// new methods
	r.Post("/update/", s.SetMetricHandler(ctx, backuper))
	r.Post("/updates/", s.SetMetricListHandler(ctx, backuper))
	r.Post("/value/", s.GetMetricHandler)
	r.Get("/ping", backuper.CheckStorageStatus)

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
func (s Service) StopServer(ctx context.Context, cancel context.CancelFunc, backuper StorageBackuper) {
	log.Println("Received a SIGINT! Stopping application")
	err := backuper.SaveMetric(ctx, s.Metrics)
	if err != nil {
		log.Print(err)
	}
	cancel()
	log.Println("Canceled all goroutines.")
	os.Exit(1)
}
