package platform_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"ai-playground/internal/platform"
)

var (
	testPrivKey *ecdsa.PrivateKey
	testPubKey  *ecdsa.PublicKey
)

func init() {
	var err error
	testPrivKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic("failed to generate test ECDSA key: " + err.Error())
	}
	testPubKey = &testPrivKey.PublicKey
}

func makeToken(t *testing.T, privKey *ecdsa.PrivateKey, sub, fullName string, expired bool) string {
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
	tok, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(privKey)
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return tok
}

func TestValidateJWT_Valid(t *testing.T) {
	tok := makeToken(t, testPrivKey, "auth-uuid", "Alice", false)
	authID, name, err := platform.ValidateJWT(tok, testPubKey)
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

func TestValidateJWT_WrongKey(t *testing.T) {
	tok := makeToken(t, testPrivKey, "id", "Bob", false)
	wrongKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	_, _, err := platform.ValidateJWT(tok, &wrongKey.PublicKey)
	if err == nil {
		t.Error("expected error for wrong key, got nil")
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	tok := makeToken(t, testPrivKey, "id", "Carol", true)
	_, _, err := platform.ValidateJWT(tok, testPubKey)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestValidateJWT_Empty(t *testing.T) {
	_, _, err := platform.ValidateJWT("", testPubKey)
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}
}
