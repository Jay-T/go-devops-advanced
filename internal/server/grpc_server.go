package server

import "context"

type GRPCServer struct {
	*GenericService
}

func NewGRPCServerctx(ctx context.Context, cfg *Config, backuper StorageBackuper) (*GRPCServer, error) {
	genericService, err := NewService(ctx, cfg, backuper)
	if err != nil {
		return nil, err
	}

	return &GRPCServer{
		genericService,
	}, nil
}
