// Package auth verifies the JWT access tokens issued by the Java API. The
// gateway is a verifier only: it shares the HS256 secret but never mints tokens.
package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var ErrNotAccessToken = errors.New("token is not an access token")

// Principal is the authenticated identity extracted from a valid access token.
type Principal struct {
	UserID string
	Email  string
	Role   string
}

type Verifier struct {
	secret []byte
}

func NewVerifier(secret []byte) *Verifier { return &Verifier{secret: secret} }

// Verify validates the signature and expiry, enforces HS256 (so a caller can't
// downgrade to "alg":"none"), and rejects refresh tokens presented as access
// tokens.
func (v *Verifier) Verify(tokenString string) (Principal, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return Principal{}, err
	}

	if typ, _ := claims["typ"].(string); typ != "access" {
		return Principal{}, ErrNotAccessToken
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return Principal{}, errors.New("token missing subject")
	}
	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)

	return Principal{UserID: sub, Email: email, Role: role}, nil
}
