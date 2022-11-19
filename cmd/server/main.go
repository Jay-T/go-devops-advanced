package main

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
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env"
	"github.com/go-chi/chi"
	_ "github.com/lib/pq"
)

const (
	gauge   = "gauge"
	counter = "counter"
)

var metrics = make(map[string]Metric)
var dataMap = make(map[string]float64)

type Config struct {
	Address       string        `env:"ADDRESS" envDefault:"127.0.0.1:8080"`
	StoreInterval time.Duration `env:"STORE_INTERVAL" envDefault:"300s"`
	StoreFile     string        `env:"STORE_FILE" envDefault:"/tmp/devops-metrics-db.json"`
	Restore       bool          `env:"RESTORE" envDefault:"true"`
	Key           string        `env:"KEY"`
	DB            string        `env:"DATABASE_DSN"`
}

type Metric struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
}

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

type Service struct {
	cfg Config
	db  *sql.DB
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
	if s.db != nil {
		s.SaveMetricToDB(ctx)
	} else if s.cfg.StoreFile != "" && s.cfg.StoreInterval == time.Duration(0) {
		s.SaveMetricToFile()
	}
}

func (s Service) RestoreMetricFromFile() error {
	flags := os.O_RDONLY | os.O_CREATE
	consumer, err := NewConsumer(s.cfg.StoreFile, flags)
	if err != nil {
		return err
	}
	consumer.ReadEvents()
	return nil
}

func (s *Service) SaveMetricToFile() {
	var MetricList []Metric
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	producer, err := NewProducer(s.cfg.StoreFile, flags)
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

func (s Service) StartRecordInterval(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.StoreInterval)
	for {
		select {
		case <-ticker.C:
			if s.db != nil {
				s.SaveMetricToDB(ctx)
			} else {
				s.SaveMetricToFile()
			}
		case <-ctx.Done():
			fmt.Println("Context has been canceled successfully.")
			return
		}
	}
}

func (s Service) StartServer(ctx context.Context) {
	r := chi.NewRouter()
	r.Use(gzipHandle)
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
		Addr:    s.cfg.Address,
		Handler: r,
	}
	srv.SetKeepAlivesEnabled(false)
	log.Printf("Listening socket: %s", s.cfg.Address)
	log.Fatal(srv.ListenAndServe())
}

func (s Service) GenerateHash(m *Metric) []byte {
	var data string

	h := hmac.New(sha256.New, []byte(s.cfg.Key))
	switch m.MType {
	case gauge:
		data = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	case counter:
		data = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	}
	h.Write([]byte(data))
	return h.Sum(nil)
}

func CloseApp() {
	log.Println("SIGINT!")
	os.Exit(1)
}

func RewriteConfigWithEnvs(s *Service) {
	if _, present := os.LookupEnv("ADDRESS"); !present {
		s.cfg.Address = *address
	}
	if _, present := os.LookupEnv("STORE_INTERVAL"); !present {
		s.cfg.StoreInterval = *storeInterval
	}
	if _, present := os.LookupEnv("STORE_FILE"); !present {
		s.cfg.StoreFile = *storeFile
	}
	if _, present := os.LookupEnv("RESTORE"); !present {
		s.cfg.Restore = *restore
	}
	if _, present := os.LookupEnv("KEY"); !present {
		s.cfg.Key = *key
	}
	if _, present := os.LookupEnv("DATABASE_DSN"); !present {
		s.cfg.DB = *db
	}
}

var (
	address       *string
	restore       *bool
	storeInterval *time.Duration
	storeFile     *string
	s             Service
	key           *string
	db            *string
)

func main() {
	err := env.Parse(&s.cfg)
	if err != nil {
		log.Fatal(err)
	}

	address = flag.String("a", "localhost:8080", "Socket to listen on")
	restore = flag.Bool("r", true, "Restore data from file")
	storeInterval = flag.Duration("i", time.Duration(300*time.Second), "Save data interval")
	storeFile = flag.String("f", "/tmp/devops-metrics-db.json", "File for saving data")
	key = flag.String("k", "", "Encryption key")
	db = flag.String("d", "", "Database address")
	flag.Parse()
	RewriteConfigWithEnvs(&s)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())
	if s.cfg.DB != "" {
		s.db, err = sql.Open("postgres", s.cfg.DB)
		if err != nil {
			log.Println(err)
		}
		defer s.db.Close()
		s.DBInit(ctx)
	}

	go s.StartServer(ctx)

	if s.cfg.Restore {
		if s.db != nil {
			log.Printf("Restoring metrics from DB")
			if err := s.RestoreMetricFromDB(ctx); err != nil {
				log.Println(err)
			}
		} else if s.cfg.StoreFile != "" {
			log.Printf("Restoring metrics from file '%s'", s.cfg.StoreFile)
			if err := s.RestoreMetricFromFile(); err != nil {
				log.Println(err)
			}
		}
	}

	if s.cfg.StoreFile != "" && s.cfg.StoreInterval > time.Duration(0) {
		log.Printf("Saving results to file with interval %s", s.cfg.StoreInterval)
		go s.StartRecordInterval(ctx)
	}

	<-sigChan
	if s.db != nil {
		s.SaveMetricToDB(ctx)
	} else {
		s.SaveMetricToFile()
	}
	cancel()
	CloseApp()
}
