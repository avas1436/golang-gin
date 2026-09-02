// pkg/auth/refresh_token.go

package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	appErrors "pkg/errors"
)

// GenerateRefreshToken یک رشته‌ی رندوم و غیرقابل‌ حدس می‌ سازد.
//
// برخلاف access token که JWT است refresh token عمداً یک JWT نیست: چون باید بتوانیم
// آن را قبل از انقضای طبیعی‌اش باطل کنیم — مثلاً هنگام
// لاگ اوت یا اگر مشکوک به لو رفتن باشد
func GenerateRefreshToken() (string, error) {

	// ۳۲ بایت = ۲۵۶ بیت آنتروپی، به‌اندازه‌ی کافی غیرقابل‌حدس
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to generate refresh token",
		)
	}

	return hex.EncodeToString(b), nil
}

// HashRefreshToken قبل از ذخیره در دیتابیس روی توکن اجرا می‌شود.
//
// دلیلش شبیه ذخیره‌ی پسورد است: اگر دیتابیس لو برود، نباید توکن‌های
// خام و قابل‌استفاده در آن باشند. برخلاف پسورد از bcrypt استفاده
// نمی‌کنیم چون خود توکن از قبل کاملاً رندوم و پرآنتروپی است؛ یک هش
// سریع مثل SHA-256 برای این مورد کافی و مناسب‌تر است.
func HashRefreshToken(token string) string {

	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}
