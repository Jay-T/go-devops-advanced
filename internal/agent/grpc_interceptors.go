package agent

import (
	"context"

	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// getClientInterceptor returns an interceptor which adds Request-ID and X-Real-Ip (if needed) to request metadata
func getClientInterceptor(address string) func(ctx context.Context, method string, req interface{},
	reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption) error {

	return func(ctx context.Context, method string, req interface{},
		reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {
		reqID := xid.New()
		ctx = metadata.AppendToOutgoingContext(ctx, "Request-ID", reqID.String(), "X-Real-Ip", address)

		err := invoker(ctx, method, req, reply, cc, opts...)

		return err
	}
}
