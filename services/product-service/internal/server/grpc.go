// services/product-service/internal/server/grpc.go

package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"pkg/auth"
	"pkg/grpcmiddleware"
	pb "pkg/proto/product"
	"pkg/ratelimit"
	redispkg "pkg/redis"

	"product-service/config"
	"product-service/internal/handler"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Module = fx.Module(
	"server",
	fx.Provide(
		NewTokenManager,
		NewRateLimiter,
		NewServer,
	),
	fx.Invoke(RegisterHooks),
)

// NewTokenManager ساخت TokenManager جهت اعتبارسنجی توکن‌ها در AuthInterceptor
func NewTokenManager(cfg *config.Config) auth.TokenManager {
	return auth.NewTokenManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL)
}

// NewRateLimiter ساخت RateLimiter برای RateLimitInterceptor
func NewRateLimiter(client *redispkg.Client) ratelimit.Limiter {
	return ratelimit.New(client)
}

// NewServer یک *grpc.Server کامل با زنجیره‌ی Interceptor می‌سازد و
// ProductService را رویش رجیستر می‌کند.
func NewServer(
	grpcHandler *handler.GRPCServer,
	tokens auth.TokenManager,
	limiter ratelimit.Limiter,
) *grpc.Server {

	chain := grpc.ChainUnaryInterceptor(
		grpcmiddleware.RecoveryInterceptor(),
		grpcmiddleware.LoggingInterceptor(),
		grpcmiddleware.RateLimitInterceptor(
			limiter,
			handler.RateLimitRules(),
			nil, // استفاده از IP پیش‌فرض
		),
		grpcmiddleware.AuthInterceptor(
			tokens,
			handler.PublicMethods(),
		),
	)

	grpcServer := grpc.NewServer(chain)

	pb.RegisterProductServiceServer(grpcServer, grpcHandler)

	// فعال‌سازی gRPC Reflection
	reflection.Register(grpcServer)

	return grpcServer
}

// RegisterHooks سرور را به چرخه‌ی حیات Fx متصل می‌کند.
func RegisterHooks(
	lc fx.Lifecycle,
	grpcServer *grpc.Server,
	cfg *config.Config,
) {

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {

			lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
			if err != nil {
				return fmt.Errorf(
					"failed to listen on port %s: %w",
					cfg.GRPCPort,
					err,
				)
			}

			go func() {
				log.Printf(
					"product-service: gRPC server listening on :%s",
					cfg.GRPCPort,
				)

				if err := grpcServer.Serve(lis); err != nil {
					log.Printf("product-service: grpc server stopped: %v", err)
				}
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {
			grpcServer.GracefulStop()
			return nil
		},
	})
}
