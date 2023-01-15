package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Jay-T/go-devops.git/internal/utils/metric"
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
			s := HTTPServer{
				&GenericService{
					Cfg: &Config{
						Address:       "localhost:8080",
						StoreInterval: 10,
						StoreFile:     "file.json",
						Restore:       true,
						Key:           "testkey",
					},
					Metrics: map[string]metric.Metric{},
					backuper: &FileStorageBackuper{
						filename: "/tmp/test",
					},
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
		metric   metric.Metric
		wantCode int
	}{
		{
			name: "Test One",
			metric: metric.Metric{
				ID:    "Alloc",
				MType: gauge,
				Value: getFloatPointer(354872),
				Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
			},
			wantCode: 200,
		},
		{
			name: "Test Two",
			metric: metric.Metric{
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
			s := HTTPServer{
				&GenericService{
					Cfg: &Config{
						Address:       "localhost:8080",
						StoreInterval: 10,
						StoreFile:     "file.json",
						Restore:       true,
						Key:           "testkey",
					},
					Metrics: map[string]metric.Metric{},
				},
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
		metric metric.Metric
		want   []byte
	}{
		{
			name: "Test One",
			metric: metric.Metric{
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
			s := HTTPServer{
				&GenericService{
					Cfg: &Config{
						Address:       "localhost:8080",
						StoreInterval: 10,
						StoreFile:     "file.json",
						Restore:       true,
						Key:           "testkey",
					},
					Metrics: map[string]metric.Metric{},
				},
			}
			res := tt.metric.GenerateHash(s.Cfg.Key)

			assert.Equal(t, tt.want, res)
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

	s := HTTPServer{
		&GenericService{
			Metrics: map[string]metric.Metric{},
			backuper: &DBStorageBackuper{
				db: db,
			},
		},
	}

	tests := []struct {
		name string
		ml   []metric.Metric
		m    metric.Metric
		want int
	}{
		{
			name: "Test One",
			ml: []metric.Metric{
				{
					ID:    "Alloc",
					MType: gauge,
					Value: getFloatPointer(354872),
					Hash:  "a2bc398d457f8e417dce8776440f230519f0ee5e2a0cf96130cc631272a9987b",
				},
			},
			m:    metric.Metric{},
			want: 200,
		},
		{
			name: "Test Two",
			ml:   []metric.Metric{},
			m: metric.Metric{
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

			h := http.HandlerFunc(s.SetMetricListHandler(ctx))

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

func TestCheckStorageStatusHandler(t *testing.T) {
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

	s := HTTPServer{
		&GenericService{
			Metrics: map[string]metric.Metric{},
			backuper: &DBStorageBackuper{
				db: db,
			},
		},
	}

	request := httptest.NewRequest(http.MethodPost, "/", nil)

	mock.ExpectPing()
	w := httptest.NewRecorder()
	h := http.HandlerFunc(s.CheckStorageStatusHandler)

	h.ServeHTTP(w, request)
	res := w.Result()
	assert.Equal(t, res.StatusCode, 200)
	err = res.Body.Close()
	if err != nil {
		log.Println(err)
	}

	mock.ExpectPing().WillReturnError(New("TestError"))
	w = httptest.NewRecorder()
	h = http.HandlerFunc(s.CheckStorageStatusHandler)

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
	s := HTTPServer{
		&GenericService{
			Metrics: map[string]metric.Metric{
				"PollCount": {
					ID:    "PollCount",
					MType: counter,
					Delta: NewDelta(2),
				},
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

func TestDecryptHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           []byte
		httpStatusCode int
	}{
		{
			name:           "Test One",
			body:           []byte{150, 175, 105, 166, 14, 50, 47, 198, 72, 24, 248, 111, 191, 191, 46, 169, 41, 70, 123, 188, 39, 139, 57, 2, 35, 98, 58, 68, 55, 72, 191, 114, 227, 224, 221, 199, 213, 179, 56, 240, 246, 54, 27, 176, 15, 135, 61, 171, 49, 104, 156, 40, 101, 174, 193, 58, 14, 16, 196, 173, 9, 55, 209, 129, 37, 206, 248, 171, 75, 33, 39, 157, 152, 12, 48, 94, 160, 202, 194, 32, 65, 96, 255, 245, 190, 29, 98, 91, 53, 255, 254, 218, 96, 47, 173, 72, 79, 56, 61, 156, 130, 166, 101, 45, 10, 128, 254, 12, 183, 156, 157, 27, 98, 146, 19, 156, 34, 109, 54, 70, 206, 250, 65, 3, 32, 178, 239, 163, 73, 80, 13, 105, 173, 73, 56, 238, 94, 84, 43, 193, 71, 10, 75, 51, 92, 18, 180, 88, 178, 82, 115, 77, 207, 111, 255, 34, 213, 226, 87, 24, 238, 2, 215, 125, 24, 35, 29, 179, 187, 3, 181, 234, 86, 130, 26, 172, 6, 155, 141, 145, 224, 82, 15, 165, 226, 226, 6, 114, 37, 97, 188, 165, 21, 149, 155, 12, 7, 183, 89, 103, 78, 184, 241, 184, 53, 134, 1, 44, 182, 220, 50, 6, 157, 79, 75, 51, 167, 96, 244, 252, 101, 62, 218, 125, 138, 65, 190, 167, 128, 52, 243, 249, 158, 45, 70, 101, 147, 86, 0, 96, 96, 182, 247, 42, 238, 69, 175, 88, 219, 152, 19, 137, 158, 39, 107, 102, 97, 246, 14, 28, 18, 115, 118, 116, 166, 132, 0, 81, 189, 226, 249, 133, 148, 155, 32, 213, 132, 37, 157, 12, 214, 251, 132, 62, 210, 223, 1, 123, 219, 79, 65, 208, 147, 150, 240, 20, 197, 159, 240, 98, 0, 101, 134, 58, 175, 164, 78, 248, 67, 135, 218, 27, 55, 214, 112, 109, 133, 9, 157, 243, 175, 3, 103, 141, 17, 194, 126, 17, 26, 41, 55, 70, 140, 113, 126, 188, 161, 21, 57, 248, 150, 233, 37, 118, 90, 19, 194, 56, 180, 248, 125, 90, 133, 180, 14, 22, 138, 209, 104, 50, 137, 21, 210, 121, 31, 14, 4, 126, 203, 126, 36, 242, 152, 128, 114, 81, 192, 138, 193, 178, 26, 47, 3, 119, 120, 206, 75, 114, 12, 216, 112, 87, 150, 134, 111, 234, 46, 185, 215, 197, 110, 28, 100, 224, 230, 37, 130, 57, 159, 255, 181, 173, 32, 104, 229, 102, 246, 46, 213, 98, 36, 142, 159, 169, 36, 9, 224, 54, 168, 150, 177, 136, 74, 42, 5, 106, 0, 213, 147, 209, 163, 206, 154, 9, 165, 104, 219, 136, 61, 72, 13, 84, 12, 54, 214, 169, 246, 230, 156, 72, 90, 225, 221, 133, 236, 11, 48, 197, 203, 191, 71, 220, 97, 232, 248, 44, 33, 252, 141, 163, 190, 53, 42, 121, 132, 68, 248, 112, 86, 209, 249, 100, 196, 29, 164, 121, 189, 25, 3, 143, 253, 43, 90, 251, 236, 179, 24, 230, 133, 164, 200, 250},
			httpStatusCode: 200,
		},
		{
			name:           "Test Two",
			body:           []byte{166, 14, 50},
			httpStatusCode: 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decryptor, err := NewDecryptor("testkey.priv")
			if err != nil {
				log.Fatal(err)
			}

			s := HTTPServer{
				&GenericService{
					Decryptor: decryptor,
				},
			}
			encrypted := bytes.NewReader(tt.body)
			request := httptest.NewRequest(http.MethodPost, "/", encrypted)

			w := httptest.NewRecorder()
			h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			h0 := http.Handler(s.decryptHandler(h1))
			h0.ServeHTTP(w, request)

			res := w.Result()
			assert.Equal(t, tt.httpStatusCode, res.StatusCode)

			err = res.Body.Close()
			if err != nil {
				log.Println(err)
			}
		})
	}
}

func TestTrustedNetworkCheckHandler(t *testing.T) {
	tests := []struct {
		name           string
		trustedSubnet  string
		requestAddress string
		expectedCode   int
	}{
		{
			name:           "Test One. Valid request.",
			trustedSubnet:  "127.0.0.0/8",
			requestAddress: "127.0.0.1",
			expectedCode:   200,
		},
		{
			name:           "Test Three. Forbidden address.",
			trustedSubnet:  "10.0.0.0/8",
			requestAddress: "127.0.0.1",
			expectedCode:   403,
		},
		{
			name:           "Test Three. Empty X-Real-Op header.",
			trustedSubnet:  "10.0.0.0/8",
			requestAddress: "",
			expectedCode:   403,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipV4Net, _ := net.ParseCIDR(tt.trustedSubnet)
			s := HTTPServer{
				&GenericService{
					Cfg: &Config{
						TrustedSubnet: tt.trustedSubnet,
					},
					trustedSubnet: ipV4Net,
				},
			}

			request := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.requestAddress != "" {
				request.Header.Add("X-Real-Ip", tt.requestAddress)
			}

			w := httptest.NewRecorder()
			h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			h0 := http.Handler(s.trustedNetworkCheckHandler(h1))

			h0.ServeHTTP(w, request)
			res := w.Result()

			assert.Equal(t, tt.expectedCode, res.StatusCode)

			err := res.Body.Close()
			if err != nil {
				log.Println(err)
			}
		})
	}
}
