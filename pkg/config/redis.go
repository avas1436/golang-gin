// pkg/config/redis.go

package config

import "time"

// RedisConfig ساختار مشترک تنظیمات ردیس برای تمام سرویس‌ها
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	ConnMaxIdle  time.Duration
}
