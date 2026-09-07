// api/internal/handler/auth/dto.go

package auth

import pb "pkg/proto/user"

// فرمت ارور
type errorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}

// پاسخ های متنی
type messageResponse struct {
	Message string `json:"message" example:"logged out"`
}

// پاسخ احراز هویت
type authResponseBody struct {
	AccessToken string    `json:"access_token" example:"eyJhbGciOiJIUzI1Ni..."`
	ExpiresIn   int64     `json:"expires_in" example:"900"`
	User        *userBody `json:"user,omitempty"`
}

// نمایش اطلاعات کاربر
type userBody struct {
	ID          string `json:"id" example:"usr_12345"`
	Email       string `json:"email" example:"user@example.com"`
	PhoneNumber string `json:"phone_number" example:"+989123456789"`
	FullName    string `json:"full_name" example:"John Doe"`
	Role        string `json:"role" example:"USER"`
	CreatedAt   string `json:"created_at" example:"2026-01-01T10:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-01-01T10:00:00Z"`
}

// درخواست ثبت نام
type registerRequestBody struct {
	PhoneNumber string `json:"phone_number" binding:"required" example:"+989123456789"`
	Email       string `json:"email" example:"user@example.com"`
	Password    string `json:"password" binding:"required" example:"SecretPass123"`
	FullName    string `json:"full_name" example:"John Doe"`
}

// درخواست ورود با رمز عبور
type passwordLoginRequestBody struct {
	Identifier string `json:"identifier" binding:"required" example:"user@example.com"`
	Password   string `json:"password" binding:"required" example:"SecretPass123"`
}

// درخواست ورود با OTP
type otpLoginRequestBody struct {
	PhoneNumber string `json:"phone_number" binding:"required" example:"09123456789"`
}

// پاسخ ارسال کد برای کاربر
type otpLoginResponseBody struct {
	ChallengeID      string `json:"challenge_id" example:"ch_987654321"`
	ExpiresInSeconds int64  `json:"expires_in_seconds" example:"120"`
}

// تایید کد ورود OTP
type verifyOTPRequestBody struct {
	ChallengeID string `json:"otp_challenge_id" binding:"required" example:"ch_987654321"`
	Code        string `json:"otp_code" binding:"required" example:"123456"`
}

// مپر ساخت ساختار کاربر
func toUserBody(u *pb.User) *userBody {
	if u == nil {
		return nil
	}
	return &userBody{
		ID:          u.Id,
		Email:       u.Email,
		PhoneNumber: u.PhoneNumber,
		FullName:    u.FullName,
		Role:        u.Role.String(),
		CreatedAt:   u.CreatedAt.String(),
		UpdatedAt:   u.UpdatedAt.String(),
	}
}
