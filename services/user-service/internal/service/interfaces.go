// services/user-service/internal/service/interfaces.go

package service

import (
	"context"

	"github.com/google/uuid"
)

// EventPublisher اینترفیس ناشر رویدادهای user-service است
type EventPublisher interface {
	PublishUserOTPRequested(
		ctx context.Context,
		userID uuid.UUID,
		phoneNumber string,
		code string,
	) error
}
