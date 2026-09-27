// services/user-service/cmd/main.go

package main

import (
	"pkg/auth"
	"pkg/postgres"
	"pkg/rabbitmq"
	"pkg/ratelimit"
	"pkg/redis"

	"user-service/config"
	"user-service/internal/handler"
	"user-service/internal/messaging"
	"user-service/internal/repository"
	"user-service/internal/server"
	"user-service/internal/service"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		// ----------------------------------
		// Configuration
		// ----------------------------------

		// بارگذاری کانفیگ پروژه
		config.Module,

		// ----------------------------------
		// Shared Infrastructure
		// ----------------------------------

		// کانکشن پول دیتابیس PostgreSQL
		postgres.Module,

		// اتصال ردیس
		redis.Module,

		// ناشر و مصرف‌کننده RabbitMQ
		rabbitmq.Module,

		// محدودکننده نرخ درخواست
		ratelimit.Module,

		// مدیریت توکن و احراز هویت JWT
		auth.Module,

		// ----------------------------------
		// Payment Service
		// ----------------------------------

		// لایه دسترسی به دیتابیس
		repository.Module,

		// لایه منطق تجاری و مدیریت Saga
		service.Module,

		// لایه gRPC Handler
		handler.Module,

		// Consumerها و Publisherهای رویدادها
		messaging.Module,

		// سرور gRPC
		server.Module,
	).Run()
}
