// Application for receiving metrics over WEB and storing in DB.
package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"time"

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
type GenericService struct {
	Cfg       *Config
	Metrics   map[string]Metric
	Decryptor *Decryptor
	backuper  StorageBackuper
}

// NewService returns Service with config parsed from flags or ENV vars.
func NewService(ctx context.Context, cfg *Config, backuper StorageBackuper) (*GenericService, error) {
	var s GenericService
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

	s.backuper = backuper
	return &s, nil
}

func (s GenericService) saveMetric(ctx context.Context, m *Metric) {
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
	err := s.backuper.SaveMetric(ctx, s.Metrics)
	if err != nil {
		log.Print(err)
	}
}

// StartRecordInterval preiodically saves metrics.
func (s GenericService) StartRecordInterval(ctx context.Context, backuper StorageBackuper) {
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

// GenerateHash generates sha256 hash for http request's body fields for message validation.
func (s GenericService) GenerateHash(m *Metric) []byte {
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
func (s GenericService) StopServer(ctx context.Context, cancel context.CancelFunc, backuper StorageBackuper) {
	log.Println("Received a SIGINT! Stopping application")
	err := backuper.SaveMetric(ctx, s.Metrics)
	if err != nil {
		log.Print(err)
	}
	cancel()
	log.Println("Canceled all goroutines.")
	os.Exit(1)
}
