package jwt

import (
	"time"

	gjwt "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint64   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	gjwt.RegisteredClaims
}

func Generate(secret string, userID uint64, username string, roles []string, expire time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: gjwt.RegisteredClaims{
			ExpiresAt: gjwt.NewNumericDate(time.Now().Add(expire)),
			IssuedAt:  gjwt.NewNumericDate(time.Now()),
		},
	}
	token := gjwt.NewWithClaims(gjwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func Parse(secret, tokenText string) (*Claims, error) {
	token, err := gjwt.ParseWithClaims(tokenText, &Claims{}, func(_ *gjwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, gjwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
