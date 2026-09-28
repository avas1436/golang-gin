// services/product-service/internal/server/grpc.go

package server

import (
	"context"
	stdErrors "errors"
	"fmt"
	"log"
	"net"

	"pkg/auth"
	"pkg/grpcmiddleware"
	pb "pkg/proto/product"
	"pkg/ratelimit"

	"product-service/internal/handler"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer مسئول مدیریت چرخه حیات gRPC Server در Product Service است
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
}

// NewServer یک *grpc.Server کامل با زنجیره‌ی Interceptor می‌سازد و
// ProductService را رویش رجیستر می‌کند.
func NewServer(
	grpcHandler *handler.GRPCServer,
	tokens auth.TokenManager,
	limiter ratelimit.Limiter,
) *GRPCServer {

	chain := grpc.ChainUnaryInterceptor(
		// ۱. Recovery برای مدیریت Panicهای ناگهانی
		grpcmiddleware.RecoveryInterceptor(),

		// ۲. ثبت لاگ درخواست‌های ورودی و خروجی
		grpcmiddleware.LoggingInterceptor(),

		// ۳. محدودکننده نرخ درخواست (Rate Limiting)
		grpcmiddleware.RateLimitInterceptor(
			limiter,
			handler.RateLimitRules(),
			nil,
		),

		// ۴. احراز هویت درخواست‌ها (Authentication)
		grpcmiddleware.AuthInterceptor(
			tokens,
			handler.PublicMethods(),
		),
	)

	grpcServer := grpc.NewServer(chain)

	// ثبت Product Service
	pb.RegisterProductServiceServer(
		grpcServer,
		grpcHandler,
	)

	// فعال‌سازی reflection برای دیباگ
	reflection.Register(grpcServer)

	return &GRPCServer{
		server: grpcServer,
	}
}

func (s *GRPCServer) Start(port string) error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("product-service: failed to listen on port %s: %w", port, err)
	}

	s.listener = listener

	go func() {
		log.Printf("product-service: gRPC server listening on :%s", port)

		if err := s.server.Serve(listener); err != nil && !stdErrors.Is(
			err,
			grpc.ErrServerStopped,
		) {
			log.Printf("product-service: gRPC server stopped unexpectedly: %v", err)
		}
	}()

	return nil
}

// Stop خاموش‌سازی امن (Graceful Shutdown) سرور
func (s *GRPCServer) Stop(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("product-service: gRPC server stopped gracefully")
		return nil
	case <-ctx.Done():
		log.Println(
			"product-service: gRPC server forced to stop due to context timeout",
		)
		s.server.Stop()
		return ctx.Err()
	}
}
