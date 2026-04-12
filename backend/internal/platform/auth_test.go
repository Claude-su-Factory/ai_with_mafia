package platform_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"ai-playground/internal/platform"
)

func makeToken(t *testing.T, secret, sub, fullName string, expired bool) string {
	t.Helper()
	exp := time.Now().Add(time.Hour)
	if expired {
		exp = time.Now().Add(-time.Hour)
	}
	claims := jwt.MapClaims{
		"sub": sub,
		"exp": exp.Unix(),
		"user_metadata": map[string]any{"full_name": fullName},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return tok
}

func TestValidateJWT_Valid(t *testing.T) {
	tok := makeToken(t, "secret", "auth-uuid", "Alice", false)
	authID, name, err := platform.ValidateJWT(tok, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authID != "auth-uuid" {
		t.Errorf("authID = %q, want %q", authID, "auth-uuid")
	}
	if name != "Alice" {
		t.Errorf("displayName = %q, want %q", name, "Alice")
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	tok := makeToken(t, "secret", "id", "Bob", false)
	_, _, err := platform.ValidateJWT(tok, "wrong")
	if err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	tok := makeToken(t, "secret", "id", "Carol", true)
	_, _, err := platform.ValidateJWT(tok, "secret")
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestValidateJWT_Empty(t *testing.T) {
	_, _, err := platform.ValidateJWT("", "secret")
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}
}
