package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *GRPCServer) checkIPInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.NotFound, "Not MD found when expected")
	}
	reqID := md.Get("Request-ID")[0]
	reqXRealIPList := md.Get("X-Real-Ip")
	if len(reqXRealIPList) == 0 {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("X-Real-Ip is not found in MD. Req-ID: %s", reqID))
	}

	reqXRealIP := reqXRealIPList[0]
	ip := net.ParseIP(reqXRealIP)

	if !s.trustedSubnet.Contains(ip) {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("X-Real-Ip is not trusted. Aborting request. Req-ID: %s", reqID))
	}

	return handler(ctx, req)
}

func (s *GRPCServer) checkReqIDInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.NotFound, "Not MD found when expected")
	}

	reqIDList := md.Get("Request-ID")
	if len(reqIDList) == 0 {
		return nil, status.Error(codes.NotFound, "Request-ID is not found in MD")
	}
	return handler(ctx, req)
}
