// pkg/config/rabbitmq.go

package config

// ساختار مشترک برای فراخوانی rabbitmq
type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
}
