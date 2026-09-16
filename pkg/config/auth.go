// pkg/config/auth.go

package config

import "time"

// ساختار مشترک همه سرویس ها
type JWTConfig struct {
	Secret         string
	AccessTokenTTL time.Duration
}

// این ساختار تنها مختص سرویس کاربران است
type RefreshTokenConfig struct {
	TTL time.Duration
}
