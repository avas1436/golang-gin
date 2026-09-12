// pkg/config/rabbitmq.go

package config

import "fmt"

// ساختار مشترک برای فراخوانی rabbitmq
type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
}

// URL آدرس اتصال AMQP را می‌سازد
func (c RabbitMQConfig) URL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.VHost,
	)
}
