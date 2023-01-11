package server

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"fmt"
	"log"
	"net"

	pb "github.com/Jay-T/go-devops.git/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

// GRPCServer struct describes GRPC server based on GenericService.
type GRPCServer struct {
	*GenericService
	pb.UnimplementedMetricsAgentServer
}

// NewGRPCServer returns new GRPCServer.
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

// StartServer launches GRPC server.
func (s *GRPCServer) StartServer(ctx context.Context, backuper StorageBackuper) {
	listen, err := net.Listen("tcp", s.Cfg.Address)
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(s.checkReqIDInterceptor, s.checkIPInterceptor))
	pb.RegisterMetricsAgentServer(server, s)
	reflection.Register(server)

	go func() {
		log.Printf("Starting GRPC server on socket %s", s.Cfg.Address)
		if err := server.Serve(listen); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Printf("Finished to serve gRPC requests")
}

func (s *GRPCServer) convertData(m *pb.Metric) (*Metric, error) {
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

// UpdateMetric receives a Metric from client and updates it in storage.
func (s *GRPCServer) UpdateMetric(ctx context.Context, in *pb.UpdateMetricRequest) (*pb.UpdateMetricResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	reqID := md.Get("Request-ID")[0]

	m, err := s.convertData(in.Metric)
	if err != nil {
		return &pb.UpdateMetricResponse{
			Error: fmt.Sprintf("Could not convert received data. Req-id: %s", reqID),
		}, nil
	}
	var remoteHash []byte

	if s.Cfg.Key != "" {
		localHash := s.GenerateHash(m)
		remoteHash, err = hex.DecodeString(m.Hash)
		if err != nil {
			return &pb.UpdateMetricResponse{
				Error: fmt.Sprintf("Hash validation error. Req-id: %s", reqID),
			}, nil
		}
		if !hmac.Equal(localHash, remoteHash) {
			return &pb.UpdateMetricResponse{
				Error: fmt.Sprintf("Hash validation error. Req-id: %s", reqID),
			}, nil
		}
	}
	s.saveMetric(ctx, m)

	return &pb.UpdateMetricResponse{}, nil
}

// UpdateMetrics receives a slice of Metric from client and updates these metrics in storage.
func (s *GRPCServer) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	reqID := md.Get("Request-ID")[0]

	mList := make([]Metric, 0, 43)
	for _, i := range in.Metrics {
		m, err := s.convertData(i)
		if err != nil {
			return &pb.UpdateMetricsResponse{
				Error: fmt.Sprintf("Could not convert received data. Req-id: %s", reqID),
			}, nil
		}
		mList = append(mList, *m)
	}

	err := s.saveListToDB(ctx, &mList)
	if err != nil {
		return &pb.UpdateMetricsResponse{
			Error: fmt.Sprintf("Could not save received data to storage. Req-id: %s", reqID),
		}, nil
	}

	return &pb.UpdateMetricsResponse{}, nil
}
