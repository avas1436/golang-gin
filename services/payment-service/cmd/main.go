// services/payment-service/cmd/main.go

package main

import (
	"pkg/auth"
	"pkg/grpcclient"
	"pkg/postgres"
	"pkg/rabbitmq"
	"pkg/ratelimit"
	"pkg/redis"

	"payment-service/config"
	"payment-service/internal/client"
	"payment-service/internal/handler"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	"payment-service/internal/server"
	"payment-service/internal/service"

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

		grpcclient.Module,

		// محدودکننده نرخ درخواست
		ratelimit.Module,

		// مدیریت توکن و احراز هویت JWT
		auth.Module,

		// ----------------------------------
		// Payment Service
		// ----------------------------------

		// کلاینت ارتباط با درگاه پرداخت زرین پال
		client.Module,

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
