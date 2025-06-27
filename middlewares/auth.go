package middlewares

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ray-remotestate/restro/config"
	"github.com/ray-remotestate/restro/models"
)

type Claims struct {
	UserID uuid.UUID
	Roles	[]string
	jwt.RegisteredClaims
}

type ContextKey string

const (
	userContextKey ContextKey = "user"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr, err := extractBearerToken(r)
		if err != nil {
			http.Error(w, "unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		token, err:= jwt.ParseWithClaims(tokenStr, claims, func (token *jwt.Token) (interface{}, error)  {
			return []byte(config.SecretKey), nil
		})
		if err != nil || !token.Valid{
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, &claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetAuthenticatedUser(r *http.Request) (*Claims, error) {
	claims, ok := r.Context().Value(userContextKey).(*Claims)
	if !ok {
		return nil, errors.New("no user in context")
	}
	return claims, nil
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header missing")
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid authorization format")
	}
	return parts[1], nil
}

func RoleBasedMiddleware(allowedRoles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool)
	for _, role := range allowedRoles {
		allowed[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := GetAuthenticatedUser(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if any of the user's roles match the allowed ones
			for _, userRole := range claims.Roles {
				if allowed[models.Role(strings.ToLower(userRole))] {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "forbidden: insufficient role", http.StatusForbidden)
		})
	}
}
