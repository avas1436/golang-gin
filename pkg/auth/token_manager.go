// pkg/auth/token_manager.go

package auth

import "time"

// TokenManager عملیات مربوط به access/refresh token را پشت یک
// abstraction واحد جمع می‌کند.
type TokenManager interface {
	GenerateAccessToken(userID, role string) (string, error)
	ParseAccessToken(tokenString string) (*AccessClaims, error)
	GenerateRefreshToken() (string, error)
	HashRefreshToken(token string) string
}

type tokenManager struct {
	secret         string
	accessTokenTTL time.Duration
}

// NewTokenManager یک TokenManager می‌سازد. secret و مدت اعتبار access
// token فقط همین‌جا نگه داشته می‌شوند، نه در هر سرویسی که ازش استفاده
// می‌کند.
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

	return GenerateAccessToken(m.secret, userID, role, m.accessTokenTTL)
}

// دریافت اطلاعات توکن
func (m *tokenManager) ParseAccessToken(tokenString string) (*AccessClaims, error) {

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
