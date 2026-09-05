// pkg/config/postgres.go

package config

import "fmt"

// PostgresConfig ساختار مشترک تنظیمات دیتابیس
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Data Source Name
// DSN آدرس اتصال را به فرمتی که pgxpool انتظار دارد می‌سازد.
func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
}
