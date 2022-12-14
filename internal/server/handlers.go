package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"
)

//go:embed metrics.html
var htmlPage []byte

// GetAllMetricHandler returns HTML page with all metrics values.
// URI: "/".
func (s Service) GetAllMetricHandler(w http.ResponseWriter, r *http.Request) {
	var floatVal float64
	dataMap := map[string]float64{}

	for key, val := range s.Metrics {
		if val.MType == gauge {
			floatVal = *val.Value
		} else {
			floatVal = float64(*val.Delta)
		}
		dataMap[key] = floatVal
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("").Parse(string(htmlPage)))
	err := tmpl.Execute(w, dataMap)
	if err != nil {
		log.Print(err)
	}
}

// SetMetricHandler saves metric from HTTP POST request.
// URI: "/update/".
func (s Service) SetMetricHandler(ctx context.Context, backuper StorageBackuper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m, err := s.GetBody(r)
		var remoteHash []byte

		if s.Cfg.Key != "" {
			localHash := s.GenerateHash(m)
			remoteHash, err = hex.DecodeString(m.Hash)
			if err != nil {
				http.Error(w, "Hash validation error", http.StatusInternalServerError)
				return
			}
			if !hmac.Equal(localHash, remoteHash) {
				http.Error(w, "Hash validation error", http.StatusBadRequest)
				return
			}
		}

		if err != nil {
			http.Error(w, "Internal error during JSON parsing", http.StatusInternalServerError)
			return
		}
		s.saveMetric(ctx, backuper, m)
		w.WriteHeader(http.StatusOK)
		err = r.Body.Close()
		if err != nil {
			log.Print(err)
		}
	})
}

// SetMetricListHandler saves a list of metrics from HTTP POST request.
// URI: "/updates/".
func (s Service) SetMetricListHandler(ctx context.Context, backuper StorageBackuper) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)

		if err != nil {
			log.Println(err)
		}
		m := make([]Metric, 0, 43)
		err = json.Unmarshal(body, &m)
		if err != nil {
			http.Error(w, "Internal error during JSON parsing", http.StatusInternalServerError)
			return
		}
		err = s.saveListToDB(ctx, &m, backuper)
		if err != nil {
			log.Print(err)
		}
		err = r.Body.Close()
		if err != nil {
			log.Print(err)
		}
	})
}

// GetMetricHandler returns a metric which was specified in HTTP POST request.
// URI: "/value/".
func (s Service) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	m, err := s.GetBody(r)
	if err != nil {
		http.Error(w, "Internal error during JSON unmarshal", http.StatusInternalServerError)
		return
	}
	err = r.Body.Close()
	if err != nil {
		log.Print(err)
	}

	w.Header().Add("Content-Type", "application/json")
	data, found := s.Metrics[m.ID]
	if !found {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if s.Cfg.Key != "" {
		data.Hash = hex.EncodeToString(s.GenerateHash(&data))
	}

	res, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Internal error during JSON marshal", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(res)
	if err != nil {
		log.Print(err)
	}
}

// NotImplemented handler returns HTTP StatusNotImplemented (code: 501) .
func NotImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Uknown type", http.StatusNotImplemented)
}

// NotFound handler returns HTTP StatusNotFound (code: 404).
func NotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not Found", http.StatusNotFound)
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			_, err = io.WriteString(w, err.Error())
			if err != nil {
				log.Print(err)
			}
			return
		}
		defer func() {
			err = gz.Close()
			if err != nil {
				log.Print(err)
			}
		}()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func (s *Service) decryptHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Could not read request body.", http.StatusBadRequest)
			return
		}

		if len(body) > 0 {
			body, err = s.Decryptor.decrypt(body)
			if err != nil {
				http.Error(w, "Could not decrypt message.", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}
		next.ServeHTTP(w, r)
	})
}
