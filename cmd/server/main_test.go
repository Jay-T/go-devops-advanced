package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MetricNew struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *float64 `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

func getFloatPointer(val float64) *float64 {
	return &val
}

func getIntPointer(val int64) *int64 {
	return &val
}

func TestSetMetricOldHandler(t *testing.T) {
	tests := []struct {
		name       string
		requestURL string
		wantCode   int
	}{
		{
			name:       "test one",
			requestURL: "/update/counter/TestMetric/28",
			wantCode:   200,
		},
		{
			name:       "test two",
			requestURL: "/update/counter/TestMetric2/aaa",
			wantCode:   400,
		},
		{
			name:       "test three",
			requestURL: "/update/gauge/TestMetric3/6464.5",
			wantCode:   200,
		},
		{
			name:       "test four",
			requestURL: "/update/gauge/TestMetric4/aaa",
			wantCode:   400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				cfg: Config{
					Address:       "localhost:8080",
					StoreInterval: 10,
					StoreFile:     "file.json",
					Restore:       true,
				},
			}
			request := httptest.NewRequest(http.MethodPost, tt.requestURL, nil)
			w := httptest.NewRecorder()

			ctx := context.TODO()
			h := http.HandlerFunc(s.SetMetricOldHandler(ctx))

			h.ServeHTTP(w, request)
			res := w.Result()

			assert.Equal(t, tt.wantCode, res.StatusCode)
			defer res.Body.Close()
		})
	}
}

func TestGetBody(t *testing.T) {
	tests := []struct {
		name    string
		metric  Metric
		want    string
		want2   string
		wantErr bool
	}{
		{
			name: "One",
			metric: Metric{
				ID:    "Alloc",
				MType: gauge,
				Value: getFloatPointer(1.5),
			},
			want:    "Alloc",
			want2:   "1.5",
			wantErr: false,
		},
		{
			name: "Two",
			metric: Metric{
				ID:    "PollCount",
				MType: counter,
				Delta: getIntPointer(4),
			},
			want:    "PollCount",
			want2:   "4",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mSer, _ := json.Marshal(tt.metric)
			request := httptest.NewRequest(http.MethodGet, "http://yandex.ru", bytes.NewBuffer(mSer))

			got, _ := GetBody(request)
			request.Body.Close()
			assert.Equal(t, tt.metric, *got)
		})
	}
}

func TestSetMetricHandler(t *testing.T) {
	tests := []struct {
		name     string
		metric   MetricNew
		wantCode int
	}{
		{
			name: "Test One",
			metric: MetricNew{
				ID:    "Alloc",
				MType: gauge,
				Value: getFloatPointer(4),
			},
			wantCode: 200,
		},
		{
			name: "Test Two",
			metric: MetricNew{
				ID:    "PollCount",
				MType: counter,
				Delta: getFloatPointer(4.5),
			},
			wantCode: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				cfg: Config{
					Address:       "localhost:8080",
					StoreInterval: 10,
					StoreFile:     "file.json",
					Restore:       true,
				},
			}

			mSer, _ := json.Marshal(tt.metric)
			request := httptest.NewRequest(http.MethodPost, "http://localhost", bytes.NewBuffer(mSer))
			w := httptest.NewRecorder()

			ctx := context.TODO()

			h := http.HandlerFunc(s.SetMetricHandler(ctx))

			h.ServeHTTP(w, request)
			res := w.Result()

			assert.Equal(t, tt.wantCode, res.StatusCode)
			defer res.Body.Close()
		})
	}
}
