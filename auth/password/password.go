package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func ValidateNewPassword(pw string, minimumLength ...int) error {
	if pw == "" {
		return errors.New("password must not be empty")
	}
	minimum := 1
	if len(minimumLength) > 0 && minimumLength[0] > minimum {
		minimum = minimumLength[0]
	}
	if len([]rune(pw)) < minimum {
		return fmt.Errorf("password must be at least %d characters", minimum)
	}
	if len([]byte(pw)) > 72 {
		return bcrypt.ErrPasswordTooLong
	}
	return nil
}

// CreatePassword returns a hashed version of the given password.
func CreatePassword(pw string, strength int) ([]byte, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pw), strength)
	return hashedPassword, err
}

// ComparePassword compares a hashed password with its possible plaintext equivalent.
func ComparePassword(hashedPassword, password []byte) bool {
	return bcrypt.CompareHashAndPassword(hashedPassword, password) == nil
}
