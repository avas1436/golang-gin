// pkg/auth/module.go

package auth

import (
	"go.uber.org/fx"

	commonConfig "pkg/config"
)

func NewTokenManagerProvider(
	cfg commonConfig.JWTConfig,
) TokenManager {
	return NewTokenManager(
		cfg.Secret,
		cfg.AccessTokenTTL,
	)
}

var Module = fx.Module(
	"auth",
	fx.Provide(
		NewTokenManagerProvider,
	),
)
