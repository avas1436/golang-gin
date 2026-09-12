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

	"product-service/config"
	"product-service/internal/handler"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Module = fx.Module(
	"server",
	fx.Provide(
		NewServer,
	),
	fx.Invoke(RegisterHooks),
)

// NewServer یک *grpc.Server کامل با زنجیره‌ی Interceptor می‌سازد و
// ProductService را رویش رجیستر می‌کند.
//
// ترتیب زنجیره عمداً این‌طور است:
//  1. Recovery — باید بیرونی‌ترین لایه باشد تا پنیک هرکدام از
//     لایه‌های داخلی‌تر (حتی خودِ Logging) را هم بگیرد
//  2. Logging — نتیجه‌ی نهایی درخواست (شامل رد شدن توسط Auth یا
//     RateLimit) را ثبت می‌کند
//  3. RateLimit — قبل از Auth اجرا می‌شود چون بر اساس IP است و
//     نیازی به parse کردن JWT ندارد؛ ترافیک مخرب/سیل‌آسا را قبل از
//     صرف هزینه‌ی اعتبارسنجی توکن متوقف می‌کند
//  4. Auth — آخرین لایه، دقیقاً قبل از رسیدن به منطق واقعی handler
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
			nil, // nil یعنی از DefaultKeyFunc (بر اساس IP) استفاده شود
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
//
// OnStart باید سریع برگردد (Fx منتظرش می‌ماند)، در حالی که
// grpcServer.Serve بلاک‌کننده است؛ به همین دلیل داخل یک goroutine
// اجرا می‌شود. OnStop با GracefulStop به درخواست‌های در حال پردازش
// فرصت می‌دهد قبل از بسته‌شدن واقعی کانکشن‌ها تمام شوند (برخلاف
// Stop که بی‌رحمانه قطع می‌کند)
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
