package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/caarlos0/env"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

const (
	gauge   = "gauge"
	counter = "counter"
)

type Metric struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
}

func (m Metric) GetValueInt() int64 {
	return int64(*m.Delta)
}

func (m Metric) GetValueFloat() float64 {
	return *m.Value
}

type Config struct {
	Address        string        `env:"ADDRESS" envDefault:"127.0.0.1:8080"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" envDefault:"10s"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" envDefault:"2s"`
	Key            string        `env:"KEY"`
}

type Agent struct {
	cfg Config
}

func (a Agent) AddHash(m *Metric) {
	var data string

	h := hmac.New(sha256.New, []byte(a.cfg.Key))
	switch m.MType {
	case gauge:
		data = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	case counter:
		data = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	}
	h.Write([]byte(data))
	m.Hash = hex.EncodeToString(h.Sum(nil))
}

func (a Agent) sendData(m *Metric) error {
	var url string
	if a.cfg.Key != "" {
		a.AddHash(m)
	}

	mSer, err := json.Marshal(*m)
	if err != nil {
		return err
	}
	url = fmt.Sprintf("http://%s/update/", a.cfg.Address)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(mSer))

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return err
}

// Gouroutine polls memory metrics each time it receives signal from syncChan
func (a Agent) getDataByInterval(ctx context.Context, dataChan chan<- Data, syncChan <-chan time.Time) {
	var rtm runtime.MemStats

	for {
		select {
		case <-syncChan:
			PollCount += 1
			runtime.ReadMemStats(&rtm)

			dataChan <- Data{name: "Alloc", gaugeValue: float64(rtm.Alloc)}
			dataChan <- Data{name: "TotalAlloc", gaugeValue: float64(rtm.TotalAlloc)}
			dataChan <- Data{name: "BuckHashSys", gaugeValue: float64(rtm.BuckHashSys)}
			dataChan <- Data{name: "Frees", gaugeValue: float64(rtm.Frees)}
			dataChan <- Data{name: "GCCPUFraction", gaugeValue: float64(rtm.GCCPUFraction)}
			dataChan <- Data{name: "GCSys", gaugeValue: float64(rtm.GCSys)}
			dataChan <- Data{name: "HeapAlloc", gaugeValue: float64(rtm.HeapAlloc)}
			dataChan <- Data{name: "HeapIdle", gaugeValue: float64(rtm.HeapIdle)}
			dataChan <- Data{name: "HeapInuse", gaugeValue: float64(rtm.HeapInuse)}
			dataChan <- Data{name: "HeapObjects", gaugeValue: float64(rtm.HeapObjects)}
			dataChan <- Data{name: "HeapReleased", gaugeValue: float64(rtm.HeapReleased)}
			dataChan <- Data{name: "HeapSys", gaugeValue: float64(rtm.HeapSys)}
			dataChan <- Data{name: "LastGC", gaugeValue: float64(rtm.LastGC)}
			dataChan <- Data{name: "Lookups", gaugeValue: float64(rtm.Lookups)}
			dataChan <- Data{name: "MCacheInuse", gaugeValue: float64(rtm.MCacheInuse)}
			dataChan <- Data{name: "MCacheSys", gaugeValue: float64(rtm.MCacheSys)}
			dataChan <- Data{name: "MSpanInuse", gaugeValue: float64(rtm.MSpanInuse)}
			dataChan <- Data{name: "MSpanSys", gaugeValue: float64(rtm.MSpanSys)}
			dataChan <- Data{name: "Mallocs", gaugeValue: float64(rtm.Mallocs)}
			dataChan <- Data{name: "NextGC", gaugeValue: float64(rtm.NextGC)}
			dataChan <- Data{name: "NumForcedGC", gaugeValue: float64(rtm.NumForcedGC)}
			dataChan <- Data{name: "NumGC", gaugeValue: float64(rtm.NumGC)}
			dataChan <- Data{name: "OtherSys", gaugeValue: float64(rtm.OtherSys)}
			dataChan <- Data{name: "PauseTotalNs", gaugeValue: float64(rtm.PauseTotalNs)}
			dataChan <- Data{name: "StackInuse", gaugeValue: float64(rtm.StackInuse)}
			dataChan <- Data{name: "StackSys", gaugeValue: float64(rtm.StackSys)}
			dataChan <- Data{name: "Sys", gaugeValue: float64(rtm.Sys)}
			dataChan <- Data{name: "RandomValue", gaugeValue: rand.Float64() * 100}
			dataChan <- Data{name: "PollCount", counterValue: int64(PollCount)}

		case <-ctx.Done():
			fmt.Println("getDataByInterval has been canceled successfully.")
			return
		}
	}
}

// Gouroutine polls memory metrics each time it receives signal from syncChan
func (a Agent) getMemDataByInterval(ctx context.Context, gaugeChan chan<- Data, syncChan <-chan time.Time) {
	for {
		select {
		case <-syncChan:
			v, _ := mem.VirtualMemory()
			gaugeChan <- Data{name: "TotalMemory", gaugeValue: float64(v.Free)}
			gaugeChan <- Data{name: "FreeMemory", gaugeValue: float64(v.Free)}

		case <-ctx.Done():
			fmt.Println("getMemDataByInterval has been canceled successfully.")
			return
		}
	}
}

