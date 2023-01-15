package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/Jay-T/go-devops.git/internal/utils/converter"
	"github.com/Jay-T/go-devops.git/internal/utils/metric"
)

//go:embed metrics.html
var htmlPage []byte

// GetAllMetricHandler returns HTML page with all metrics values.
// URI: "/".
func (s HTTPServer) GetAllMetricHandler(w http.ResponseWriter, r *http.Request) {
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
func (s HTTPServer) SetMetricHandler(ctx context.Context) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m, err := converter.GetBody(r)
		var remoteHash []byte

		if s.Cfg.Key != "" {
			localHash := m.GenerateHash(s.Cfg.Key)
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
		s.saveMetric(ctx, m)
		w.WriteHeader(http.StatusOK)
		err = r.Body.Close()
		if err != nil {
			log.Print(err)
		}
	})
}

// SetMetricListHandler saves a list of metrics from HTTP POST request.
// URI: "/updates/".
func (s HTTPServer) SetMetricListHandler(ctx context.Context) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)

		if err != nil {
			log.Println(err)
		}
		m := make([]metric.Metric, 0, 43)
		err = json.Unmarshal(body, &m)
		if err != nil {
			http.Error(w, "Internal error during JSON parsing", http.StatusInternalServerError)
			return
		}
		err = s.saveListToDB(ctx, &m)
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
func (s HTTPServer) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	m, err := converter.GetBody(r)
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
	res, err := data.PrepareMetricAsJSON(s.Cfg.Key)
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

func (s *HTTPServer) decryptHandler(next http.Handler) http.Handler {
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

func (s *HTTPServer) trustedNetworkCheckHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqXRealIP := r.Header.Get("X-Real-Ip")
		if reqXRealIP == "" {
			http.Error(w, "Request does not have X-Real-Ip header.", http.StatusForbidden)
			return
		}

		ip := net.ParseIP(reqXRealIP)

		if !s.trustedSubnet.Contains(ip) {
			http.Error(w, fmt.Sprintf("Access is forbidden for %s", ip), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *HTTPServer) CheckStorageStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := s.backuper.CheckStorageStatus(ctx); err != nil {
		http.Error(w, "Storage is inaccesible.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
