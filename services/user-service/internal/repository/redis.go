// services/user-service/internal/repository/redis.go

package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// کلاینت اتصال به ردیس را میسازد و یک کانکشن پول هم برایش ایجاد میکند
func NewRedisClient(
	ctx context.Context,
	addr string,
	password string,
	db int,
) (
	*redis.Client, error,
) {

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}
