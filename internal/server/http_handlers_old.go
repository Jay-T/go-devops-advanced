package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Jay-T/go-devops.git/internal/utils/metric"
)

// SetMetricOldHandler - an old handler that receives metrics in URI.
// URI: "/update/gauge/{metricName}/{metricValue}".
// URI: "/update/gcounter/{metricName}/{metricValue}".
//
// Deprecated: use SetMetricHandler instead.
func (s HTTPServer) SetMetricOldHandler(ctx context.Context) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m metric.Metric

		mType := strings.Split(r.RequestURI, "/")[2]
		mName := strings.Split(r.RequestURI, "/")[3]
		mValue := strings.Split(r.RequestURI, "/")[4]

		switch mType {
		case gauge:
			val, err := strconv.ParseFloat(string(mValue), 64)
			if err != nil {
				http.Error(w, "parsing error. Bad request", http.StatusBadRequest)
				return
			}
			m = metric.Metric{
				ID:    mName,
				MType: mType,
				Value: &val,
			}
		case counter:
			val, err := strconv.ParseInt(string(mValue), 10, 64)
			if err != nil {
				http.Error(w, "parsing error. Bad request", http.StatusBadRequest)
				return
			}
			m = metric.Metric{
				ID:    mName,
				MType: mType,
				Delta: &val,
			}
		default:
			log.Printf("Metric type '%s' is not expected. Skipping.", mType)
		}
		w.WriteHeader(http.StatusOK)
		s.saveMetric(ctx, &m)
	})
}

// GetMetricOldHandler - an old handler that returns a plain text metric value.
// URI: "/value/".
//
// Deprecated: use GetMetricHandler instead.
func (s HTTPServer) GetMetricOldHandler(w http.ResponseWriter, r *http.Request) {
	var returnValue float64
	splitURL := strings.Split(r.URL.Path, "/")
	if len(splitURL) < 4 {
		http.Error(w, "There is no metric you requested", http.StatusNotFound)
		return
	}
	metricName := splitURL[3]
	val, found := s.Metrics[metricName]
	if !found {
		http.Error(w, "There is no metric you requested", http.StatusNotFound)
		return
	}
	if val.MType == counter {
		returnValue = float64(*val.Delta)
	} else {
		returnValue = float64(*val.Value)
	}
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(fmt.Sprint(returnValue)))
	if err != nil {
		log.Print(err)
	}
}
