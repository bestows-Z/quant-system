package auth

import (
	"context"

	qjwt "quant-system/backend-go/common/jwt"
)

type contextKey string

const ClaimsKey contextKey = "jwt_claims"

func WithClaims(ctx context.Context, claims *qjwt.Claims) context.Context {
	return context.WithValue(ctx, ClaimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (*qjwt.Claims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*qjwt.Claims)
	return claims, ok
}
