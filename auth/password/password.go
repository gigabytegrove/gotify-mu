package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func ValidateNewPassword(pw string) error {
	return ValidateNewPasswordWithMin(pw, 1)
}

func ValidateNewPasswordWithMin(pw string, minLength int) error {
	if pw == "" { return errors.New("password must not be empty") }
	if len([]byte(pw)) > 72 { return bcrypt.ErrPasswordTooLong }
	if minLength < 1 { minLength = 1 }
	if len([]rune(pw)) < minLength {
		return fmt.Errorf("password must be at least %d characters", minLength)
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
