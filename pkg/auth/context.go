// pkg/auth/context.go
package auth

// به طور کلی کار این فایل این است که محتوی احراز هویت
// کاربر را داخل کانتکست قرار میدهد تا هرجایی بتوان آنرا بررسی کرد
// این کار در میدل ور احراز هویت انجام میشود

import "context"

// یک نوع داده اختصاصی برای کلید کانتکست
type contextKey int

// یک کلید اختصاصی برای برچسب های داخل کانتکست که ثابت است
const claimsContextKey contextKey = iota

// این تابع اطلاعات احراز هویت کاربر را داخل کانتکست قرار داده
// و یک کانتکست جدید برای کاربر ارایه میدهد
func ContextWithClaims(
	ctx context.Context,
	claims *AccessClaims,
) context.Context {

	return context.WithValue(
		ctx,              // context
		claimsContextKey, // key
		claims,           // value
	)
}

// این تابع اطلاعات احراز هویت را از کانتکست استخراج میکند
func ClaimsFromContext(
	ctx context.Context,
) (
	*AccessClaims,
	bool,
) {

	claims, ok := ctx.Value(claimsContextKey).(*AccessClaims)

	return claims, ok
}
