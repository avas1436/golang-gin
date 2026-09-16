// services/order-service/internal/server/module.go

package server

import (
	"context"

	"order-service/config"

	"go.uber.org/fx"
)

// RegisterHooks سرور gRPC را به lifecycle مربوط به Fx متصل می‌کند.
//
// OnStart:
//
//   - Server را روی پورت مشخص‌شده اجرا می‌کند.
//   - Start به دلیل اجرای Serve در goroutine، سریع برمی‌گردد.
//
// OnStop:
//
//   - ابتدا GracefulStop را امتحان می‌کند.
//   - اگر context مربوط به shutdown تمام شود، Server را
//     به‌صورت اجباری Stop می‌کند.
func RegisterHooks(
	lc fx.Lifecycle,
	grpcServer *GRPCServer,
	cfg *config.Config,
) {

	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {

				return grpcServer.Start(
					cfg.GRPCPort,
				)
			},

			OnStop: func(ctx context.Context) error {

				return grpcServer.Stop(ctx)
			},
		},
	)
}

// Module مربوط به gRPC Server در Order Service است.
//
// Dependencyهای مورد نیاز NewServer:
//
//   - *handler.GRPCServer
//   - auth.TokenManager
//   - ratelimit.Limiter
//
// این dependencyها توسط Moduleهای مربوط به خودشان
// در application graph ساخته می‌شوند.
var Module = fx.Module(
	"server",

	fx.Provide(
		NewServer,
	),

	fx.Invoke(
		RegisterHooks,
	),
)
