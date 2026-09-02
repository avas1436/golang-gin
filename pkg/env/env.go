// pkg/env/env.go

package env

import (
	"log"

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
