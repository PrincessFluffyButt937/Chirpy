package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	exp := now.Add(expiresIn)
	claims := &jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	string_token, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return string_token, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	s_uuid, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, err
	}
	user_ID, err := uuid.Parse(s_uuid)
	if err != nil {
		return uuid.UUID{}, err
	}
	return user_ID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	b_token := headers.Get("Authorization")
	if b_token == "" {
		return "", fmt.Errorf("No authorization token.")
	}
	if !strings.HasPrefix(b_token, "Bearer ") {
		return "", fmt.Errorf("Invalid bearer token format")
	}
	trimmed_token := strings.TrimPrefix(b_token, "Bearer ")
	return strings.TrimSpace(trimmed_token), nil
}

func GetApiKey(headers http.Header) (string, error) {
	api_key := headers.Get("Authorization")
	if api_key == "" {
		return "", fmt.Errorf("No authorization token.")
	}
	if !strings.HasPrefix(api_key, "ApiKey ") {
		return "", fmt.Errorf("Invalid ApiKey format")
	}
	trimmed_api_key := strings.TrimPrefix(api_key, "ApiKey ")
	return strings.TrimSpace(trimmed_api_key), nil
}

func MakeRefreshToken() string {
	key := make([]byte, 32)
	rand.Read(key)
	return hex.EncodeToString(key)
}
