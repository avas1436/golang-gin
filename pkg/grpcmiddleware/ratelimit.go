// pkg/grpcmiddleware/ratelimit.go

package grpcmiddleware

import (
	"context"
	"net"

	"pkg/ratelimit"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// این همون تایپ فایل مشترک ratelimit است
// در یک مپ مشخص میکند هر متد چه محدودیتی دارد
type RateLimitRule struct {
	Limit ratelimit.Limit
}

// یک تابع که تعیین میکند قسمت آخر کلید بر چه اساسی ساخته شود
// مثلا میتواند آیدی، شماره، یا مهمان تعریف شود
type KeyFunc func(ctx context.Context, req any) string

// به صورت پیش فرض اگر هیچی نباشه این تابع کلید را میسازد
// البته این روش وقتی reverse-proxy داریم کار درستی نیست
func DefaultKeyFunc(ctx context.Context, _ any) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}

	// این قسمت تنها آیپی درخواست را میدهد و پورت را حذف میکند
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}

	return host
}

func RateLimitInterceptor(
	limiter ratelimit.Limiter,
	rules map[string]RateLimitRule,
	keyFunc KeyFunc,
) grpc.UnaryServerInterceptor {

	if keyFunc == nil {
		keyFunc = DefaultKeyFunc
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		rule, exist := rules[info.FullMethod]

		// اگر قانونی تعریف نشده باشد عبور میکند
		if !exist {
			return handler(ctx, req)
		}

		key := "ratelimit:" + info.FullMethod + ":" + keyFunc(ctx, req)

		result, err := limiter.Check(
			ctx,
			key,
			rule.Limit,
		)

		// در صورت مشکل در عملکرد لیمیتر
		// Fail-closed یعنی اگر ردیس خراب بود کلا سرویس کار نکنه
		if err != nil {
			return nil, status.Error(
				codes.Internal,
				"rate limit service unavailable",
			)
		}

		// در صورت محدود شدن
		if !result.Allowed {
			return nil, status.Error(
				codes.ResourceExhausted, // یعنی بیش از حد استفاده شده
				"too many requests, please try again later",
			)
		}

		return handler(ctx, req)
	}
}
