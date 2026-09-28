// services/product-service/cmd/main.go

package main

import (
	"pkg/auth"
	"pkg/postgres"
	"pkg/rabbitmq"
	"pkg/ratelimit"
	redispkg "pkg/redis"

	"product-service/config"
	"product-service/internal/cache"
	"product-service/internal/handler"
	"product-service/internal/messaging"
	"product-service/internal/repository"
	"product-service/internal/server"
	"product-service/internal/service"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(

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
		redispkg.Module,

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

		// لایه کشینگ
		cache.Module,

		// لایه منطق تجاری و مدیریت
		service.Module,

		// لایه gRPC Handler
		handler.Module,

		// Consumerها و Publisherهای رویدادها
		messaging.Module,

		// سرور gRPC
		server.Module,
	)

	app.Run()
}
