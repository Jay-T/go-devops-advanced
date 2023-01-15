// Application for receiving metrics over WEB and storing in DB.
package server

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/Jay-T/go-devops.git/internal/utils/metric"
	_ "github.com/lib/pq"
)

const (
	gauge   = "gauge"
	counter = "counter"
)

// Server common interface for gRPC and HTTP server implementations.
type Server interface {
	StartServer(context.Context, StorageBackuper)
	StopServer(context.Context, context.CancelFunc, StorageBackuper)
}

// NewServer returns a gRPC or HTTP server depending on config GRPC flag.
func NewServer(ctx context.Context, cfg *Config, backuper StorageBackuper) (Server, error) {
	if cfg.GRPC {
		log.Printf("Running server in gRPC mode.")
		s, err := NewGRPCServer(ctx, cfg, backuper)
		if err != nil {
			log.Printf("Could not run GRPC server. Error: %s", err)
			return nil, err
		}
		return s, nil
	}

	log.Printf("Running server in HTTP mode.")
	s, err := NewHTTPService(ctx, cfg, backuper)
	if err != nil {
		log.Printf("Could not run HTTP server. Error: %s", err)
		return nil, err
	}
	return s, nil
}

// GenericService structure. Holds application config and db connector.
type GenericService struct {
	Cfg           *Config
	Metrics       map[string]metric.Metric
	Decryptor     *Decryptor
	backuper      StorageBackuper
	trustedSubnet *net.IPNet
}

// NewService returns GenericService with config parsed from flags or ENV vars.
func NewService(ctx context.Context, cfg *Config, backuper StorageBackuper) (*GenericService, error) {
	var s GenericService
	var err error

	s.Metrics = map[string]metric.Metric{}
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

	if s.Cfg.TrustedSubnet != "" {
		_, ipV4Net, err := net.ParseCIDR(s.Cfg.TrustedSubnet)
		if err != nil {
			return nil, err
		}

		s.trustedSubnet = ipV4Net
	}

	s.backuper = backuper
	return &s, nil
}

func (s GenericService) saveMetric(ctx context.Context, m *metric.Metric) {
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
