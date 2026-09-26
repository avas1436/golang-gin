// services/user-service/internal/service/module.go

package service

import (
	"pkg/auth"
	"user-service/config"
	"user-service/internal/repository"

	"go.uber.org/fx"
)

func NewUserService(
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	eventPublisher EventPublisher,
	tokens auth.TokenManager,
	cfg *config.Config,
) *UserService {

	return &UserService{
		userRepo:         userRepo,
		otpRepo:          otpRepo,
		refreshTokenRepo: refreshTokenRepo,
		eventPublisher:   eventPublisher,
		tokens:           tokens,
		refreshTokenTTL:  cfg.JWT.RefreshTokenTTL,
	}

}

// Module ارائه دهنده‌ی لایه Service به سیستم تزریق وابستگی Fx
var Module = fx.Module(
	"service",
	fx.Provide(
		NewUserService,
	),
)
