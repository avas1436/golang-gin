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
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection یک wrapper نازک روی اتصال RabbitMQ است.
//
// Connection مربوط به کل ارتباط TCP با RabbitMQ است.
//
// Channel یک کانال AMQP جدید از روی Connection می‌سازد.
//
// چند Publisher و Consumer می‌توانند یک Connection را
// به صورت مشترک استفاده کنند.
//
// اما هر Publisher / Consumer بهتر است Channel اختصاصی
// خودش را داشته باشد.
//
// TODO: فعلا این نظم در سرویس ها نیست
type Connection struct {
	conn *amqp.Connection

	// این پارامتر برای این است که تنها یکبار قابلیت اجرا به یک تابع بدهیم
	closeOnce sync.Once
}

// Connect یک اتصال جدید به RabbitMQ باز می‌کند.
// url = commonConfig.RabbitMQConfig.URL()
func Connect(url string) (*Connection, error) {

	// اعتبار سنجی آدرس اتصال
	if url == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"rabbitmq url is empty",
		)
	}

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

// Channel یک Channel جدید روی Connection ایجاد می‌کند.
//
// هر Publisher یا Consumer بهتر است Channel مخصوص خودش
// را داشته باشد.
//
// Connection مشترک است، Channelها مستقل هستند.
func (c *Connection) Channel() (*amqp.Channel, error) {

	// تست اینکه کانکشن خالی نباشد
	if c == nil || c.conn == nil {
		return nil, appErrors.New(
			appErrors.KindInternal,
			"rabbitmq connection is nil",
		)
	}

	// تست اینکه اتصال برقرار است
	if c.conn.IsClosed() {
		return nil, appErrors.New(
			appErrors.KindInternal,
			"rabbitmq connection is closed",
		)
	}

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

// Close اتصال RabbitMQ را می‌بندد.
//
// closeOnce باعث می‌شود اگر به هر دلیلی Close چند بار
// فراخوانی شد، فقط یک بار عملیات واقعی انجام شود.
func (c *Connection) Close() error {

	// تست اینکه کانکشن خالی نباشد
	if c == nil || c.conn == nil {
		return nil
	}

	var err error

	c.closeOnce.Do(func() {
		err = c.conn.Close()
	})

	return err
}
