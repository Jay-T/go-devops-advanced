// Application collects system metrics and sends to server.
package agent

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sync"
	"time"

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

// Data struct describes message format between goroutines
type Data struct {
	name         string
	gaugeValue   float64
	counterValue int64
}

// Agent struct accepts Config and handles all metrics manipulations.
type GenericAgent struct {
	sync.RWMutex
	Cfg          *Config
	Metrics      map[string]Metric
	Encryptor    *Encryptor
	localAddress string
}

// NewAgent configures Agent and returns pointer on it.
func NewGenericAgent(cfg *Config) (*GenericAgent, error) {
	var a GenericAgent
	var err error

	a.Metrics = map[string]Metric{}
	a.Cfg = cfg

	if a.Cfg.CryptoKey != "" {
		a.Encryptor, err = NewEncryptor(a.Cfg.CryptoKey)
		if err != nil {
			return nil, err
		}
	}

	if a.Cfg.LocalInterface != "" {
		iface, err := net.InterfaceByName(cfg.LocalInterface)
		if err != nil {
			log.Fatal("Error while getting local interfaces.", err.Error())
		}

		if iface != nil {
			addresses, err := iface.Addrs()
			if err != nil {
				log.Fatal("Error while getting an address from local interface.", err.Error())
			}
			address := addresses[0]
			if ipnet, ok := address.(*net.IPNet); ok {
				a.localAddress = ipnet.IP.String()
			}
		}
	}

	return &a, nil
}

// AddHash computes hash for Metric fields for validation before sending it to server.
func (a *GenericAgent) AddHash(m *Metric) {
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

// GetDataByInterval gouroutine polls memory metrics each time it receives signal from syncChan.
func (a *GenericAgent) GetDataByInterval(ctx context.Context, dataChan chan<- Data, syncChan <-chan time.Time) {
	var rtm runtime.MemStats

	log.Printf("Polling data with interval: %s", a.Cfg.PollInterval)
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
func (a *GenericAgent) GetMemDataByInterval(ctx context.Context, gaugeChan chan<- Data, syncChan <-chan time.Time) {
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
func (a *GenericAgent) GetCPUDataByInterval(ctx context.Context, gaugeChan chan<- Data) {
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

// RunTicker function syncronizes goroutines that poll system metrics by sending signal to syncChan.
// Goroutines that receive signal, poll system metrics with same interval.
func (a *GenericAgent) RunTicker(ctx context.Context, syncChan chan<- time.Time) {
	ticker := time.NewTicker(a.Cfg.PollInterval)
	for {
		select {
		case t := <-ticker.C:
			syncChan <- t
		case <-ctx.Done():
			log.Println("RunTicker has been canceled successfully.")
			return
		}
	}
}

// StopAgent stops the application.
func (a *GenericAgent) StopAgent(sigChan <-chan os.Signal, doneChan <-chan struct{}, cancel context.CancelFunc) {
	<-sigChan
	log.Println("Receieved a SIGINT! Stopping the agent.")
	cancel()

	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.Println("Stopped all goroutines.")
			os.Exit(1)

		case <-doneChan:
			log.Println("Stopped all goroutines gracefully.")
			os.Exit(1)
		}
	}
}

// NewMetric saves new incoming Data from channel to metric map in Metric format.
func (a *GenericAgent) NewMetric(ctx context.Context, dataChan <-chan Data) {
	assignValue := func(data Data) {
		a.Lock()
		defer a.Unlock()
		if data.name == "PollCount" {
			a.Metrics[data.name] = Metric{ID: data.name, MType: counter, Delta: &data.counterValue}
		} else {
			a.Metrics[data.name] = Metric{ID: data.name, MType: gauge, Value: &data.gaugeValue}
		}
	}

	for {
		select {
		case data := <-dataChan:
			assignValue(data)
		case <-ctx.Done():
			data := <-dataChan
			assignValue(data)
			log.Println("NewMetric has been canceled successfully.")
			return
		}
	}
}
