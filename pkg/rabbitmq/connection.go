// pkg/rabbitmq/connection.go

package rabbitmq

// Channel (کانال): در RabbitMQ، ایجاد اتصال شبکه (TCP Connection) هزینه بالایی دارد. به همین دلیل از یک Connection باز استفاده می‌شود و درون آن کانال‌های مجازی (Channels) ایجاد می‌شود. تمام عملیات‌ها (ارسال، دریافت، ساخت صف) روی Channel انجام می‌شود.
// ۲. Queue (صف) و Exchange (مرکز تبادل):

//     Exchange: گیرنده اولیه پیام‌ها از سمت تولیدکننده (Publisher) است.

//     Queue: محل ذخیره پیام‌هاست تا Consumer آن‌ها را بردارد.

//     Binding & Routing Key: برای اینکه Exchange بداند پیام را به کدام Queue بفرستد، از یک کلید مسیریابی (Routing Key) و اتصالی به نام Binding استفاده می‌کند.

//     ۳. Durable (ماندگاری): وقتی یک صف Durable باشد، یعنی اگر سرور RabbitMQ خاموش و روشن شود، صف (و پیام‌های داخل آن در صورت Persistent بودن) از بین نمی‌رود.

//     ۴. ACK و NACK (تایید و عدم تایید): وقتی Consumer پیامی را می‌گیرد، RabbitMQ باید بداند آیا پردازش موفق بوده یا خیر.

//     ACK (Acknowledgement): پردازش موفق بود، پیام را از صف پاک کن.

//     NACK (Negative Acknowledgement): پردازش خطا داشت. (می‌توان پیام را دور انداخت یا دوباره به صف برگرداند - Requeue).

//     ۵. QoS و Prefetch: اگر میلیون‌ها پیام در صف باشد، RabbitMQ نباید همه را یکجا به Consumer بفرستد (چون Ram پر می‌شود). با Prefetch تعیین می‌کنیم که در هر لحظه، Consumer چند پیامِ در حال پردازش (بدون ACK) می‌تواند داشته باشد.

import (
	appErrors "pkg/errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection یک wrapper نازک روی اتصال AMQP است.
type Connection struct {
	conn *amqp.Connection
}

// Connect یک اتصال جدید به RabbitMQ باز می‌کند. url معمولاً از
// commonConfig.RabbitMQConfig.URL() ساخته می‌شود
func Connect(url string) (*Connection, error) {

	conn, err := amqp.Dial(url)

	if err != nil {

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to connect to rabbitmq",
		)

	}

	return &Connection{conn: conn}, nil
}

// Channel یک کانال AMQP جدید از روی Connection می‌سازد.
//
// Publisherها و Consumerها بهتر است Channel اختصاصی خودشان را داشته باشند
// تا lifecycle و خطاهای هر بخش از یکدیگر مستقل باشند.
//
// خود Connection می‌تواند بین چند Publisher و Consumer به اشتراک گذاشته شود.
func (c *Connection) Channel() (*amqp.Channel, error) {

	ch, err := c.conn.Channel()

	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to open rabbitmq channel",
		)
	}

	return ch, nil
}

// قطع اتصال کانال
func (c *Connection) Close() error {
	return c.conn.Close()
}
