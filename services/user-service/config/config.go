// services/user-service/config/config.go

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// این فایل مسیول نگهداری تنظیمات اجرای این سرویس است
// هر زمینه به ساختار های کوچک تر تقسیم شده تا بتوان هر بخش را جدا پاس داد
type Config struct {
	GRPCPort string

	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

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

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	ConnMaxIdle  time.Duration
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// Load مقادیر را از متغیرهای محیطی می‌خواند.
// در صورت نبود کانفیگیوریشن های fail-fast برنامه درجا بسته میشود
func Load() (*Config, error) {

	// fail-fast
	dbPassword, err := requireEnv("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	// fail-fast
	jwtSecret, err := requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	accessTTL, err := envDuration("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	refreshTTL, err := envDuration("JWT_REFRESH_TTL", 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	redisDB, err := envInt("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	redisPoolSize, err := envInt("REDIS_POOL_SIZE", 10)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	redisMinIdleConns, err := envInt("REDIS_MIN_IDLE_CONNS", 2)
	if err != nil {
		return nil, err
	}

	// این هم مهمه ولی مقدار پیش فرض داره
	redisConnMaxIdle, err := envDuration(
		"REDIS_CONN_MAX_IDLE",
		5*time.Minute,
	)
	if err != nil {
		return nil, err
	}

	// در اینجا مقادیر وارد کانفیگ میشه
	cfg := &Config{
		GRPCPort: envString("GRPC_PORT", "50051"),

		Postgres: PostgresConfig{
			Host:     envString("DB_HOST", "localhost"),
			Port:     envString("DB_PORT", "5432"),
			User:     envString("DB_USER", "user_service"),
			Password: dbPassword,
			DBName:   envString("DB_NAME", "user_service_db"),
			SSLMode:  envString("DB_SSLMODE", "disable"),
		},

		Redis: RedisConfig{
			Addr:         envString("REDIS_ADDR", "localhost:6379"),
			Password:     envString("REDIS_PASSWORD", ""),
			DB:           redisDB,
			PoolSize:     redisPoolSize,
			MinIdleConns: redisMinIdleConns,
			ConnMaxIdle:  redisConnMaxIdle,
		},

		JWT: JWTConfig{
			Secret:          jwtSecret,
			AccessTokenTTL:  accessTTL,
			RefreshTokenTTL: refreshTTL,
		},
	}

	return cfg, nil
}

// این یک تابع کمکی است و میگوید اگر مقدار بود مقدار را بده در غیر این صورت فالبک بده
func envString(key, fallback string) string {

	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

// خروجی عدد میدهد
func envInt(key string, fallback int) (int, error) {

	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %w", key, err)
	}

	return n, nil
}

// خروجی زمان را اعتبار سنجی میکند
func envDuration(
	key string,
	fallback time.Duration,
) (
	time.Duration,
	error,
) {

	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}

	return d, nil
}

// درصورت نبود متغیر ارور میدهد
func requireEnv(key string) (string, error) {

	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf(
			"required environment variable %s is not set",
			key,
		)
	}

	return v, nil
}
