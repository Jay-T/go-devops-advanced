// Application collects system metrics and sends to server.
package agent

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
	"runtime"
	"sync"
	"time"

	"github.com/caarlos0/env"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

const (
	gauge   = "gauge"
	counter = "counter"
)

var (
	PollCount int64
)

// Metric struct describes format of metric messages.
type Metric struct {
	ID    string   `json:"id"`              // metric's name
	MType string   `json:"type"`            // parameter taking value of gauge or counter
	Delta *int64   `json:"delta,omitempty"` // metric value in case of MType == counter
	Value *float64 `json:"value,omitempty"` // metric value in case of MType == gauge
	Hash  string   `json:"hash,omitempty"`  // hash value
}

// GetValueInt returns pointer to int64 value.
func (m Metric) GetValueInt() int64 {
	return int64(*m.Delta)
}

// GetValueFloat returns pointer to float64 value.
func (m Metric) GetValueFloat() float64 {
	return *m.Value
}

// Config struct describes application config.
type Config struct {
	Address        string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	Key            string        `env:"KEY"`
	CryptoKey      string        `env:"CRYPTO_KEY"`
}

// RewriteConfigWithEnvs rewrites values from ENV variables if same variable is specified as flag.
func GetConfig() (*Config, error) {
	c := &Config{}

	flag.StringVar(&c.Address, "a", "localhost:8080", "Address for sending data to")
	flag.DurationVar(&c.ReportInterval, "r", time.Duration(10*time.Second), "Metric report to server interval")
	flag.DurationVar(&c.PollInterval, "p", time.Duration(2*time.Second), "Metric poll interval")
	flag.StringVar(&c.Key, "k", "testkey", "Encryption key")
	flag.StringVar(&c.CryptoKey, "crypto-key", "", "Path to public key")
	flag.Parse()

	err := env.Parse(c)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return c, nil
}

// Data struct describes message format between goroutines
type Data struct {
	name         string
	gaugeValue   float64
	counterValue int64
}

// Agent struct accepts Config and handles all metrics manipulations.
type Agent struct {
	Cfg       *Config
	Metrics   map[string]Metric
	l         sync.RWMutex
	Encryptor *Encryptor
}

// NewAgent configures Agent and returns pointer on it.
func NewAgent() (*Agent, error) {
	var a Agent

	cfg, err := GetConfig()
	if err != nil {
		log.Fatal("Error while getting config.", err.Error())
	}
	a.Metrics = map[string]Metric{}
	a.Cfg = cfg

	if a.Cfg.CryptoKey != "" {
		a.Encryptor, err = NewEncryptor(a.Cfg.CryptoKey)
		if err != nil {
			return nil, err
		}
	}

	return &a, nil
}

// AddHash computes hash for Metric fields for validation before sending it to server.
func (a *Agent) AddHash(m *Metric) {
	var data string

	h := hmac.New(sha256.New, []byte(a.Cfg.Key))
	switch m.MType {
	case gauge:
		data = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	case counter:
		data = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	}
	h.Write([]byte(data))
	m.Hash = hex.EncodeToString(h.Sum(nil))
}

func (a *Agent) sendData(m *Metric) error {
	var url string
	if a.Cfg.Key != "" {
		a.AddHash(m)
	}

	mSer, err := json.Marshal(*m)
	if err != nil {
		return err
	}
	url = fmt.Sprintf("http://%s/update/", a.Cfg.Address)

	if a.Encryptor != nil {
		mSer, err = a.Encryptor.encrypt(mSer)
		if err != nil {
			return err
		}
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(mSer))

	if err != nil {
		return err
	}

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return NewDecryptError(fmt.Sprintf("Non-OK HTTP status: %d", resp.StatusCode))
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}
	return nil
}

// GetDataByInterval gouroutine polls memory metrics each time it receives signal from syncChan.
func (a *Agent) GetDataByInterval(ctx context.Context, dataChan chan<- Data, syncChan <-chan time.Time) {
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
			log.Println("GetDataByInterval has been canceled successfully.")
			return
		}
	}
}

