// services/product-service/internal/server/module.go

package server

import (
	"context"
	tlsutil "pkg/tls"

	"product-service/config"

	"go.uber.org/fx"
)

// RegisterHooks سرور gRPC را به Fx Lifecycle متصل می‌کند
func RegisterHooks(
	lc fx.Lifecycle,
	grpcServer *GRPCServer,
	cfg *config.Config,
) {
	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				return grpcServer.Start(cfg.GRPCPort)
			},
			OnStop: func(ctx context.Context) error {
				return grpcServer.Stop(ctx)
			},
		},
	)
}

var Module = fx.Module(
	"server",
	fx.Provide(
		NewServer,
		tlsutil.ServerCredentials,
	),
	fx.Invoke(
		RegisterHooks,
	),
)
