// services/product-service/internal/messaging/module.go

package messaging

import (
	"context"
	"log"

	"pkg/events"
	"pkg/rabbitmq"

	"go.uber.org/fx"
)

// RegisterConsumers صف‌ها را تعریف و Bind کرده و شنود رویدادها را در Goroutineهای مجزا آغاز می‌کند
func RegisterConsumers(
	lc fx.Lifecycle,
	consumer *rabbitmq.Consumer,
	releaseConsumer *StockReleaseConsumer,
	confirmConsumer *StockConfirmConsumer,
) {
	// ساخت کانتکست قابل لغو برای مدیریت خروج تمیز کانسومرها در OnStop
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(
		fx.Hook{
			OnStart: func(startCtx context.Context) error {

				// ۱. ساخت و Bind کردن صف آزادسازی رزرو انبار (Stock Release)
				if err := consumer.BindQueue(
					events.QueueProductStockRelease,
					events.ExchangeOrderEvents,
					events.RoutingKeyStockReleaseRequested,
				); err != nil {

					return err

				}

				// ۲. ساخت و Bind کردن صف قطعی‌سازی کسر انبار (Stock Confirm)
				if err := consumer.BindQueue(
					events.QueueProductStockConfirm,
					events.ExchangeOrderEvents,
					events.RoutingKeyStockConfirmRequested,
				); err != nil {

					return err
				}

				// اجرای شنود صف Stock Release در یک Goroutine مجزا (جلوگیری از Blocking)
				go func() {
					log.Println(
						"product-service: starting stock release consumer...",
					)
					if err := consumer.Consume(
						ctx,
						events.QueueProductStockRelease,
						releaseConsumer.Handle,
					); err != nil {

						log.Printf("product-service: stock release consumer stopped with error: %v", err)
					}
				}()

				// اجرای شنود صف Stock Confirm در یک Goroutine مجزا
				go func() {
					log.Println("product-service: starting stock confirm consumer...")
					if err := consumer.Consume(
						ctx,
						events.QueueProductStockConfirm,
						confirmConsumer.Handle,
					); err != nil {

						log.Printf("product-service: stock confirm consumer stopped with error: %v", err)
					}
				}()

				return nil
			},

			OnStop: func(stopCtx context.Context) error {
				log.Println(
					"product-service: stopping rabbitmq consumers...",
				)

				cancel() // خروج از حلقه متد Consume در تمام کانسومرها

				return nil
			},
		},
	)
}

// Module مربوط به مدیریت پیام‌رسانی و دریافت رویدادها در Product Service
var Module = fx.Module(
	"messaging",

	fx.Provide(
		NewStockReleaseConsumer,
		NewStockConfirmConsumer,
	),

	fx.Invoke(
		RegisterConsumers,
	),
)
