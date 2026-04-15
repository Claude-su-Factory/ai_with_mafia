package platform

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type supabaseClaims struct {
	jwt.RegisteredClaims
	UserMetadata struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	} `json:"user_metadata"`
}

// ValidateJWT validates a Supabase ES256 JWT and returns (authID, displayName, error).
func ValidateJWT(tokenStr string, pubKey *ecdsa.PublicKey) (authID, displayName string, err error) {
	if tokenStr == "" {
		return "", "", errors.New("empty token")
	}
	token, err := jwt.ParseWithClaims(tokenStr, &supabaseClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return "", "", err
	}
	claims, ok := token.Claims.(*supabaseClaims)
	if !ok || !token.Valid {
		return "", "", errors.New("invalid token claims")
	}
	authID = claims.Subject
	displayName = claims.UserMetadata.FullName
	if displayName == "" {
		displayName = claims.UserMetadata.Name
	}
	if displayName == "" && len(authID) >= 8 {
		displayName = authID[:8]
	}
	return authID, displayName, nil
}
