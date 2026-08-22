// pkg/auth/access_token.go

package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AccessClaims چیزی است که داخل access token رمزگشایی می‌شود.
// role را عمداً این‌جا نگه می‌داریم چون طبق تصمیم
// معماری پروژه، بررسی RBAC باید در سطح هر سرویس و بر اساس claims
// توکن انجام شود، بدون نیاز به کوئری اضافه به دیتابیس کاربر.
//
// این پکیج نباید به مدل داخلی هیچ سرویسی وابسته باشد، برای همین اینجا رول را ساده به صورت
// رشته نگه می‌داریم؛ تبدیل مدل داخلی رشته بر عهده‌ی سرویسی
// است که از این تابع استفاده می‌کند.
type AccessClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`

	jwt.RegisteredClaims
}

// GenerateAccessToken یک JWT امضاشده با HMAC-SHA256 می‌سازد.
func GenerateAccessToken(
	secret string,
	userID string,
	role string,
	ttl time.Duration,
) (string, error) {

	now := time.Now()

	claims := AccessClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signed, nil
}

// ParseAccessToken امضا و انقضای توکن را بررسی می‌کند و در صورت معتبر
// بودن، claims داخلش را برمی‌گرداند.
func ParseAccessToken(
	secret string,
	tokenString string,
) (*AccessClaims, error) {

	claims := &AccessClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			// این چک جلوی حمله‌ی "alg confusion" رو می‌گیره: بدون این،
			// یه کلاینت مخرب می‌تونه هدر توکن رو عوض کنه و الگوریتم
			// امضا رو به چیزی غیر از HMAC تغییر بده.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf(
					"unexpected signing method: %v", t.Header["alg"],
				)
			}

			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok {
		return nil, fmt.Errorf("invalid access token claims")
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid access token")
	}

	return claims, nil
}
