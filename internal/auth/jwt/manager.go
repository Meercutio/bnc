package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type Manager struct {
	secret    []byte
	accessTTL time.Duration
}

func New(secret string, accessTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL}
}

func (m *Manager) NewAccessToken(userID string) (string, error) {
	now := time.Now()
	claims := jwtlib.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwtlib.NewNumericDate(now),
		ExpiresAt: jwtlib.NewNumericDate(now.Add(m.accessTTL)),
	}
	t := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return t.SignedString(m.secret)
}

func (m *Manager) ValidateAccessToken(token string) (string, error) {
	parsed, err := jwtlib.ParseWithClaims(token, &jwtlib.RegisteredClaims{}, func(t *jwtlib.Token) (any, error) {
		if t.Method != jwtlib.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*jwtlib.RegisteredClaims)
	if !ok || !parsed.Valid || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
