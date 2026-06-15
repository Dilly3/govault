package auth

import (
	"errors"
	"time"

	"github.com/dilly3/govault/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecretKey = config.GetConfig().JWTKey

var jwtSecret = []byte(jwtSecretKey)

type Claims struct {
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

type JWTAuthenticator struct {
	jwtSecretKey string
}

func NewJWTAuthenticator(jwtSecretKey string) *JWTAuthenticator {
	return &JWTAuthenticator{
		jwtSecretKey: jwtSecretKey,
	}
}

func (a *JWTAuthenticator) GenerateToken(userID string, userEmail string, role string) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		UserID:    userID,
		UserEmail: userEmail,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func (a *JWTAuthenticator) RefreshToken(tokenString string) (string, error) {
	claims, err := a.ValidateToken(tokenString)
	if err != nil {
		return "", errors.New("invalid or expired token")
	}
	expirationTime := time.Now().Add(1 * time.Hour)
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(expirationTime)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecretKey)
}

func (a *JWTAuthenticator) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}
	if claims.UserEmail == "" {
		return nil, errors.New("user email not found")
	}
	if claims.Role == "" {
		return nil, errors.New("role not found")
	}
	return claims, nil
}
