package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"

var jwks keyfunc.Keyfunc

func InitJWKS(jwksURL string) error {
	var err error
	jwks, err = keyfunc.NewDefault([]string{jwksURL})
	return err
}

func AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "token manquant", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, jwks.Keyfunc, jwt.WithValidMethods([]string{"EdDSA"}))
		if err != nil || !token.Valid {
			http.Error(w, "token invalide", http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userID := claims["sub"].(string)

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
