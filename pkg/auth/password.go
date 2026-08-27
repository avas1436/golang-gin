// pkg/auth/password.go

package auth

import (
	stdErrors "errors"

	"golang.org/x/crypto/bcrypt"

	appErrors "pkg/errors"
)

// HashPassword
func HashPassword(
	password string,
) (
	string, error,
) {

	if password == "" {
		return "", appErrors.New(
			appErrors.KindInvalidInput,
			"password cannot be empty",
		)
	}

	if len(password) < 8 {
		return "", appErrors.New(
			appErrors.KindInvalidInput,
			"password must be at least 8 characters",
		)
	}

	hashed, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost, // مقدار 10 راند رو به صورت پیش فرض داره
	)

	if err != nil {
		return "", appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to hash password",
		)
	}

	return string(hashed), nil
}

// Compare Password
func ComparePassword(
	hashedPassword,
	password string,
) error {

	err := bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword),
		[]byte(password),
	)

	if err != nil {

		// تشخیص نوع خطا
		if stdErrors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {

			return appErrors.New(
				appErrors.KindInvalidInput,
				"invalid password",
			)

		}

		if stdErrors.Is(err, bcrypt.ErrHashTooShort) {

			return appErrors.New(
				appErrors.KindInvalidInput,
				"invalid password format",
			)

		}

		// خطاهای دیگر مانند bcrypt.ErrPasswordTooLong
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to compare password",
		)
	}

	return nil
}
