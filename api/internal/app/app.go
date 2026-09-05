// api/internal/app/app.go

package app

import (
	"api/config"
	"api/internal/client"
	"api/internal/handler"
	"api/internal/router"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"pkg/env"
	"pkg/ratelimit"
	redispkg "pkg/redis"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type App struct {
	cfg         *config.Config
	httpServer  *http.Server
	redisClient *redispkg.Client
	userClient  *client.UserClient
}

func New(ctx context.Context) (*App, error) {

	// لود کردن فایل متغیرهای محیطی
	env.Load(".env")

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 1. Redis Connection
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
	// در صورت ارور در اجرای ردیس بسته هم بشه
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// در صورتی که در حین اجرای برنامه اروری رخ دهد، ردیس بسته شود
	var success bool
	defer func() {
		if !success {
			_ = redisClient.Close()
		}
	}()

	// 2. gRPC Clients
	userClient, err := client.NewUserClient(cfg.UserServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("connect user-service: %w", err)
	}
	defer func() {
		if !success {
			_ = userClient.Close()
		}
	}()

	// 4. Middleware
	limiter := ratelimit.New(redisClient)

	// 5. Handlers
	authHandler := handler.NewAuthHandler(userClient, cfg.Cookie)
	// orderHandler := handler.NewOrderHandler(userClient, cfg.Cookie)
	// productHandler := handler.NewProductHandler(userClient, cfg.Cookie)

	handlers := router.Handlers{
		Auth: authHandler,
		// Order: orderHandler,
		// Product: productHandler,
	}

	// 6. Router & Server
	engine := gin.New()

	router.Setup(router.Config{
		Engine:   engine,
		Handlers: handlers,
		Limiter:  limiter,
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// تمام منابع با موفقیت راه‌اندازی شدند
	success = true

	return &App{
		cfg:         cfg,
		httpServer:  httpServer,
		redisClient: redisClient,
		userClient:  userClient,
	}, nil
}

func (a *App) Run(ctx context.Context) error {

	// استفاده از signal.NotifyContext
	// که راهکار مدرن و گو-آیدیوماتیک مدیریت سیگنال است
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)

	go func() {
		log.Printf("api gateway listening on :%s", a.cfg.HTTPPort)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {

	case err := <-serverErr:
		return fmt.Errorf("http server failure: %w", err)

	case <-ctx.Done():
		log.Println("received shutdown signal")

	}

	log.Println("shutting down api gateway gracefully...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown failure: %w", err)
	}

	log.Println("api gateway HTTP server stopped")
	return nil
}

// Close تمامی اتصالات دیتابیس و کلاینت‌های gRPC را به‌صورت متمرکز می‌بندد
func (a *App) Close() error {
	log.Println("closing application resources...")

	var errs []error

	if a.userClient != nil {
		if err := a.userClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("user client close: %w", err))
		}
	}

	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			errs = append(
				errs,
				fmt.Errorf("redis client close: %w",
					err,
				),
			)
		}
	}

	return errors.Join(errs...)
}
