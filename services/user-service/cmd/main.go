// services/user-service/cmd/main.go

package main

import (
	"context"
	"log"

	"pkg/auth"
	"pkg/env"
	"pkg/postgres"

	"user-service/config"
	"user-service/internal/repository"
	"user-service/internal/service"
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
	redisClient, err := repository.NewRedisClient(
		ctx,
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
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

	tokens := auth.NewTokenManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL)

	userService := service.NewUserService(
		userRepo,
		otpRepo,
		refreshTokenRepo,
		tokens,
		cfg.JWT.RefreshTokenTTL,
	)

	_ = userService // هنوز لایه هندلر نوشته نشده

	log.Println("User service started")
}
