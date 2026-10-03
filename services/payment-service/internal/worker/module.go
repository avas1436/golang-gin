// services/payment-service/internal/worker/module.go

package worker

import (
	"context"
	"time"

	"go.uber.org/fx"
)

// ExpirationWorker مدیریت اجرای دوره‌ای منقضی کردن پرداخت‌ها را بر عهده دارد
func NewExpirationWorker(
	service StalePaymentExpirer,
	interval time.Duration,
	staleTimeout time.Duration,
) *ExpirationWorker {

	return &ExpirationWorker{
		service:      service,
		interval:     interval,
		staleTimeout: staleTimeout,
		stopChan:     make(chan struct{}),
	}

}

// RegisterLifecycle اتصال ورکر به Lifecycle Uber FX با Graceful Shutdown واقعی
func RegisterLifecycle(
	lc fx.Lifecycle,
	worker *ExpirationWorker,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			worker.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// ارسال stopCtx به متد Stop جهت منتظر ماندن برای اتمام کار جارِی
			return worker.Stop(ctx)
		},
	})
}

// Module ثبت ماژول ورکر در Uber FX
var Module = fx.Module(
	"worker",

	fx.Provide(
		NewExpirationWorker,
	),

	fx.Invoke(
		RegisterLifecycle,
	),
)
