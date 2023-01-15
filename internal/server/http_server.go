package server

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// HTTPServer struct describes HTTP server based on GenericService.
type HTTPServer struct {
	*GenericService
}

// NewHTTPService returns new HTTPServer.
func NewHTTPService(ctx context.Context, cfg *Config, backuper StorageBackuper) (*HTTPServer, error) {
	genericService, err := NewService(ctx, cfg, backuper)
	if err != nil {
		return nil, err
	}

	return &HTTPServer{
		genericService,
	}, nil
}

// StartServer launches HTTP server.
func (s HTTPServer) StartServer(ctx context.Context, backuper StorageBackuper) {
	log.Println("Starting HTTP server")
	r := chi.NewRouter()
	// middlewares
	middlewares := []func(http.Handler) http.Handler{
		gzipHandle,
	}
	if s.Cfg.TrustedSubnet != "" {
		middlewares = append(middlewares, s.trustedNetworkCheckHandler)
	}
	if s.Decryptor != nil {
		middlewares = append(middlewares, s.decryptHandler)
	}

	r.Use(middlewares...)

	r.Mount("/debug", middleware.Profiler())
	// old methods
	r.Post("/update/gauge/{metricName}/{metricValue}", s.SetMetricOldHandler(ctx))
	r.Post("/update/counter/{metricName}/{metricValue}", s.SetMetricOldHandler(ctx))
	r.Post("/update/*", NotImplemented)
	r.Post("/update/{metricName}/", NotFound)
	r.Get("/value/*", s.GetMetricOldHandler)
	r.Get("/", s.GetAllMetricHandler)
	// new methods
	r.Post("/update/", s.SetMetricHandler(ctx))
	r.Post("/updates/", s.SetMetricListHandler(ctx))
	r.Post("/value/", s.GetMetricHandler)
	r.Get("/ping", backuper.CheckStorageStatus)

	srv := &http.Server{
		Addr:    s.Cfg.Address,
		Handler: r,
	}

	srv.SetKeepAlivesEnabled(false)
	log.Printf("Listening socket: %s", s.Cfg.Address)
	log.Fatal(srv.ListenAndServe())
}
