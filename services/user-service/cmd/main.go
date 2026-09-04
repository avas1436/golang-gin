// services/user-service/cmd/main.go

package main

import (
	"context"
	"log"
	"user-service/internal/app"
)

func main() {
	ctx := context.Background()

	application, err := app.New(ctx)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("application runtime error: %v", err)
	}
}
