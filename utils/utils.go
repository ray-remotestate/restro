package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"github.com/ray-remotestate/restro/config"
	"github.com/ray-remotestate/restro/middlewares"
)

func GenerateTokens(userID uuid.UUID, roles []string) (accessToken string, refreshToken string, err error) {
	now := time.Now()

	accessToken, err = GenerateAccessToken(userID, roles)
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(config.SecretKey))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func GenerateAccessToken(userID uuid.UUID, roles []string) (accessToken string, err error) {
	now := time.Now()

	accessClaims := &middlewares.Claims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(config.SecretKey))
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

func HashPassword(pw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(bytes), err
}