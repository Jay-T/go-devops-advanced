package agent

import (
	"context"

	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// clientInterceptor adds Request-ID and X-Real-Ip (if needed) to request metadata
func (a *GRPCAgent) clientInterceptor(ctx context.Context, method string, req interface{},
	reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption) error {

	if a.localAddress != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "X-Real-Ip", a.localAddress)
	}

	reqID := xid.New()
	ctx = metadata.AppendToOutgoingContext(ctx, "Request-ID", reqID.String())

	err := invoker(ctx, method, req, reply, cc, opts...)

	return err
}
