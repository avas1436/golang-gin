// services/user-service/internal/repository/module.go

package repository

import "go.uber.org/fx"

// Module ارائه‌دهنده‌ی تمام پیاده‌سازی‌های لایه Repository به Uber Fx است
var Module = fx.Module(
	"repository",
	fx.Provide(
		NewUserRepository,
		NewRefreshTokenRepository,
		NewOTPRepository,
	),
)
