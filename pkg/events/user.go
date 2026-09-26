// pkg/events/user.go

package events

import (
	"time"

	"github.com/google/uuid"
)

// کلیدهای مسیریابی مربوط به رویدادهای کاربر
const (
	RoutingKeyUserOTPRequested = "user.otp.requested"
)

// UserOTPRequested رویدادی است که هنگام درخواست کد تأیید (OTP) منتشر می‌شود
// تا notification-service آن را دریافت کرده و پیامک ارسال کند
type UserOTPRequested struct {
	EventID     uuid.UUID `json:"event_id"`
	UserID      uuid.UUID `json:"user_id"`
	PhoneNumber string    `json:"phone_number"`
	Code        string    `json:"code"`
	RequestedAt time.Time `json:"requested_at"`
}
