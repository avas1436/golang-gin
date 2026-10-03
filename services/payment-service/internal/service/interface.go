// services/payment-service/internal/service/interface.go

package service

import (
	"context"

	"github.com/google/uuid"
)

// EventPublisher چیزی است که PaymentService برای انتشار نتیجه‌ی
// نهایی پرداخت نیاز دارد. پیاده‌سازی آن در internal/messaging است
type EventPublisher interface {
	PublishPaymentCompleted(
		ctx context.Context,
		paymentID uuid.UUID,
		orderID uuid.UUID,
		gatewayName string,
		gatewayRefID string,
	) error

	PublishPaymentFailed(
		ctx context.Context,
		paymentID uuid.UUID,
		orderID uuid.UUID,
		reason string,
	) error

	PublishPaymentInitiated(
		ctx context.Context,
		paymentID uuid.UUID,
		orderID uuid.UUID,
		userID uuid.UUID,
		redirectURL string,
		authority string,
	) error
}
