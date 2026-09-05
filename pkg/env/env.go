// pkg/env/env.go

package env

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Load سعی می‌کند یک فایل .env را از مسیر داده‌شده بخواند و
// مقادیرش را در process environment قرار دهد.
//
// اگر هیچ مسیری داده نشود، پیش‌فرض ".env"
// در نظر گرفته می‌شود. هر سرویس معمولاً این تابع را همان ابتدای
// main.go، قبل از config.Load()، صدا می‌زند.
//
// رفتار کلیدی: این تابع هیچ متغیری را که از قبل در
// process environment وجود دارد override نمی‌کند. نتیجه این می‌شود:
//
// یعنی این تابع عمداً هیچ‌وقت با خطای «فایل پیدا نشد» سرویس را متوقف
// نمی‌کند؛ نبودن .env یک حالت طبیعی است، نه یک خطا.
func Load(paths ...string) {

	if len(paths) == 0 {
		paths = []string{".env"}
	}

	if err := godotenv.Load(paths...); err != nil {
		log.Printf(
			"no .env file found at %v, relying on environment variables already set (e.g. from Docker)",
			paths,
		)
	}
}

// این یک تابع کمکی است و میگوید اگر مقدار بود مقدار را بده در غیر این صورت فالبک بده
func String(key, fallback string) string {

	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

// خروجی عدد میدهد
func Int(key string, fallback int) (int, error) {

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

// Bool مقدار بولین متغیر محیطی را برمی‌گرداند.
func Bool(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf(
			"invalid boolean value for %s: %w",
			key,
			err,
		)
	}
	return b, nil
}

// خروجی زمان را اعتبار سنجی میکند
func Duration(
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
func Require(key string) (string, error) {

	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf(
			"required environment variable %s is not set",
			key,
		)
	}

	return v, nil
}
