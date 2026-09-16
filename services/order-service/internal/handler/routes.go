// services/order-service/internal/handler/routes.go

package handler

import (
	"pkg/grpcmiddleware"
	pb "pkg/proto/order"
	"pkg/ratelimit"
)

// این ها لیست متد های عمومی است (بدون نیاز به JWT معتبر)
// هر متدی در این لیست نباشد، AuthInterceptor برایش Authorization
// header معتبر می‌خواهد
func PublicMethods() map[string]bool {

	return map[string]bool{}
}

// لیست متد ها به همراه محدودیت ها
func RateLimitRules() map[string]grpcmiddleware.RateLimitRule {

	return map[string]grpcmiddleware.RateLimitRule{
		// ایجاد سفارش یک عملیات write است و برای جلوگیری از
		// سوءاستفاده، rate limit نسبتاً سختی دارد.
		pb.OrderService_CreateOrder_FullMethodName: {
			Limit: ratelimit.PerMinute(5),
		},

		// دریافت لیست سفارش‌های کاربر احراز هویت‌شده
		// با rate limit مناسب برای جلوگیری از abuse.
		pb.OrderService_GetOrder_FullMethodName: {
			Limit: ratelimit.PerMinute(30),
		},

		pb.OrderService_ListMyOrders_FullMethodName: {
			Limit: ratelimit.PerMinute(30),
		},
	}
}
