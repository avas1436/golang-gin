// // services/order-service/config/module.go

package config

import "go.uber.org/fx"

// ارائه خروجی کانفیگ به Fx
var Module = fx.Module(
	"config",
	fx.Provide(Load),
)
