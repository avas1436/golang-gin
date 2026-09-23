// services/payment-service/internal/handler/routes.go

package handler

import (
	"pkg/grpcmiddleware"
	pb "pkg/proto/payment"
	"pkg/ratelimit"
)

// PublicMethods لیست متدهای عمومی است (بدون نیاز به JWT معتبر)
// متد GetPaymentByOrderID نیازمند احراز هویت (و نقش admin) است، پس این نقشه خالی برمی‌گردد
func PublicMethods() map[string]bool {
	return map[string]bool{}
}

// RateLimitRules قوانین محدودیت تعداد درخواست به ازای هر متد را مشخص می‌کند
func RateLimitRules() map[string]grpcmiddleware.RateLimitRule {
	return map[string]grpcmiddleware.RateLimitRule{
		pb.PaymentService_GetPaymentByOrderID_FullMethodName: {
			Limit: ratelimit.PerMinute(30),
		},
	}
}
