// services/payment-service/internal/server

package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"pkg/auth"
	"pkg/grpcmiddleware"
	pb "pkg/proto/payment"
	"pkg/ratelimit"

	"payment-service/internal/handler"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer مسئول مدیریت چرخه حیات gRPC Server در Payment Service است
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
}

// NewServer سرور اصلی gRPC را همراه با Middelwareها، Interceptorها و ثبت RPCها می‌سازد
func NewServer(
	grpcHandler *handler.GRPCServer,
	tokens auth.TokenManager,
	limiter ratelimit.Limiter,
) *GRPCServer {

	chain := grpc.ChainUnaryInterceptor(
		// Recovery باید بیرونی‌ترین interceptor باشد تا panicهای داخلی را مدیریت کند
		grpcmiddleware.RecoveryInterceptor(),

		// ثبت لاگ نهایی درخواست‌ها
		grpcmiddleware.LoggingInterceptor(),

		// Rate Limit بر اساس IP قبل از Auth اجرا می‌شود
		grpcmiddleware.RateLimitInterceptor(
			limiter,
			handler.RateLimitRules(),
			nil, // استفاده از DefaultKeyFunc بر اساس IP
		),

		// Authentication آخرین interceptor قبل از رسیدن درخواست به Handler است
		grpcmiddleware.AuthInterceptor(
			tokens,
			handler.PublicMethods(),
		),
	)

	grpcServer := grpc.NewServer(chain)

	// ثبت Payment Service در gRPC Server
	pb.RegisterPaymentServiceServer(
		grpcServer,
		grpcHandler,
	)

	// فعال‌سازی gRPC Reflection برای دیباگ با gRPCurl و Postman
	reflection.Register(grpcServer)

	return &GRPCServer{
		server: grpcServer,
	}
}

// Start listener شبکه را تنظیم کرده و سرور gRPC را در یک goroutine مجزا اجرا می‌کند
func (s *GRPCServer) Start(port string) error {

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
			"payment-service: gRPC server listening on :%s",
			port,
		)

		if err := s.server.Serve(listener); err != nil {
			log.Printf(
				"payment-service: gRPC server stopped: %v",
				err,
			)
		}
	}()

	return nil
}

// Stop سرور را به‌صورت Graceful متوقف کرده و در صورت اتمام مهلت context، به اجبار خاموش می‌کند
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
		s.server.Stop()
		return ctx.Err()
	}
}
