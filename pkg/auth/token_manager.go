// pkg/auth/token_manager.go

package auth

import "time"

// TokenManager abstraction مربوط به tokenها.
//
// سرویس‌ها به جای وابستگی مستقیم به implementation مربوط به JWT
// فقط به این interface وابسته می‌شوند.
type TokenManager interface {

	// ساخت Access Token
	GenerateAccessToken(userID string, role string) (string, error)

	// بررسی و decode کردن Access Token
	ParseAccessToken(tokenString string) (*AccessClaims, error)

	// ساخت Refresh Token تصادفی
	GenerateRefreshToken() (string, error)

	// ساخت hash از Refresh Token
	HashRefreshToken(token string) string

	// TTL مربوط به Access Token
	AccessTokenTTL() time.Duration
}

type tokenManager struct {
	secret         string
	accessTokenTTL time.Duration
}

// NewTokenManager یک TokenManager می‌سازد.
//
// secret و TTL فقط داخل implementation مربوط به auth نگهداری می‌شوند
// و سرویس‌های مختلف لازم نیست خودشان JWT configuration را مدیریت کنند.
func NewTokenManager(
	secret string,
	accessTokenTTL time.Duration,
) TokenManager {

	return &tokenManager{
		secret:         secret,
		accessTokenTTL: accessTokenTTL,
	}
}

// تابع ساخت اکسس توکن
func (
	m *tokenManager,
) GenerateAccessToken(
	userID, role string,
) (string, error) {

	return GenerateAccessToken(
		m.secret,
		userID,
		role,
		m.accessTokenTTL,
	)
}

// دریافت اطلاعات توکن
func (
	m *tokenManager,
) ParseAccessToken(
	tokenString string,
) (
	*AccessClaims,
	error,
) {

	return ParseAccessToken(m.secret, tokenString)
}

// ساخت رفرش توکن
func (m *tokenManager) GenerateRefreshToken() (string, error) {
	return GenerateRefreshToken()
}

// هش کردن رفرش توکن در دیتابیس
func (m *tokenManager) HashRefreshToken(token string) string {
	return HashRefreshToken(token)
}

// خروجی زمان انقضای اکسس توکن
func (m *tokenManager) AccessTokenTTL() time.Duration {
	return m.accessTokenTTL
}
