package auth

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	if len(password) < 4 {
		return "", fmt.Errorf("Password needs to be at least 4 characters.")
	}
	hashed_password, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hashed_password, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	if len(password) < 4 {
		return false, fmt.Errorf("Password needs to be at least 4 characters.")
	}
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}
