package server

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"fmt"
	"log"
	"net"

	pb "github.com/Jay-T/go-devops.git/internal/pb"
	"github.com/Jay-T/go-devops.git/internal/utils/converter"
	"github.com/Jay-T/go-devops.git/internal/utils/helpers"
	"github.com/Jay-T/go-devops.git/internal/utils/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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

	interceptors := []grpc.UnaryServerInterceptor{
		s.checkReqIDInterceptor,
	}

	if s.Cfg.TrustedSubnet != "" {
		interceptors = append(interceptors, s.checkIPInterceptor)
	}

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))
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

// UpdateMetric receives a Metric from client and updates it in storage.
func (s *GRPCServer) UpdateMetric(ctx context.Context, in *pb.UpdateMetricRequest) (*pb.UpdateMetricResponse, error) {
	reqID := helpers.GetReqID(ctx)

	m, err := converter.ConvertData(in.Metric)
	if err != nil {
		return &pb.UpdateMetricResponse{
			Error: fmt.Sprintf("Could not convert received data. Req-id: %s", reqID),
		}, nil
	}
	var remoteHash []byte

	if s.Cfg.Key != "" {
		localHash := m.GenerateHash(s.Cfg.Key)
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
	reqID := helpers.GetReqID(ctx)

	mList := make([]metric.Metric, 0, 43)
	for _, i := range in.Metrics {
		m, err := converter.ConvertData(i)
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

func (s *GRPCServer) GetMetric(ctx context.Context, in *pb.GetMetricRequest) (*pb.GetMetricResponse, error) {
	reqID := helpers.GetReqID(ctx)

	m, found := s.Metrics[in.Id]
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Unknown metric id: %s. Req-id: %s", in.Id, reqID))
	}

	mpb := m.ConvertMetricToPB(s.Cfg.Key)

	return &pb.GetMetricResponse{
		Metric: mpb,
	}, nil
}

func (s *GRPCServer) GetAllMetrics(ctx context.Context, in *emptypb.Empty) (*pb.GetAllMetricsResponse, error) {
	var mList []*pb.Metric

	for _, m := range s.Metrics {
		mpb := m.ConvertMetricToPB(s.Cfg.Key)
		mList = append(mList, mpb)
	}

	return &pb.GetAllMetricsResponse{
		Metrics: mList,
	}, nil
}

func (s *GRPCServer) CheckStorageStatus(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.backuper.CheckStorageStatus(ctx); err != nil {
		return nil, status.Error(codes.Internal, "storage is inaccesible.")
	}

	return &emptypb.Empty{}, nil
}
