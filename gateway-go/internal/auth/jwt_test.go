package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("test-secret-test-secret-test-secret-0123456789")

func mint(t *testing.T, secret []byte, typ string, ttl time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "42",
		"email": "trader@example.com",
		"role":  "USER",
		"typ":   typ,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func TestVerifyValidAccessToken(t *testing.T) {
	v := NewVerifier(testSecret)
	p, err := v.Verify(mint(t, testSecret, "access", time.Minute))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.UserID != "42" || p.Email != "trader@example.com" || p.Role != "USER" {
		t.Fatalf("unexpected principal: %+v", p)
	}
}

func TestRefreshTokenRejectedAsAccess(t *testing.T) {
	v := NewVerifier(testSecret)
	_, err := v.Verify(mint(t, testSecret, "refresh", time.Minute))
	if !errors.Is(err, ErrNotAccessToken) {
		t.Fatalf("expected ErrNotAccessToken, got %v", err)
	}
}

func TestWrongSecretRejected(t *testing.T) {
	v := NewVerifier(testSecret)
	other := []byte("another-secret-another-secret-0123456789abc")
	if _, err := v.Verify(mint(t, other, "access", time.Minute)); err == nil {
		t.Fatal("expected signature verification to fail")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	v := NewVerifier(testSecret)
	if _, err := v.Verify(mint(t, testSecret, "access", -time.Minute)); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestNoneAlgorithmRejected(t *testing.T) {
	v := NewVerifier(testSecret)
	// Forge an unsigned token; the verifier must refuse non-HMAC algorithms.
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "42", "typ": "access",
		"exp": time.Now().Add(time.Minute).Unix(),
	})
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}
	if _, err := v.Verify(signed); err == nil {
		t.Fatal("expected 'none' algorithm token to be rejected")
	}
}
