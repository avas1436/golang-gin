// services/user-service/cmd/main.go

package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"pkg/auth"
	"pkg/env"
	"pkg/grpcmiddleware"
	"pkg/postgres"
	pb "pkg/proto/user"
	"pkg/ratelimit"
	redispkg "pkg/redis"
	"user-service/config"
	"user-service/internal/handler"
	"user-service/internal/repository"
	"user-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	// لود کردن فایل متغیر های محیطی در محیط پردازش برنامه
	env.Load(".env")

	// یک کانتکست پایه برای شروع پروژه
	ctx := context.Background()

	// بارگذاری اولیه کانفیگ ها و استفاده از آنها در کل پروژه با cfg
	cfg, err := config.Load()

	// اگر کانفیگ ها لود نشدند لاگ چاپ شده و برنامه بسته میشود
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// اینجا دیتابیس ایجاد شده و یک Pool میسازد
	pool, err := postgres.NewPool(ctx, cfg.Postgres.DSN())

	// مجدد اگر دیتابیس وصل نشد کل برنامه بسته میشود
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	// بعد از بستن برنامه اتصال هم قطع میشود
	defer pool.Close()

	// اتصال به ردیس
	redisClient, err := redispkg.NewClient(
		ctx,
		redispkg.Config{
			Addr:         cfg.Redis.Addr,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
			ConnMaxIdle:  cfg.Redis.ConnMaxIdle,
		},
	)

	// Terminate service
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	// Redis cleanup
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("failed to close redis client: %v", err)
		}
	}()

	// Dependency Injection for database, redis, token and service
	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)
	otpRepo := repository.NewOTPRepository(redisClient)

	tokens := auth.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
	)

	userService := service.NewUserService(
		userRepo,
		otpRepo,
		refreshTokenRepo,
		tokens,
		cfg.JWT.RefreshTokenTTL,
	)

	// اتصال به هندلر gRPC و ثبت آن در سرور gRPC
	userHandler := handler.NewGRPCServer(userService)

	// تعریف لیمیتر
	limiter := ratelimit.New(redisClient)

	// در این قسمت میدل ور ها وارد سرور میشن
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// بیرونی ترین ریکاوری است تا مانع از دسترس خارج شدن سرور بشه
			grpcmiddleware.RecoveryInterceptor(),
			// لایه میانی لاگ میکنه
			grpcmiddleware.LoggingInterceptor(),
			// لایه محدود کننده سرعت
			grpcmiddleware.RateLimitInterceptor(
				limiter,
				rateLimitRules,
				nil,
			),
			// درونی ترین لایه احراز هویت میکنه
			grpcmiddleware.AuthInterceptor(tokens, publicMethods),
		),
	)
	pb.RegisterUserServiceServer(grpcServer, userHandler)

	// فعال کردن رفلکشن ها در سرور تا بتوانیم از طریق ابزار هایی مانند
	// gpcurl با سرور ارتباط برقرار کنیم
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	// Graceful shutdown:
	// وقتی سیگنال SIGINT/SIGTERM دریافت شود، به‌جای قطع ناگهانی
	// اتصال‌ها، فرصت می‌دهیم درخواست‌های در حال پردازش را تمام
	// کند، بعد سرور را متوقف کند.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		log.Println("shutting down user service gracefully...")
		grpcServer.GracefulStop()
	}()

	log.Printf("user service listening on :%s", cfg.GRPCPort)

	// Serve بلاک‌کننده است؛ تا وقتی GracefulStop صدا زده نشود، این
	// خط برنمی‌گردد.
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve grpc server: %v", err)
	}
}
