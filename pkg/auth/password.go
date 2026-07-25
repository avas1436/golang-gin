package auth

import "golang.org/x/crypto/bcrypt"

// Hash Password
func HashPassword(
	password string,
) (
	string, error,
) {

	hashed, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost, // مقدار 10 راند رو به صورت پیش فرض داره
	)

	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

// Compare Password
func ComparePassword(
	hashedPassword,
	password string,
) error {

	return bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword),
		[]byte(password),
	)
}
