package converter

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Jay-T/go-devops.git/internal/pb"
	"github.com/Jay-T/go-devops.git/internal/utils/metric"
)

const (
	gauge   = "gauge"
	counter = "counter"
)

// ConvertData converts pb.Metric struct into metric.Metric struct.
func ConvertData(pbm *pb.Metric) (*metric.Metric, error) {
	if pbm.Mtype == counter {
		return &metric.Metric{
			ID:    pbm.Id,
			MType: pbm.Mtype,
			Delta: pbm.Delta,
			Hash:  pbm.Hash,
		}, nil
	}
	return &metric.Metric{
		ID:    pbm.Id,
		MType: pbm.Mtype,
		Value: pbm.Value,
		Hash:  pbm.Hash,
	}, nil
}

// GetBody parses HTTP request's body and returns Metric.
func GetBody(r *http.Request) (*metric.Metric, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	m := &metric.Metric{}
	err = json.Unmarshal(body, m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
