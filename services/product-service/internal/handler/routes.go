// services/product-service/internal/handler/routes.go

package handler

import (
	"pkg/grpcmiddleware"
	pb "pkg/proto/product"
	"pkg/ratelimit"
)

// این ها لیست متد های عمومی است (بدون نیاز به JWT معتبر)
// هر متدی در این لیست نباشد، AuthInterceptor برایش Authorization
// header معتبر می‌خواهد
func PublicMethods() map[string]bool {

	return map[string]bool{
		// مرور و جست‌وجوی فروشگاه نباید نیاز به لاگین داشته باشد
		pb.ProductService_GetProduct_FullMethodName:     true,
		pb.ProductService_SearchProducts_FullMethodName: true,

		// این سه متد را Order Service به‌صورت مستقیم (نه از طریق
		// API Gateway) فراخوانی می‌کند، پس هیچ‌وقت JWT کاربر نهایی
		// همراهشان نیست. مرز امنیتی واقعی این‌ها فعلاً شبکه‌ی داخلی
		// Docker Compose است، نه احراز هویت سطح اپلیکیشن؛ اگر روزی
		// این سرویس‌ها خارج از یک شبکه‌ی ایزوله دیپلوی شدند، باید
		// این‌جا یک مکانیزم auth سرویس‌به‌سرویس (مثلاً mTLS یا یک
		// internal token جدا) اضافه شود
		pb.ProductService_ReserveStock_FullMethodName: true,
		pb.ProductService_ReleaseStock_FullMethodName: true,
		pb.ProductService_ConfirmStock_FullMethodName: true,
	}
}

// لیست متد ها به همراه محدودیت ها
func RateLimitRules() map[string]grpcmiddleware.RateLimitRule {

	return map[string]grpcmiddleware.RateLimitRule{
		// نوشتن محدود به ادمین است، پس محدودیت سخت‌گیرانه لازم نیست
		// ولی همچنان به‌عنوان یک لایه‌ی محافظتی خوبه
		pb.ProductService_CreateProduct_FullMethodName: {
			Limit: ratelimit.PerMinute(10),
		},

		pb.ProductService_UpdateProduct_FullMethodName: {
			Limit: ratelimit.PerMinute(30),
		},

		// ترافیک عمومی مرور فروشگاه؛ محدودیت بازتر
		pb.ProductService_GetProduct_FullMethodName: {
			Limit: ratelimit.PerMinute(80),
		},

		pb.ProductService_SearchProducts_FullMethodName: {
			Limit: ratelimit.PerMinute(40),
		},

		// عمداً برای سه متد داخلی Reserve/Release/Confirm قانونی
		// تعریف نشده: چون RateLimitInterceptor وقتی متدی در این
		// map نباشد، بدون محدودیت رد می‌شود و این سه متد فقط توسط
		// Order Service (نه کاربر نهایی) با فرکانس بالا صدا زده
		// می‌شوند، پس Rate Limit کردنشان روی IP معنایی ندارد
	}
}