// Polling time here depends on internal CPUPollTime value.
func (a Agent) getCPUDataByInterval(ctx context.Context, gaugeChan chan<- Data) {
	const CPUPollTime time.Duration = 10 * time.Second

	for {
		select {
		default:
			cSlice, _ := cpu.Percent(CPUPollTime, true)

			for i, c := range cSlice {
				gaugeChan <- Data{name: fmt.Sprintf("CPUutilization%d", i), gaugeValue: float64(c)}
			}

		case <-ctx.Done():
			fmt.Println("getCPUDataByInterval has been canceled successfully.")
			return
		}
	}
}

func (a Agent) sendBulkData(mList *[]Metric) error {
	var url string

	mSer, err := json.Marshal(*mList)
	if err != nil {
		return err
	}
	url = fmt.Sprintf("http://%s/updates/", a.cfg.Address)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(mSer))

	if err != nil {
		log.Println(err)
		return err
	}

	defer resp.Body.Close()
	return err
}

func (a Agent) sendDataByInterval(ctx context.Context, dataChan chan<- Data) {
	ticker := time.NewTicker(a.cfg.ReportInterval)
	for {
		select {
		case <-ticker.C:
			metricsSnapshot := metrics
			var mList []Metric

			for _, m := range metricsSnapshot {
				err := a.sendData(&m)
				if err != nil {
					log.Printf("metric: %s, error: %s", m.ID, err)
				}
				mList = append(mList, m)
				if m.ID == "PollCount" {
					PollCount = 0
					dataChan <- Data{name: "PollCount", counterValue: 0}
				}
			}
			if len(mList) > 0 {
				a.sendBulkData(&mList)
			}
		case <-ctx.Done():
			fmt.Println("Context has been canceled successfully.")
			return
		}
	}
}

type Data struct {
	name         string
	gaugeValue   float64
	counterValue int64
}

func newMetric(ctx context.Context, dataChan <-chan Data) {
	var m sync.Mutex
	for {
		select {
		case data := <-dataChan:
			m.Lock()
			if data.name == "PollCount" {
				metrics[data.name] = Metric{ID: data.name, MType: counter, Delta: &data.counterValue}
			} else {
				metrics[data.name] = Metric{ID: data.name, MType: gauge, Value: &data.gaugeValue}
			}
			m.Unlock()
		case <-ctx.Done():
			fmt.Println("newMetric has been canceled successfully.")
		}
	}
}

func closeApp() {
	log.Println("SIGINT!")
	os.Exit(1)
}

// Function syncronizes goroutines that poll system metrics by sending signal to syncChan;
// Goroutines that receive signal, poll system metrics with same interval.
func RunTicker(ctx context.Context, syncChan chan<- time.Time) {
	ticker := time.NewTicker(a.cfg.PollInterval)
	for {
		select {
		case t := <-ticker.C:
			syncChan <- t
		case <-ctx.Done():
			fmt.Println("RunTicker has been canceled successfully.")
		}
	}
}

func RewriteConfigWithEnvs(a *Agent) {
	if _, present := os.LookupEnv("ADDRESS"); !present {
		a.cfg.Address = *address
	}
	if _, present := os.LookupEnv("REPORT_INTERVAL"); !present {
		a.cfg.ReportInterval = *reportInterval
	}
	if _, present := os.LookupEnv("POLL_INTERVAL"); !present {
		a.cfg.PollInterval = *pollInterval
	}
	if _, present := os.LookupEnv("KEY"); !present {
		a.cfg.Key = *key
	}
}

var (
	address        *string
	reportInterval *time.Duration
	pollInterval   *time.Duration
	a              Agent
	PollCount      int64
	key            *string
)

var metrics = make(map[string]Metric)

func main() {

	err := env.Parse(&a.cfg)
	if err != nil {
		log.Fatal(err)
	}

	address = flag.String("a", "localhost:8080", "Address for sending data to")
	reportInterval = flag.Duration("r", time.Duration(10*time.Second), "Metric report to server interval")
	pollInterval = flag.Duration("p", time.Duration(2*time.Second), "Metric poll interval")
	key = flag.String("k", "testkey", "Encryption key")
	flag.Parse()
	RewriteConfigWithEnvs(&a)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	dataChan := make(chan Data)
	syncChan := make(chan time.Time)

	ctx, cancel := context.WithCancel(context.Background())
	go RunTicker(ctx, syncChan)
	go newMetric(ctx, dataChan)
	go a.getDataByInterval(ctx, dataChan, syncChan)
	go a.getMemDataByInterval(ctx, dataChan, syncChan)
	go a.getCPUDataByInterval(ctx, dataChan)
	go a.sendDataByInterval(ctx, dataChan)
	<-sigChan
	cancel()
	closeApp()
}
