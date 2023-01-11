package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type MetricNew struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
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
			s := HTTPServer{
				&GenericService{
					Cfg: &Config{
						Address:       "localhost:8080",
						StoreInterval: 10,
						StoreFile:     "file.json",
						Restore:       true,
					},
					Metrics: map[string]Metric{},
					backuper: &FileStorageBackuper{
						filename: "/tmp/test",
					},
				},
			}
			request := httptest.NewRequest(http.MethodPost, tt.requestURL, nil)
			w := httptest.NewRecorder()

			ctx := context.TODO()
			h := http.HandlerFunc(s.SetMetricOldHandler(ctx))

			h.ServeHTTP(w, request)
			res := w.Result()

			assert.Equal(t, tt.wantCode, res.StatusCode)
			err := res.Body.Close()
			if err != nil {
				log.Println(err)
			}
		})
	}
}

func TestGetBody(t *testing.T) {
	s := HTTPServer{
		&GenericService{
			Cfg: &Config{
				Address:       "localhost:8080",
				StoreInterval: 10,
				StoreFile:     "file.json",
				Restore:       true,
			},
			Metrics: map[string]Metric{},
		},
	}
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

			got, _ := s.GetBody(request)
			err := request.Body.Close()
			if err != nil {
				log.Println(err)
			}
			assert.Equal(t, tt.metric, *got)
		})
	}
}

func TestStartServer(t *testing.T) {
	fs := &FileStorageBackuper{
		filename: "/tmp/test",
	}
	s := HTTPServer{
		&GenericService{
			Metrics: map[string]Metric{},
			Cfg: &Config{
				Address: "localhost:8080",
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go s.StartServer(ctx, fs)

	time.Sleep(1 * time.Second)
	url := fmt.Sprintf("http://%s", s.Cfg.Address)
	resp, err := http.Get(url)
	if err != nil {
		assert.NoError(t, err, "Server did not start.")
	}
	cancel()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	err = resp.Body.Close()
	if err != nil {
		log.Println(err)
	}

}

func TestNewService(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	tests := []struct {
		name     string
		backuper StorageBackuper
		cfg      *Config
		wantErr  bool
	}{
		{
			name: "TestOne",
			backuper: &FileStorageBackuper{
				filename: "/tmp/test",
			},
			cfg: &Config{
				Restore: true,
			},
			wantErr: false,
		},
		{
			name: "TestTwo",
			backuper: &DBStorageBackuper{
				db: db,
			},
			cfg: &Config{
				StoreInterval: time.Second * 1,
				StoreFile:     "/tmp/test",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()

			_, err := NewService(ctx, tt.cfg, tt.backuper)
			if tt.wantErr {
				assert.Error(t, err, "NewService did not return an error as expected.")
			} else {
				assert.NoError(t, err, "NewService returned unexpected error.")
			}
		})
	}
}

func BenchmarkSetMetricHandler(b *testing.B) {
	s := HTTPServer{
		&GenericService{
			Cfg: &Config{
				Address:       "localhost:8080",
				StoreInterval: 10,
				StoreFile:     "file.json",
				Restore:       true,
			},
			Metrics: map[string]Metric{},
			backuper: &FileStorageBackuper{
				filename: "/tmp/test",
			},
		},
	}
	// triesN := 1

	data := MetricNew{
		ID:    "Alloc",
		MType: gauge,
		Value: getFloatPointer(4),
	}

	mSer, _ := json.Marshal(data)

	w := httptest.NewRecorder()

	ctx := context.TODO()

	requestPost := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(mSer))
	h := http.HandlerFunc(s.SetMetricHandler(ctx))
	b.ResetTimer()
	b.Run("SetMetricHandler", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			h.ServeHTTP(w, requestPost)
		}
	})

	requestPost = httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewBuffer(mSer))
	h = http.HandlerFunc(s.SetMetricListHandler(ctx))
	b.ResetTimer()
	b.Run("SetMetricListHandler", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			h.ServeHTTP(w, requestPost)
		}
	})

	requestPost = httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/2", nil)
	h = http.HandlerFunc(s.SetMetricOldHandler(ctx))
	b.ResetTimer()
	b.Run("SetMetricOldHandlerGauge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			h.ServeHTTP(w, requestPost)
		}
	})

	requestPost = httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/2", nil)
	h = http.HandlerFunc(s.SetMetricOldHandler(ctx))
	b.ResetTimer()
	b.Run("SetMetricOldHandlerCounter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			h.ServeHTTP(w, requestPost)
		}
	})

}

func BenchmarkGetAllMetricHandler(b *testing.B) {
	s := HTTPServer{
		&GenericService{
			Cfg: &Config{
				Address:       "localhost:8080",
				StoreInterval: 10,
				StoreFile:     "file.json",
				Restore:       true,
			},
			Metrics: map[string]Metric{},
		},
	}

	w := httptest.NewRecorder()
	requestGet := httptest.NewRequest(http.MethodGet, "/", nil)
	h := http.HandlerFunc(s.GetAllMetricHandler)
	b.Run("GetAllMetricHandler", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			h.ServeHTTP(w, requestGet)
		}
	})
}

func TestStopServer(t *testing.T) {
	s := &GenericService{}
	fakeExit := func(i int) {}

	patch := monkey.Patch(os.Exit, fakeExit)
	defer patch.Unpatch()

	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	fs := &FileStorageBackuper{
		filename: "/tmp/test",
	}

	testFunc := func() int {
		s.StopServer(ctx, cancel, fs)
		return 1
	}

	assert.Equal(t, 1, testFunc())
}
