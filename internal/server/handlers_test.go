package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

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
				Value: getFloatPointer(354872),
				Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
			},
			wantCode: 200,
		},
		{
			name: "Test Two",
			metric: MetricNew{
				ID:    "PollCount",
				MType: counter,
				Delta: getIntPointer(2),
				Hash:  "1f2edcacbea902b88106c8af86113ae66294d464dea6fe7635115e269ceac84b",
			},
			wantCode: 200,
		},
		{
			name: "Test Three",
			metric: MetricNew{
				ID:    "Alloc",
				MType: gauge,
				Value: getFloatPointer(355872),
				Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
			},
			wantCode: 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				Cfg: &Config{
					Address:       "localhost:8080",
					StoreInterval: 10,
					StoreFile:     "file.json",
					Restore:       true,
					Key:           "testkey",
				},
				Metrics: map[string]Metric{},
			}

			mSer, _ := json.Marshal(tt.metric)
			request := httptest.NewRequest(http.MethodPost, "http://localhost", bytes.NewBuffer(mSer))
			w := httptest.NewRecorder()

			ctx := context.TODO()

			fs := &FileStorageBackuper{
				filename: "/tmp/test",
			}
			h := http.HandlerFunc(s.SetMetricHandler(ctx, fs))

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

func TestGetMetricHandler(t *testing.T) {
	tests := []struct {
		name     string
		metric   Metric
		wantCode int
	}{
		{
			name: "Test One",
			metric: Metric{
				ID:    "Alloc",
				MType: gauge,
				Value: getFloatPointer(354872),
				Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
			},
			wantCode: 200,
		},
		{
			name: "Test Two",
			metric: Metric{
				ID:    "NoMetric",
				MType: gauge,
				Value: getFloatPointer(354872),
				Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
			},
			wantCode: 404,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				Cfg: &Config{
					Address:       "localhost:8080",
					StoreInterval: 10,
					StoreFile:     "file.json",
					Restore:       true,
					Key:           "testkey",
				},
				Metrics: map[string]Metric{},
			}

			s.Metrics["Alloc"] = tt.metric
			mSer, _ := json.Marshal(tt.metric)
			request := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(mSer))
			w := httptest.NewRecorder()

			h := http.HandlerFunc(s.GetMetricHandler)

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

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name   string
		metric Metric
		want   []byte
	}{
		{
			name: "Test One",
			metric: Metric{
				ID:    "Alloc",
				MType: gauge,
				Value: getFloatPointer(354872),
				Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
			},
			want: []byte{0xa2, 0xbc, 0x39, 0x8d, 0x45, 0x7f, 0x8e, 0x41, 0x7d, 0xce, 0x87, 0x76, 0x44, 0xf, 0x23, 0x5, 0x19, 0xf0, 0xee, 0x5e, 0x2a, 0xc, 0xf9, 0x61, 0x30, 0xcc, 0x63, 0x12, 0x72, 0xa9, 0x98, 0x7b},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				Cfg: &Config{
					Address:       "localhost:8080",
					StoreInterval: 10,
					StoreFile:     "file.json",
					Restore:       true,
					Key:           "testkey",
				},
				Metrics: map[string]Metric{},
			}
			res := s.GenerateHash(&tt.metric)

			assert.Equal(t, tt.want, res)
		})
	}
}

func TestGzipHandle(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{
			name: "Test One",
			want: true,
		},
		{
			name: "Test Two",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.want {
				request.Header.Add("Accept-Encoding", "gzip")
			}

			w := httptest.NewRecorder()
			h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			h0 := http.Handler(gzipHandle(h1))

			h0.ServeHTTP(w, request)
			assert.Equal(t, tt.want, strings.Contains(w.Header().Get("Content-Encoding"), "gzip"))
		})
	}
}

func TestSetMetricListHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		err = db.Close()
		if err != nil {
			log.Println(err)
		}
	}()
	ctx := context.TODO()

	s := Service{
		Metrics: map[string]Metric{},
	}

	dbs := &DBStorageBackuper{
		db: db,
	}

	tests := []struct {
		name string
		ml   []Metric
		m    Metric
		want int
	}{
		{
			name: "Test One",
			ml: []Metric{
				{
					ID:    "Alloc",
					MType: gauge,
					Value: getFloatPointer(354872),
					Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
				},
			},
			m:    Metric{},
			want: 200,
		},
		{
			name: "Test Two",
			ml:   []Metric{},
			m: Metric{
				ID:    "Alloc",
				MType: gauge,
				Value: getFloatPointer(354872),
				Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
			},
			want: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mSer []byte
			if len(tt.ml) > 0 {
				mSer, _ = json.Marshal(tt.ml)
			} else {
				mSer, _ = json.Marshal(tt.m)
			}
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(mSer))
			w := httptest.NewRecorder()

			mock.ExpectBegin()
			mock.ExpectPrepare(`INSERT INTO metrics`).ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			h := http.HandlerFunc(s.SetMetricListHandler(ctx, dbs))

			h.ServeHTTP(w, request)
			res := w.Result()
			err := res.Body.Close()
			if err != nil {
				log.Println(err)
			}
			assert.Equal(t, res.StatusCode, tt.want)

		})
	}
}

func TestCheckStorageStatus(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		err = db.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	dbs := &DBStorageBackuper{
		db: db,
	}

	request := httptest.NewRequest(http.MethodPost, "/", nil)

	mock.ExpectPing()
	w := httptest.NewRecorder()
	h := http.HandlerFunc(dbs.CheckStorageStatus)

	h.ServeHTTP(w, request)
	res := w.Result()
	assert.Equal(t, res.StatusCode, 200)
	err = res.Body.Close()
	if err != nil {
		log.Println(err)
	}

	mock.ExpectPing().WillReturnError(New("TestError"))
	w = httptest.NewRecorder()
	h = http.HandlerFunc(dbs.CheckStorageStatus)

	h.ServeHTTP(w, request)
	res = w.Result()
	err = res.Body.Close()
	if err != nil {
		log.Println(err)
	}
	assert.Equal(t, res.StatusCode, 500)
}

func NewDelta(n int64) *int64 {
	return &n
}

func TestGetMetricOldHandler(t *testing.T) {
	s := Service{
		Metrics: map[string]Metric{
			"PollCount": {
				ID:    "PollCount",
				MType: counter,
				Delta: NewDelta(2),
			},
		},
	}

	tests := []struct {
		name      string
		URI       string
		wantValue string
		wantCode  int
	}{
		{
			name:      "TestOne",
			URI:       "/value/counter/PollCount",
			wantValue: "2",
			wantCode:  http.StatusOK,
		},
		{
			name:      "TestTwo",
			URI:       "/value/PollCount",
			wantValue: "There is no metric you requested\n",
			wantCode:  http.StatusNotFound,
		},
		{
			name:      "TestThree",
			URI:       "/value/counter/wrongpath/PollCount",
			wantValue: "There is no metric you requested\n",
			wantCode:  http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.URI, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.GetMetricOldHandler)

			h.ServeHTTP(w, request)
			resp := w.Result()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			value := string(bodyBytes)

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, tt.wantValue, value, "Function returned unexpected metric value")

			err = resp.Body.Close()
			if err != nil {
				log.Println(err)
			}
		})
	}
}
