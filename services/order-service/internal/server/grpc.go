// services/oder-service/internal/server/grpc.go

package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"pkg/auth"
	"pkg/grpcmiddleware"
	pb "pkg/proto/order"
	"pkg/ratelimit"

	"order-service/internal/handler"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer مسئول مدیریت gRPC Server مربوط به Order Service است.
//
// این struct علاوه بر خود grpc.Server، listener را نیز نگه می‌دارد
// تا lifecycle هر دو resource به‌صورت مشخص مدیریت شود.
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
}

// NewServer یک gRPC Server کامل برای Order Service می‌سازد.
//
// مسئولیت‌های این تابع:
//
//   - ساخت interceptor chain
//   - ساخت grpc.Server
//   - ثبت OrderService
//   - فعال کردن gRPC Reflection
//
// شروع و متوقف کردن Server در RegisterHooks انجام می‌شود.
func NewServer(
	grpcHandler *handler.GRPCServer,
	tokens auth.TokenManager,
	limiter ratelimit.Limiter,
) *GRPCServer {

	chain := grpc.ChainUnaryInterceptor(

		// Recovery باید بیرونی‌ترین interceptor باشد
		// تا panicهای لایه‌های داخلی را نیز دریافت کند.
		grpcmiddleware.RecoveryInterceptor(),

		// نتیجه نهایی درخواست را log می‌کند؛
		// بنابراین درخواست‌هایی که توسط RateLimit یا Auth
		// رد شده‌اند نیز log خواهند شد.
		grpcmiddleware.LoggingInterceptor(),

		// Rate Limit قبل از Auth اجرا می‌شود.
		//
		// در وضعیت فعلی پروژه، nil یعنی استفاده از
		// DefaultKeyFunc که بر اساس IP کار می‌کند.
		grpcmiddleware.RateLimitInterceptor(
			limiter,
			handler.RateLimitRules(),
			nil, // nil یعنی از DefaultKeyFunc (بر اساس IP) استفاده شود
		),

		// Authentication آخرین interceptor قبل از رسیدن
		// درخواست به Handler است.
		grpcmiddleware.AuthInterceptor(
			tokens,
			handler.PublicMethods(),
		),
	)

	grpcServer := grpc.NewServer(chain)

	// ثبت RPCهای مربوط به Order Service.
	pb.RegisterOrderServiceServer(
		grpcServer,
		grpcHandler,
	)

	// فعال‌سازی gRPC Reflection
	reflection.Register(grpcServer)

	// فعال کردن gRPC Reflection.
	reflection.Register(grpcServer)

	return &GRPCServer{
		server: grpcServer,
	}
}

// Start سرور gRPC را روی پورت مشخص‌شده اجرا می‌کند.
//
// net.Listen قبل از اجرای goroutine انجام می‌شود تا اگر پورت
// قابل استفاده نبود، خطا مستقیماً به Fx برگردد.
//
// Serve بلاک‌کننده است؛ بنابراین داخل goroutine اجرا می‌شود.
func (s *GRPCServer) Start(
	port string,
) error {

	listener, err := net.Listen(
		"tcp",
		fmt.Sprintf(":%s", port),
	)
	if err != nil {
		return fmt.Errorf(
			"failed to listen on port %s: %w",
			port,
			err,
		)
	}

	s.listener = listener

	go func() {

		log.Printf(
			"order-service: gRPC server listening on :%s",
			port,
		)

		if err := s.server.Serve(listener); err != nil {

			// زمانی که Stop/GracefulStop اجرا شود،
			// Serve نیز متوقف می‌شود. بنابراین این log
			// لزوماً به معنی خطای واقعی نیست.
			log.Printf(
				"order-service: gRPC server stopped: %v",
				err,
			)
		}
	}()

	return nil
}

// Stop سرور gRPC را به‌صورت graceful متوقف می‌کند.
//
// اگر graceful shutdown در مدت ctx تمام نشود، Server به‌صورت
// اجباری Stop می‌شود تا shutdown کل application گیر نکند.
func (s *GRPCServer) Stop(ctx context.Context) error {

	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {

	case <-done:
		return nil

	case <-ctx.Done():

		// درخواست‌های باقی‌مانده را بدون انتظار بیشتر قطع می‌کنیم.
		s.server.Stop()

		return ctx.Err()
	}
}
