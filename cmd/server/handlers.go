package main

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

func GetAllMetricHandler(w http.ResponseWriter, r *http.Request) {
	var floatVal float64
	for key, val := range metrics {
		if val.MType == gauge {
			floatVal = *val.Value
		} else {
			floatVal = float64(*val.Delta)
		}
		dataMap[key] = floatVal
	}

	htmlPage, err := os.ReadFile("metrics.html") // TODO: Fix file path relation
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("").Parse(string(htmlPage)))
	tmpl.Execute(w, dataMap)
}

func (s Service) SetMetricHandler(ctx context.Context) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m, err := GetBody(r)

		if s.cfg.Key != "" {
			localHash := s.GenerateHash(m)
			remoteHash, err := hex.DecodeString(m.Hash)
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
		r.Body.Close()
	})
}

func (s Service) SetMetricListHandler(ctx context.Context) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.db == nil {
			http.Error(w, "You haven`t opened the database connection", http.StatusInternalServerError)
			return
		}
		body, err := io.ReadAll(r.Body)

		if err != nil {
			log.Println(err)
		}
		m := &[]Metric{}
		err = json.Unmarshal(body, m)
		if err != nil {
			http.Error(w, "Internal error during JSON parsing", http.StatusInternalServerError)
			return
		}
		s.saveListToDB(ctx, m)
		r.Body.Close()
	})
}

func (s Service) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	m, err := GetBody(r)
	r.Body.Close()
	if err != nil {
		http.Error(w, "Internal error during JSON unmarshal", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	data, found := metrics[m.ID]
	if !found {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if s.cfg.Key != "" {
		data.Hash = hex.EncodeToString(s.GenerateHash(&data))
	}

	res, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Internal error during JSON marshal", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func NotImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Uknown type", http.StatusNotImplemented)
}

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
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func (s Service) PingDBHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		log.Println(err)
		http.Error(w, "Error in DB connection.", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
