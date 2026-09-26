// services/user-service/internal/service/module.go

package service

import "go.uber.org/fx"

// Module ارائه دهنده‌ی لایه Service به سیستم تزریق وابستگی Fx
var Module = fx.Module(
	"service",
	fx.Provide(
		NewUserService,
	),
)
