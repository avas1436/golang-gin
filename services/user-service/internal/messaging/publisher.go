// services/user-service/internal/messaging/publisher.go

package messaging

import (
	"context"
	"time"

	"pkg/events"
	"pkg/rabbitmq"

	"github.com/google/uuid"
)

type RabbitMQEventPublisher struct {
	publisher *rabbitmq.Publisher
}

// NewRabbitMQEventPublisher یک نمونه جدید از EventPublisher می‌سازد
func NewRabbitMQEventPublisher(
	publisher *rabbitmq.Publisher,
) *RabbitMQEventPublisher {
	return &RabbitMQEventPublisher{
		publisher: publisher,
	}
}

// PublishUserOTPRequested رویداد درخواست OTP را برای ارسال پیامک منتشر می‌کند
func (
	p *RabbitMQEventPublisher,
) PublishUserOTPRequested(
	ctx context.Context,
	userID uuid.UUID,
	phoneNumber string,
	code string,
) error {

	event := events.UserOTPRequested{
		EventID:     uuid.New(),
		UserID:      userID,
		PhoneNumber: phoneNumber,
		Code:        code,
		RequestedAt: time.Now().UTC(),
	}

	return p.publisher.Publish(
		ctx,
		events.RoutingKeyUserOTPRequested,
		event,
	)
}
