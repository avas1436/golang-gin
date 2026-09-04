// pkg/ratelimit/ratelimit.go

package ratelimit

// GCRA
// Generic Cell Rate Algorithm الگوریتم محدود کننده سرعت
// یک الگوریتم برای کنترل نرخ درخواست‌هاست
//  که مشخص می‌کند درخواست بعدی چه زمانی مجاز است.

import (
	"context"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// تعریف تایپ محدود کننده سرعت
type Limit = redis_rate.Limit

// به جای هربار فراخوانی کتابخانه redis_rate از این توابع استفاده میکنیم
func PerSecond(rate int) Limit { return redis_rate.PerSecond(rate) }
func PerMinute(rate int) Limit { return redis_rate.PerMinute(rate) }
func PerHour(rate int) Limit   { return redis_rate.PerHour(rate) }

// Result اطلاعات کامل نتیجه بررسی Rate Limit را نگه می‌دارد
type Result struct {
	// آیا درخواست مجاز است؟
	Allowed bool

	// چند درخواست دیگر در این بازه زمانی مجاز است؟
	Remaining int

	// اگر مجاز نبود، چند ثانیه دیگر باید صبر کند؟
	RetryAfter time.Duration
}

// Limiter یک محدودکننده‌ی نرخ درخواست مبتنی بر Redis
// است که از الگوریتم GCRA استفاده می‌کند.
type Limiter interface {
	Check(
		ctx context.Context,
		key string,
		limit Limit,
	) (*Result, error)
}

type redisLimiter struct {
	limiter *redis_rate.Limiter
}

func New(client *redis.Client) Limiter {
	return &redisLimiter{
		limiter: redis_rate.NewLimiter(client),
	}
}

// Allow بررسی می‌کند که آیا درخواست مجاز است یا خیر، همراه با اطلاعات تکمیلی
func (
	l *redisLimiter,
) Check(
	ctx context.Context,
	key string,
	limit Limit,
) (
	*Result,
	error,
) {

	res, err := l.limiter.Allow(ctx, key, limit)
	if err != nil {
		return nil, err
	}

	return &Result{
		Allowed:    res.Allowed > 0,
		Remaining:  res.Remaining,
		RetryAfter: res.RetryAfter,
	}, nil
}
