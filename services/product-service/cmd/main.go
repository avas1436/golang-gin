// services/product-service/cmd/main.go

package main

import (
	"product-service/config"
	"product-service/internal/cache"
	"product-service/internal/handler"
	"product-service/internal/platform"
	"product-service/internal/repository"
	"product-service/internal/server"
	"product-service/internal/service"

	"go.uber.org/fx"
)

func main() {

	// fx.New خودش سیگنال‌های سیستم‌عامل (SIGINT/SIGTERM) را هندل
	// می‌کند و به‌ترتیب معکوسِ ساخت، Lifecycle Hook های OnStop را
	// صدا می‌زند؛ برخلاف الگوی دستی User Service (app.New +
	// application.Run)، اینجا دیگر نیازی به نوشتن خودمان context/
	// signal handling نیست
	fx.New(
		config.Module,
		platform.Module,
		repository.Module,
		cache.Module,
		service.Module,
		handler.Module,
		server.Module,
	).Run()
}
