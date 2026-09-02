// pkg/auth/otp.go

package auth

import (
	"crypto/rand" // این پکیج برای تولید اعداد تصادفی امن استفاده می‌شود
	"fmt"
	"math/big"
	appErrors "pkg/errors"
)

// Generate OTP
func GenerateOTP() (string, error) {

	max := big.NewInt(100000)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to generate otp",
		)
	}

	return fmt.Sprintf("%05d", n.Int64()), nil
}
