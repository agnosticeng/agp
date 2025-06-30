package v1

import (
	"context"

	"github.com/agnosticeng/agp/pkg/client_ip_middleware"
	"github.com/golang-jwt/jwt/v5"
)

type claimsContextKey = struct{}

type Claims struct {
	jwt.RegisteredClaims
	QuotaKey  string `json:"quota_key"`
	Tier      string `json:"tier"`
	IsDefault bool
}

func (c *Claims) InjectIntoContext(ctx context.Context) context.Context {
	return context.WithValue(
		ctx,
		claimsContextKey{},
		c,
	)
}

func (c *Claims) WithDefaultQuotaKey(quotaKey string) *Claims {
	if len(c.QuotaKey) == 0 {
		c.QuotaKey = quotaKey
	}

	return c
}

func ClaimsFromContext(ctx context.Context) *Claims {
	var clientIp = client_ip_middleware.FromContext(ctx)

	if v := ctx.Value(claimsContextKey{}); v != nil {
		return v.(*Claims).WithDefaultQuotaKey(clientIp)
	} else {
		return (&Claims{}).WithDefaultQuotaKey(clientIp)
	}
}
