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
		// نوشتن محدود به ادمین است، پس محدودیت سخت‌گیرانه لازم نیست
		// ولی همچنان به‌عنوان یک لایه‌ی محافظتی خوبه
		pb.OrderService_CreateOrder_FullMethodName: {
			Limit: ratelimit.PerMinute(5),
		},

		pb.OrderService_GetOrder_FullMethodName: {
			Limit: ratelimit.PerMinute(30),
		},

		// ترافیک عمومی مرور فروشگاه؛ محدودیت بازتر
		pb.OrderService_ListMyOrders_FullMethodName: {
			Limit: ratelimit.PerMinute(30),
		},
	}
}
