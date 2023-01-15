package metric

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Jay-T/go-devops.git/internal/pb"
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

// GetValueInt returns pointer to int64 value.
func (m Metric) GetValueInt() int64 {
	return int64(*m.Delta)
}

// GetValueFloat returns pointer to float64 value.
func (m Metric) GetValueFloat() float64 {
	return *m.Value
}

// addHash computes hash for Metric fields for validation before sending it to server.
func (m *Metric) GenerateHash(key string) []byte {
	var data string

	h := hmac.New(sha256.New, []byte(key))
	switch m.MType {
	case gauge:
		data = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	case counter:
		data = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	}
	h.Write([]byte(data))
	return h.Sum(nil)
}

// PrepareMetric serializes Metric into JSON format.
// If key is passed - field Metric.hash is filled with hash generated with the key.
func (m *Metric) PrepareMetricAsJSON(key string) ([]byte, error) {
	if key != "" {
		hash := m.GenerateHash(key)
		m.Hash = hex.EncodeToString(hash)
	}

	mSer, err := json.Marshal(*m)
	if err != nil {
		return nil, err
	}
	return mSer, nil
}

// ConvertMetricToPB converts metric.Metric struct to pb.Metric struct
func (m *Metric) ConvertMetricToPB(key string) *pb.Metric {
	if key != "" {
		hash := m.GenerateHash(key)
		m.Hash = hex.EncodeToString(hash)
	}
	if m.MType == counter {
		return &pb.Metric{
			Id:    m.ID,
			Mtype: m.MType,
			Delta: m.Delta,
			Hash:  m.Hash,
		}
	}
	return &pb.Metric{
		Id:    m.ID,
		Mtype: m.MType,
		Value: m.Value,
		Hash:  m.Hash,
	}
}