// GetMemDataByInterval gouroutine polls CPU data from system.
// Polls CPU metrics each time it receives signal from syncChan.
func (a *Agent) GetMemDataByInterval(ctx context.Context, gaugeChan chan<- Data, syncChan <-chan time.Time) {
	for {
		select {
		case <-syncChan:
			v, _ := mem.VirtualMemory()
			gaugeChan <- Data{name: "TotalMemory", gaugeValue: float64(v.Free)}
			gaugeChan <- Data{name: "FreeMemory", gaugeValue: float64(v.Free)}

		case <-ctx.Done():
			log.Println("GetMemDataByInterval has been canceled successfully.")
			return
		}
	}
}

// GetCPUDataByInterval gouroutine polls MEM data from system.
// Polls MEM metrics each time it receives signal from syncChan.
func (a *Agent) GetCPUDataByInterval(ctx context.Context, gaugeChan chan<- Data) {
	const CPUPollTime time.Duration = 10 * time.Second

	for {
		select {
		default:
			cSlice, _ := cpu.Percent(CPUPollTime, true)

			for i, c := range cSlice {
				gaugeChan <- Data{name: fmt.Sprintf("CPUutilization%d", i), gaugeValue: float64(c)}
			}

		case <-ctx.Done():
			log.Println("GetCPUDataByInterval has been canceled successfully.")
			return
		}
	}
}

func (a *Agent) sendBulkData(mList *[]Metric) error {
	url := fmt.Sprintf("http://%s/updates/", a.Cfg.Address)
	mSer, err := json.Marshal(*mList)
	if err != nil {
		return err
	}
	if a.Encryptor != nil {
		mSer, err = a.Encryptor.encrypt(mSer)
		if err != nil {
			return err
		}
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(mSer))

	if err != nil {
		log.Println(err)
		return err
	}

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return NewDecryptError(fmt.Sprintf("Non-OK HTTP status: %d", resp.StatusCode))
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}
	return nil
}

// SendDataByInterval gorouting sends data to server every specified interval.
func (a *Agent) SendDataByInterval(ctx context.Context, dataChan chan<- Data) {
	ticker := time.NewTicker(a.Cfg.ReportInterval)
	for {
		select {
		case <-ticker.C:
			var mList []Metric
			a.l.Lock()
			for _, m := range a.Metrics {
				err := a.sendData(&m)
				if err != nil {
					log.Printf("metric: %s, error: %s", m.ID, err)
				}
				mList = append(mList, m)
				if m.ID == "PollCount" {
					PollCount = 0
				}
			}
			a.l.Unlock()
			if PollCount == 0 {
				dataChan <- Data{name: "PollCount", counterValue: 0}
			}
			if len(mList) > 0 {
				err := a.sendBulkData(&mList)
				if err != nil {
					log.Print(err)
				}
			}
		case <-ctx.Done():
			log.Println("Context has been canceled successfully.")
			return
		}
	}
}

// RunTicker function syncronizes goroutines that poll system metrics by sending signal to syncChan.
// Goroutines that receive signal, poll system metrics with same interval.
func (a *Agent) RunTicker(ctx context.Context, syncChan chan<- time.Time) {
	ticker := time.NewTicker(a.Cfg.PollInterval)
	for {
		select {
		case t := <-ticker.C:
			syncChan <- t
		case <-ctx.Done():
			log.Println("RunTicker has been canceled successfully.")
		}
	}
}

// StopAgent stops the application.
func (a *Agent) StopAgent(sigChan <-chan os.Signal, cancel context.CancelFunc) {
	<-sigChan
	log.Println("Receieved a SIGINT! Stopping the agent.")

	cancel()
	log.Println("Stopped all goroutines.")

	os.Exit(1)
}

// NewMetric saves new incoming Data from channel to metric map in Metric format.
func (a *Agent) NewMetric(ctx context.Context, dataChan <-chan Data) {
	for {
		select {
		case data := <-dataChan:
			a.l.Lock()
			if data.name == "PollCount" {
				a.Metrics[data.name] = Metric{ID: data.name, MType: counter, Delta: &data.counterValue}
			} else {
				a.Metrics[data.name] = Metric{ID: data.name, MType: gauge, Value: &data.gaugeValue}
			}
			a.l.Unlock()
		case <-ctx.Done():
			log.Println("NewMetric has been canceled successfully.")
		}
	}
}
