// pkg/rabbitmq/publisher.go

package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	appErrors "pkg/errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

// این ساختار یک فرمت یکسان برای وصل شدن به پیام رسان برای ما ایجاد میکند
type Publisher struct {
	// یک کانال ارتباطی زیرمجموعه یک اتصال
	channel *amqp.Channel

	// مثل یک مرکز مسیر یابی است نه خود صف
	exchange string
}

// برای داشتن یک شیوه اتصال ایمن که تمامی پیام ها به مقصد برسند باید
// Exchange + Queue + Message دارای قابلیت Durability باشند
//
// Exchange -> durable = true
// Queue -> durable = true
// Message -> DeliveryMode: amqp.Persistent

// یک شیوه اتصال با مقصد و روش مشخص برامون میسازه
func NewPublisher(
	channel *amqp.Channel, // نامفهوم
	exchange string, // نامفهوم
) (
	*Publisher,
	error,
) {

	// اعتبار سنجی مقدار کانال
	if channel == nil {
		return nil, appErrors.New(
			appErrors.KindInternal,
			"rabbitmq channel is nil",
		)
	}

	// اعتبار سنجی exchange
	if exchange == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"rabbitmq exchange is empty",
		)
	}

	if err := channel.ExchangeDeclare(
		exchange,

		// یک شیوه آدرس دهی که اجازه داشتن ساختار به روتینگ ما میدهد
		// stock.reserve.requested مثال
		// stock.* میتوان اینطور یک دسته را انتخاب کرد
		"topic",

		// در این حالت بعد از ریست شدن هم رویداد از بین نمیرود
		true, // durable

		// در این حالت به صورت خودکار رویداد ها حذف نمیشوند
		false, // auto-deleted

		false, // internal

		false, // no-wait

		nil,
	); err != nil {

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to declare exchange",
		)

	}

	// بررسی عملکرد صحیح چنل
	if err := channel.Confirm(false); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to enable publisher confirms",
		)
	}

	return &Publisher{channel: channel, exchange: exchange}, nil
}

func (
	p *Publisher,
) Publish(
	ctx context.Context,
	routingKey string,
	payload any,
) error {

	if routingKey == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"rabbitmq routing key is empty",
		)
	}

	// محتوی درخواست به بایت تبدیل میشود
	body, err := json.Marshal(payload)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to marshal event payload",
		)
	}

	err = p.channel.PublishWithContext(
		// کانتکست
		ctx,

		p.exchange,

		// آدرس صف
		routingKey,

		// اگر فعال باشد و هیچ صفی برای این آدرس نبود پیام از بین میرود
		true, // mandatory

		// مهم نیست و بهتره غیر فعال باشه
		false, // immediate

		// فرمت پیام و محتوی آن را تعیین میکند
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)

	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to publish event",
		)
	}

	// منتظر ACK شدن پیام توسط RabbitMQ می‌مانیم.
	ack := <-p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	if !ack.Ack {
		return appErrors.New(
			appErrors.KindInternal,
			"rabbitmq rejected published message",
		)
	}

	return nil
}
