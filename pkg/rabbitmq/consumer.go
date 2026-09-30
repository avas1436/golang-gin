// pkg/rabbitmq/consumer.go

package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"

	appErrors "pkg/errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

// تعریف سقف مجاز تلاش مجدد
const maxRetryCount = 3

// isPermanentError خطاهای غیرقابل جبران
// (مانند عدم تطابق فیلدها یا خراب بودن فرمت JSON) را شناسایی می‌کند.
// پیام با فرمت خراب با دوباره فرستاده
// شدن به صف (requeue=true) هرگز درست نمی‌شود.
func isPermanentError(err error) bool {
	var syntaxErr *json.SyntaxError
	var unmarshalErr *json.UnmarshalTypeError

	return errors.As(err, &syntaxErr) || errors.As(err, &unmarshalErr)
}

// getRetryCount تعداد دفعات رد شدن پیام را از هدر x-death
// که خود RabbitMQ پر می‌کند استخراج می‌کند.
// دلیل محاسبه دقیق و توزیع‌شده تعداد تلاش مجدد بدون نیاز
// به نگهداری state در حافظه برنامه.
func getRetryCount(headers amqp.Table) int {
	xDeath, ok := headers["x-death"].([]any)
	if !ok || len(xDeath) == 0 {
		return 0
	}

	if deathMap, ok := xDeath[0].(amqp.Table); ok {
		if count, ok := deathMap["count"].(int64); ok {
			return int(count)
		}
	}

	return 0
}

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

	// اعتبار سنجی نام صف
	if queueName == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"queue name is empty",
		)
	}

	// اعتبار سنجی exchange
	if exchange == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"exchange name is empty",
		)
	}

	// اعتبار سنجی آدرس روتینگ
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

	// بررسی اتصال چنل مصرف کننده
	if c == nil || c.channel == nil {
		return appErrors.New(
			appErrors.KindInternal,
			"rabbitmq consumer is not initialized",
		)
	}

	// اعتبار سنجی هندلر
	if handler == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"consumer handler is nil",
		)
	}

	// اعتبار سنجی نام صف
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
			} else {

				// اگر خطا ساختاری/غیرقابل جبران است (مثل Unmarshal failure)
				// دلیل: requeue = false داده می‌شود
				// تا پیام حذف شده یا مستقیماً به DLQ برود
				// و باعث حلقه بی‌نهایت پردازش نشود.
				if isPermanentError(err) {
					if nackErr := msg.Nack(false, false); nackErr != nil {
						return appErrors.Wrap(
							appErrors.KindInternal,
							nackErr,
							"failed to nack poison message",
						)
					}
					continue
				}

				// بررسی سقف تعداد مجاز تلاش مجدد (Max Retries / Redelivered)
				// دلیل: اگر پیام با خطای موقتی بیش از حد مجاز
				// (مثلا ۳ بار) شکست بخورد، به DLQ منتقل می‌شود.
				retryCount := getRetryCount(msg.Headers)
				if msg.Redelivered || retryCount >= maxRetryCount {
					if nackErr := msg.Nack(false, false); nackErr != nil {
						return appErrors.Wrap(
							appErrors.KindInternal,
							nackErr,
							"failed to nack message to dlq",
						)
					}
					continue
				}

				// برای خطاهای موقتی و قبل از رسیدن به سقف تلاش مجدد
				// -> Requeue
				if nackErr := msg.Nack(false, true); nackErr != nil {
					return appErrors.Wrap(appErrors.KindInternal, nackErr, "failed to nack rabbitmq message for requeue")
				}
			}
		}
	}
}

// قطع کننده اتصال مصرف کننده
func (c *Consumer) Close() error {

	if c == nil || c.channel == nil {
		return nil
	}

	return c.channel.Close()
}
