// pkg/rabbitmq/consumer.go

package rabbitmq

import (
	"context"

	appErrors "pkg/errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

// HandlerFunc منطق پردازش یک پیام را تعریف می‌کند.
//
// این یک تایپ است که ظاهرِ تابع پردازش‌گر را تعریف می‌کند.
// هر تابعی که بخواهد پیامی را پردازش کند، باید یک کانتکست و بادی یعنی
// متن پیام به صورت باینری بگیرد و در صورت بروز مشکل ارور برگرداند.
//
// اگر handler بدون خطا برگردد:
//
//	پیام ACK می‌شود.
//
// اگر handler خطا برگرداند:
//
//	پیام NACK می‌شود.
//
// تصمیم اینکه پیام دوباره تلاش شود یا به مسیر دیگری
// منتقل شود، متعلق به policy مربوط به Consumer/Queue است.
//
// خود Handler باید idempotent باشد، چون RabbitMQ می‌تواند
// یک پیام را بیش از یک بار تحویل دهد.
type HandlerFunc func(
	ctx context.Context,
	body []byte,
) error

// ساختار Consumer
type Consumer struct {
	channel *amqp.Channel
}

// سازنده Consumer
func NewConsumer(
	channel *amqp.Channel,
) (
	*Consumer,
	error,
) {

	if channel == nil {
		return nil, appErrors.New(
			appErrors.KindInternal,
			"rabbitmq channel is nil",
		)
	}

	return &Consumer{
		channel: channel,
	}, nil
}

// اتصال صف به مرکز تبادل
// ساخت Exchange بهتر است در topology مربوط به RabbitMQ انجام شود.
// در سیستم‌های بزرگ، Consumer نباید Exchange را بسازد
// چون وظیفه Publisher یا زیرساخت است،
// فقط باید صف خودش را بسازد و به آن Exchange متصل کند (Binding).
func (c *Consumer) BindQueue(
	queueName string,
	exchange string,
	routingKey string,
) error {

	if queueName == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"queue name is empty",
		)
	}

	if exchange == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"exchange name is empty",
		)
	}

	if routingKey == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"routing key is empty",
		)
	}

	_, err := c.channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)

	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to declare rabbitmq queue",
		)
	}

	if err := c.channel.QueueBind(
		queueName,
		routingKey,
		exchange,
		false,
		nil,
	); err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to bind rabbitmq queue",
		)
	}

	return nil
}

// Consume پیام‌ها را از Queue دریافت و به Handler تحویل می‌دهد.
//
// auto-ack عمداً false است تا پیام فقط بعد از پردازش موفق ACK شود.
func (c *Consumer) Consume(
	ctx context.Context,
	queueName string,
	handler HandlerFunc,
) error {

	if handler == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"consumer handler is nil",
		)
	}

	if queueName == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"queue name is empty",
		)
	}

	// این بخش از بمباران شدن مصرف کننده جلوگیری می‌کند! عدد
	//  10 یعنی RabbitMQ نهایتاً ۱۰ پیام را به این
	// مصرف کننده می‌دهد. تا زمانی که این مصرف کننده یکی از آن‌ها را ACK
	// نکند، پیام یازدهمی در کار نخواهد بود. این برای کنترل ترافیک و
	//  توزیع عادلانه بار (Load Balancing) بین چند Consumer حیاتی است.
	if err := c.channel.Qos(
		// یعنی نهایتا 10 پیام به مصرف کننده میدهد
		10, // prefetch count
		0,  // prefetch size
		false,
	); err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to configure rabbitmq qos",
		)
	}

	msgs, err := c.channel.Consume(
		queueName,
		"",

		// یعنی پس از ارسال پیام خودکار آن را پاک نکن
		false, // auto-ack = false
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)

	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to start rabbitmq consumer",
		)
	}

	for {
		select {

		case <-ctx.Done():
			return nil

		case msg, ok := <-msgs:

			if !ok {
				return appErrors.New(
					appErrors.KindInternal,
					"rabbitmq consumer channel closed",
				)
			}

			// اگر handler موفق شد، پیام را ACK می‌کنیم.
			if err := handler(ctx, msg.Body); err == nil {

				if err := msg.Ack(false); err != nil {
					return appErrors.Wrap(
						appErrors.KindInternal,
						err,
						"failed to ack rabbitmq message",
					)
				}

				continue
			}

			if err := msg.Nack(false, true); err != nil {
				return appErrors.Wrap(
					appErrors.KindInternal,
					err,
					"failed to nack rabbitmq message",
				)
			}
		}
	}
}
