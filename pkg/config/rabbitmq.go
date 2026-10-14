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

	EnableTLS bool `mapstructure:"enable_tls" yaml:"enable_tls"`

	TLS TLSConfig `mapstructure:"tls" yaml:"tls"`
}

// URL آدرس اتصال AMQP را می‌سازد
func (c RabbitMQConfig) URL() string {
	scheme := "amqp"
	if c.EnableTLS {
		scheme = "amqps"
	}

	return fmt.Sprintf(
		"%s://%s:%s@%s:%s/%s",
		scheme,
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.VHost,
	)
}
