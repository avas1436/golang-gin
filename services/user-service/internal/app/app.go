// services/user-service/internal/app/app.go

package app

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"pkg/auth"
	"pkg/env"
	"pkg/grpcmiddleware"
	"pkg/postgres"
	pb "pkg/proto/user"
	"pkg/ratelimit"
	redispkg "pkg/redis"
	"syscall"
	"user-service/config"
	"user-service/internal/handler"
	"user-service/internal/repository"
	"user-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	listener   net.Listener
	cleanup    func()
}

func New(ctx context.Context) (*App, error) {

	// لود کردن فایل متغیر های محیطی در محیط پردازش برنامه
	env.Load(".env")

	// بارگذاری اولیه کانفیگ ها و استفاده از آنها در کل پروژه با cfg
	cfg, err := config.Load()

	// اگر کانفیگ ها لود نشدند لاگ چاپ شده و برنامه بسته میشود
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// ----------------------------------
	// 1. Database Connections
	// ----------------------------------

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
	cleanup := func() {
		pool.Close()
		if err := redisClient.Close(); err != nil {
			log.Printf("failed to close redis: %v", err)
		}
	}

	// ----------------------------------
	// 2. Dependency Injection
	// ----------------------------------

	// Repo DI
	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)
	otpRepo := repository.NewOTPRepository(redisClient)

	// Token Manager DI
	tokens := auth.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
	)

	// Service DI
	userService := service.NewUserService(
		userRepo,
		otpRepo,
		refreshTokenRepo,
		tokens,
		cfg.JWT.RefreshTokenTTL,
	)

	// ----------------------------------
	// 3. gRPC Server & Interceptors
	// ----------------------------------

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
				handler.RateLimitRules(),
				nil,
			),
			// درونی ترین لایه احراز هویت میکنه
			grpcmiddleware.AuthInterceptor(
				tokens,
				handler.PublicMethods(),
			),
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

	return &App{
		cfg:        cfg,
		grpcServer: grpcServer,
		listener:   lis,
		cleanup:    cleanup,
	}, nil

}

func (a *App) Run() error {
	defer a.cleanup()

	// Graceful Shutdown Channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		log.Println("shutting down gRPC server gracefully...")
		a.grpcServer.GracefulStop()
	}()

	log.Printf("user service listening on :%s", a.cfg.GRPCPort)
	return a.grpcServer.Serve(a.listener)

}
