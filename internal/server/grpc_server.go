package server

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"log"
	"net"

	pb "github.com/Jay-T/go-devops.git/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	*GenericService
	pb.UnimplementedMetricsAgentServer
}

func NewGRPCServer(ctx context.Context, cfg *Config, backuper StorageBackuper) (*GRPCServer, error) {
	genericService, err := NewService(ctx, cfg, backuper)
	if err != nil {
		return nil, err
	}

	return &GRPCServer{
		genericService,
		pb.UnimplementedMetricsAgentServer{},
	}, nil
}

// StartServer launches HTTP server.
func (s GRPCServer) StartServer(ctx context.Context, backuper StorageBackuper) {
	listen, err := net.Listen("tcp", s.Cfg.Address)
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()
	pb.RegisterMetricsAgentServer(server, s)
	reflection.Register(server)

	go func() {
		log.Printf("Starting GRPC server on socket %s", s.Cfg.Address)
		if err := server.Serve(listen); err != nil {
			log.Fatal(err)
		}
	}()

	// r := chi.NewRouter()
	// // middlewares
	// r.Use(s.trustedNetworkCheckHandler)
	// r.Use(gzipHandle)
	// if s.Decryptor != nil {
	// 	r.Use(s.decryptHandler)
	// }
	// r.Mount("/debug", middleware.Profiler())
	// // old methods
	// r.Post("/update/gauge/{metricName}/{metricValue}", s.SetMetricOldHandler(ctx, backuper))
	// r.Post("/update/counter/{metricName}/{metricValue}", s.SetMetricOldHandler(ctx, backuper))
	// r.Post("/update/*", NotImplemented)
	// r.Post("/update/{metricName}/", NotFound)
	// r.Get("/value/*", s.GetMetricOldHandler)
	// r.Get("/", s.GetAllMetricHandler)
	// // new methods
	// r.Post("/update/", s.SetMetricHandler(ctx, backuper))
	// r.Post("/updates/", s.SetMetricListHandler(ctx, backuper))
	// r.Post("/value/", s.GetMetricHandler)
	// r.Get("/ping", backuper.CheckStorageStatus)

	// srv := &http.Server{
	// 	Addr:    s.Cfg.Address,
	// 	Handler: r,
	// }

	// srv.SetKeepAlivesEnabled(false)
	// log.Printf("Listening socket: %s", s.Cfg.Address)
	// log.Fatal(srv.ListenAndServe())

	<-ctx.Done()
	log.Printf("Finished to serve gRPC requests")
}

func (s GRPCServer) convertData(m *pb.Metric) (*Metric, error) {
	if m.Mtype == counter {
		return &Metric{
			ID:    m.Id,
			MType: m.Mtype,
			Delta: &m.Delta,
			Hash:  m.Hash,
		}, nil
	}
	return &Metric{
		ID:    m.Id,
		MType: m.Mtype,
		Value: &m.Value,
		Hash:  m.Hash,
	}, nil
}

func (s GRPCServer) UpdateMetric(ctx context.Context, in *pb.UpdateMetricRequest) (*pb.UpdateMetricResponse, error) {
	log.Printf("UpdateMetric request: %s", in)
	m, err := s.convertData(in.Metric)
	if err != nil {
		return &pb.UpdateMetricResponse{
			Error: "Could not convert received data",
		}, nil
	}
	var remoteHash []byte

	if s.Cfg.Key != "" {
		localHash := s.GenerateHash(m)
		remoteHash, err = hex.DecodeString(m.Hash)
		if err != nil {
			return &pb.UpdateMetricResponse{
				Error: "Hash validation error",
			}, nil
		}
		if !hmac.Equal(localHash, remoteHash) {
			return &pb.UpdateMetricResponse{
				Error: "Hash validation error",
			}, nil
		}
	}
	s.saveMetric(ctx, m)

	// return &pb.UpdateMetricResponse{
	// 	Error: "DEBUG Received UpdateMetric!",
	// }, nil

	return &pb.UpdateMetricResponse{}, nil
}

func (s GRPCServer) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	log.Printf("UpdateMetric request: %s", in)
	mList := make([]Metric, 0, 43)
	for _, i := range in.Metrics {
		m, err := s.convertData(i)
		if err != nil {
			return &pb.UpdateMetricsResponse{
				Error: "Could not convert received data",
			}, nil
		}
		mList = append(mList, *m)
	}

	err := s.saveListToDB(ctx, &mList)
	if err != nil {
		return &pb.UpdateMetricsResponse{
			Error: "Could not save received data to storage",
		}, nil
	}

	return &pb.UpdateMetricsResponse{}, nil
	// return &pb.UpdateMetricsResponse{
	// 	Error: "DEBUG Received UpdateMetrics!",
	// }, nil
}
