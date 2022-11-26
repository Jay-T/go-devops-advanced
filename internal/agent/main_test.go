package agent

import (
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func handlers() http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}

func Test_SendData(t *testing.T) {
	type fields struct {
		name     string
		typename string
		value    float64
		delta    int64
	}
	tests := []struct {
		name    string
		fields  fields
		client  *http.Client
		want    bool
		wantErr bool
	}{
		{
			name:    "test one",
			fields:  fields{name: "Alloc", typename: gauge, value: 1.5},
			client:  &http.Client{Timeout: 2 * time.Second},
			wantErr: false,
		},
		{
			name:    "test two",
			fields:  fields{name: "PollCounter", typename: counter, delta: 1},
			client:  &http.Client{Timeout: 2 * time.Second},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Agent{
				Cfg: &Config{
					Address:        "localhost:8080",
					ReportInterval: 10,
					PollInterval:   2,
				},
			}

			m := Metric{
				ID:    tt.fields.name,
				MType: tt.fields.typename,
				Value: &tt.fields.value,
				Delta: &tt.fields.delta,
			}

			l, err := net.Listen("tcp", "127.0.0.1:8080")
			if err != nil {
				log.Fatal(err)
			}
			srv := httptest.NewUnstartedServer(handlers())
			err = srv.Listener.Close()
			if err != nil {
				log.Println(err)
			}
			srv.Listener = l
			srv.Start()

			defer srv.Close()

			err = a.sendData(&m)
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func Test_SendDataOld(t *testing.T) {
	type fields struct {
		name     string
		typename string
		value    float64
		delta    int64
	}
	tests := []struct {
		name    string
		fields  fields
		client  *http.Client
		want    bool
		wantErr bool
	}{
		{
			name:    "test one",
			fields:  fields{name: "Alloc", typename: gauge, value: 1.5},
			client:  &http.Client{Timeout: 2 * time.Second},
			wantErr: false,
		},
		{
			name:    "test two",
			fields:  fields{name: "PollCounter", typename: counter, delta: 1},
			client:  &http.Client{Timeout: 2 * time.Second},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Agent{
				Cfg: &Config{
					Address:        "localhost:8080",
					ReportInterval: 10,
					PollInterval:   2,
				},
			}

			m := Metric{
				ID:    tt.fields.name,
				MType: tt.fields.typename,
				Value: &tt.fields.value,
				Delta: &tt.fields.delta,
			}

			l, err := net.Listen("tcp", "127.0.0.1:8080")
			if err != nil {
				log.Fatal(err)
			}
			srv := httptest.NewUnstartedServer(handlers())
			err = srv.Listener.Close()
			if err != nil {
				log.Println(err)
			}
			srv.Listener = l
			srv.Start()

			defer srv.Close()

			err = a.sendDataOld(&m)
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
