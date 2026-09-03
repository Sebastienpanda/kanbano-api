package middleware

import "log"

func MustInitJWKS(jwksURL string) {
	if err := InitJWKS(jwksURL); err != nil {
		log.Fatalf("failed to initialize JWKS: %v", err)
	}
	log.Println("JWKS initialized successfully")
}
