// pkg/ratelimit/module.go

package ratelimit

import (
	"go.uber.org/fx"

	redispkg "pkg/redis"
)

func NewLimiter(
	client *redispkg.Client,
) Limiter {

	return New(client)

}

var Module = fx.Module(
	"ratelimit",

	fx.Provide(
		NewLimiter,
	),
)
