// pkg/grpcclient/module.go

package grpcclient

import (
	"context"

	"go.uber.org/fx"
)

func RegisterLifecycle(
	lc fx.Lifecycle,
	manager *ClientManager,
) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return manager.CloseAll()
		},
	})
}

var Module = fx.Module(
	"grpcclient",

	fx.Provide(
		NewManager,
	),

	fx.Invoke(
		RegisterLifecycle,
	),
)
