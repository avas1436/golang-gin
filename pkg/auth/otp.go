// pkg/auth/otp.go

package auth

import (
	"crypto/rand" // این پکیج برای تولید اعداد تصادفی امن استفاده می‌شود
	"fmt"
	"math/big"
)

// Generate OTP
func GenerateOTP() (string, error) {

	max := big.NewInt(10000)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate otp: %w", err)
	}

	return fmt.Sprintf("%05d", n.Int64()), nil
}
