// api/cmd/main.go

package main

import (
	"context"
	"log"

	_ "api/docs" // لود خودکار داکیومنت‌های Swagger
	"api/internal/app"
)

// @title           API Gateway
// @version         1.0
// @description     Enterprise Microservices API Gateway
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	ctx := context.Background()

	application, err := app.New(ctx)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("error closing app resources: %v", err)
		}
	}()

	if err := application.Run(ctx); err != nil {
		log.Fatalf("application run error: %v", err)
	}
}
