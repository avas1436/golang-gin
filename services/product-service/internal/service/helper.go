// services/product-service/internal/service/helper.go

package service

import (
	"context"
	"pkg/auth"
	appErrors "pkg/errors"

	"github.com/google/uuid"
)

// مقدار ثابت نقش ادمین
const roleAdmin = "admin"

// یک تابع مشترک برای چک کردن نقش کاربر
func requireAdmin(ctx context.Context) error {

	// دریافت مقادیر احراز هویت که در کانتکست قرار گرفته
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return appErrors.New(
			appErrors.KindUnauthenticated,
			"authentication required",
		)
	}

	// بررسی میکند نقش ادمین باشد
	if claims.Role != roleAdmin {
		return appErrors.New(
			appErrors.KindPermissionDenied,
			"only admins can perform this action",
		)
	}

	return nil
}

// کار این تابع اینه که تایپ رشته دریافت شده از سرویس های دیگر
// رو به تایپ uuid تبدیل میکنه
func parseProductID(id string) (uuid.UUID, error) {

	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid product id",
		)
	}

	return parsed, nil
}
