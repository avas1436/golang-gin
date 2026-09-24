// services/payment-service/internal/messaging/publisher.go

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

// NewEventPublisher یک نمونه جدید از EventPublisher می‌سازد
func NewRabbitMQEventPublisher(
	publisher *rabbitmq.Publisher,
) *RabbitMQEventPublisher {

	return &RabbitMQEventPublisher{
		publisher: publisher,
	}
}

func (
	p *RabbitMQEventPublisher,
) PublishPaymentCompleted(
	ctx context.Context,
	paymentID uuid.UUID,
	orderID uuid.UUID,
	gatewayName string,
	gatewayRefID string,
) error {

	event := events.PaymentCompleted{
		EventID:      uuid.New(),
		OrderID:      orderID,
		PaymentID:    paymentID,
		GatewayName:  gatewayName,
		GatewayRefID: gatewayRefID,
		CompletedAt:  time.Now().UTC(),
	}

	return p.publisher.Publish(ctx, events.RoutingKeyPaymentCompleted, event)
}

func (
	p *RabbitMQEventPublisher,
) PublishPaymentFailed(
	ctx context.Context,
	paymentID uuid.UUID,
	orderID uuid.UUID,
	reason string,
) error {

	event := events.PaymentFailed{
		EventID:   uuid.New(),
		OrderID:   orderID,
		PaymentID: paymentID,
		Reason:    reason,
		FailedAt:  time.Now().UTC(),
	}

	return p.publisher.Publish(ctx, events.RoutingKeyPaymentFailed, event)
}
