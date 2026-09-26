// services/user-service/internal/server/grpc.go

package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"pkg/auth"
	"pkg/grpcmiddleware"
	pb "pkg/proto/user"
	"pkg/ratelimit"

	"user-service/internal/handler"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer مسئول مدیریت چرخه حیات gRPC Server در User Service است
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
}

// NewServer سرور اصلی gRPC را همراه با Middlewareها، Interceptorها و ثبت RPCها می‌سازد
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

		// Rate Limit بر اساس قوانین تعیین‌شده در routes.go
		grpcmiddleware.RateLimitInterceptor(
			limiter,
			handler.RateLimitRules(),
			nil, // استفاده از DefaultKeyFunc بر اساس IP
		),

		// Authentication قبل از رسیدن درخواست به Handler
		grpcmiddleware.AuthInterceptor(
			tokens,
			handler.PublicMethods(),
		),
	)

	grpcServer := grpc.NewServer(chain)

	// ثبت User Service در gRPC Server
	pb.RegisterUserServiceServer(
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
			"user-service: gRPC server listening on :%s",
			port,
		)

		if err := s.server.Serve(listener); err != nil {
			log.Printf(
				"user-service: gRPC server stopped: %v",
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
