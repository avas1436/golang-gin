// services/user-service/internal/model/otp.go

package model

// این قسمت اطلاعات یک چالش فعال را در ردیس موقتا نگه میدارد
type OTPChallenge struct {
	ID          string
	UserID      string
	PhoneNumber string
	Code        string
}
