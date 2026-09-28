// services/user-service/internal/model/otp.go

package model

import (
	"time"

	"github.com/google/uuid"
)

// OTPChallenge اطلاعات چالش OTP در ردیس
type OTPChallenge struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	PhoneNumber string    `json:"phone_number"`
	Code        string    `json:"code"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// IsExpired بررسی انقضای کد OTP
func (o *OTPChallenge) IsExpired() bool {

	return time.Now().UTC().After(o.ExpiresAt)

}
