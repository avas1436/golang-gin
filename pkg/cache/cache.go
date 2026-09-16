// pkg/cache/cache.go

package cache

import (
	"context"
	"encoding/json"
	stdErrors "errors"
	"time"

	appErrors "pkg/errors"

	"github.com/redis/go-redis/v9"
)

// Cache یک wrapper جنریک روی Redis است: هر مقداری را به JSON
// سریالایز/دیسریالایز می‌کند تا هر سرویس بجای بازنویسی مارشال/
// آن‌مارشال برای هر نوع داده، فقط یک نمونه‌ی type-safe از آن
// بسازد. جنریک بودن روی خودِ نوع T است، نه فقط کلید، چون هر سرویس
// معمولاً چند شکل داده‌ی متفاوت را کش می‌کند
// و باید در گت و ست به همان نوع اصلی
// برگردند، نه any که همیشه نیاز به type assertion دستی دارد
type Cache[T any] struct {
	client *redis.Client
}

// New یک Cache جدید برای نوع T می‌سازد
func New[T any](client *redis.Client) *Cache[T] {

	return &Cache[T]{client: client}

}

// Get مقدار ذخیره‌شده در Redis را می‌خواند.
//
// اگر کلید وجود نداشته باشد یا TTL آن تمام شده باشد:
//
//	ok  = false
//	err = nil
//
// بنابراین cache miss یک خطای واقعی محسوب نمی‌شود.
//
// اگر کلید وجود داشته باشد:
//
//	ok = true
//
// و مقدار داخل value قرار می‌گیرد.
func (c *Cache[T]) Get(
	ctx context.Context,
	key string,
) (
	value T,
	ok bool,
	err error,
) {

	// بررسی اتصال کلاینت
	if c == nil || c.client == nil {
		return value, false, appErrors.New(
			appErrors.KindInternal,
			"cache client is nil",
		)
	}

	// اعتبار سنجی کلید
	if key == "" {
		return value, false, appErrors.New(
			appErrors.KindInvalidInput,
			"cache key is empty",
		)
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {

		// redis.Nil یعنی کلید یا هیچ‌وقت ست نشده یا منقضی شده؛
		// هردو از دید فراخواننده «کش نداریم» است، نه یک خطای واقعی
		if stdErrors.Is(err, redis.Nil) {
			return value, false, nil
		}

		return value, false, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to read from cache",
		)
	}

	if err := json.Unmarshal(data, &value); err != nil {

		return value, false, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to unmarshal cached value",
		)

	}

	return value, true, nil
}

// Set مقدار را به‌صورت JSON با TTL مشخص ذخیره می‌کند.
func (c *Cache[T]) Set(
	ctx context.Context,
	key string,
	value T,
	ttl time.Duration,
) error {

	// بررسی اتصال کلاینت
	if c == nil || c.client == nil {
		return appErrors.New(
			appErrors.KindInternal,
			"cache client is nil",
		)
	}

	// اعتبار سنجی کلید
	if key == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"cache key is empty",
		)
	}

	// اعتبار سنجی زمان انقضای کلید
	if ttl <= 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"cache ttl must be greater than zero",
		)
	}

	data, err := json.Marshal(value)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to marshal value for cache",
		)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to write to cache",
		)
	}

	return nil
}

// Delete یک یا چند کلید مشخص را پاک می‌کند
// البته پاک کردن دسته ای با این تابع کار درستی نیست
// چون میتواند باعث قفل شدن ردیس شود
// برای حذف تعداد زیادی key بهتر است از DeleteByPrefix
// استفاده شود
func (c *Cache[T]) Delete(ctx context.Context, keys ...string) error {

	// بررسی اتصال کلاینت
	if c == nil || c.client == nil {
		return appErrors.New(
			appErrors.KindInternal,
			"cache client is nil",
		)
	}

	// اعتبار سنجی آرایه کلید ها
	if len(keys) == 0 {
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to delete cache keys",
		)
	}

	return nil
}

// DeleteByPrefix همه‌ی کلیدهای زیر یک prefix را پاک می‌کند
// عمداً از SCAN استفاده شده، نه KEYS؛ KEYS کل
// Redis را در یک دستور block می‌کند و روی دیتاست بزرگ می‌تواند کل
// سرویس را متوقف کند. SCAN با cursor این کار را دسته‌دسته و بدون
// مسدود کردن سایر دستورات انجام می‌دهد
func (c *Cache[T]) DeleteByPrefix(
	ctx context.Context,
	prefix string,
) error {

	// بررسی اتصال کلاینت
	if c == nil || c.client == nil {
		return appErrors.New(
			appErrors.KindInternal,
			"cache client is nil",
		)
	}

	// اعتبار سنجی عبارت
	if prefix == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"cache prefix is empty",
		)
	}

	// یعنی هربار فقط 100 کلید را بررسی کن
	const scanBatchSize = 100

	// مقدار پیش فرض کرسر صفر است یعنی از اولین کلید شروع میکند
	var cursor uint64

	for {

		// ابتدا کلید ها را در هر ست 100 تایی پیدا میکنیم
		keys, nextCursor, err := c.client.Scan(
			ctx,
			cursor,
			prefix+"*",
			scanBatchSize,
		).Result()

		if err != nil {
			return appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to scan cache keys for invalidation",
			)
		}

		// اگر کلیدی با شرایط ما بود همه را پاک میکنیم
		if len(keys) > 0 {
			if err := c.Delete(ctx, keys...); err != nil {
				return err
			}
		}

		// اینجا مقدار کرسر از آخرین کلید بررسی شده آپدیت میشود
		cursor = nextCursor

		// SCAN وقتی cursor صفر برگرداند یعنی یک دور کامل تمام شده
		if cursor == 0 {
			break
		}
	}

	return nil
}
