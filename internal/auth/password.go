package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt with the specified number of rounds.
func HashPassword(password string, rounds int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), rounds)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword verifies a password against a bcrypt hash.
func VerifyPassword(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, fmt.Errorf("verify password: %w", err)
	}
	return true, nil
}

// PasswordStrength validates password strength.
// Returns validity and a list of error messages (in Swedish).
func PasswordStrength(password string) (bool, []string) {
	var errors []string

	if len(password) < 8 {
		errors = append(errors, "Lösenordet måste vara minst 8 tecken.")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false

	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}

	if !hasUpper {
		errors = append(errors, "Lösenordet måste innehålla minst en stor bokstav.")
	}
	if !hasLower {
		errors = append(errors, "Lösenordet måste innehålla minst en liten bokstav.")
	}
	if !hasDigit {
		errors = append(errors, "Lösenordet måste innehålla minst en siffra.")
	}

	return len(errors) == 0, errors
}

// RandomHex generates a hex-encoded random string of n bytes.
func RandomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
