// services/order-service/cmd/main.go

package main

import (
	"go.uber.org/fx"

	"order-service/config"
	"order-service/internal/client"
	"order-service/internal/handler"
	"order-service/internal/messaging"
	"order-service/internal/repository"
	"order-service/internal/server"
	"order-service/internal/service"

	"pkg/auth"
	"pkg/grpcclient"
	"pkg/postgres"
	"pkg/rabbitmq"
	"pkg/ratelimit"
	"pkg/redis"
)

func main() {
	fx.New(
		// ----------------------------------
		// Configuration
		// ----------------------------------
		config.Module,

		// ----------------------------------
		// Shared Infrastructure
		// ----------------------------------
		postgres.Module,
		redis.Module,
		rabbitmq.Module,
		grpcclient.Module,
		ratelimit.Module,
		auth.Module,

		// ----------------------------------
		// Order Service
		// ----------------------------------
		client.Module,
		repository.Module,
		service.Module,
		messaging.Module,
		handler.Module,
		server.Module,
	).Run()
}
