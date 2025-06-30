package openapi3_auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/golang-jwt/jwt/v5"
)

type ContextClaims interface {
	jwt.Claims
	InjectIntoContext(context.Context) context.Context
}

type OpenAPI3JWTConfig struct {
	Key        string
	KeySetUrls []string
}

func OpenAPI3JWT[T any, PT interface {
	ContextClaims
	*T
}](conf OpenAPI3JWTConfig) (openapi3filter.AuthenticationFunc, error) {
	var kf jwt.Keyfunc

	if len(conf.KeySetUrls) > 0 {
		k, err := keyfunc.NewDefault(conf.KeySetUrls)

		if err != nil {
			return nil, err
		}

		kf = k.Keyfunc
	} else {
		kf = func(t *jwt.Token) (interface{}, error) {
			return []byte(conf.Key), nil
		}
	}

	return func(ctx context.Context, ai *openapi3filter.AuthenticationInput) error {
		var claims T

		if ai.SecuritySchemeName != "Secret" {
			return fmt.Errorf("invalid security scheme: %s", ai.SecuritySchemeName)
		}

		var v = ai.RequestValidationInput.Request.Header.Get("Authorization")
		v = strings.TrimPrefix(v, "Bearer ")
		v = strings.TrimPrefix(v, "bearer ")

		_, err := jwt.ParseWithClaims(v, PT(&claims), kf)

		if err != nil {
			return err
		}

		var rCtx = PT(&claims).InjectIntoContext(ai.RequestValidationInput.Request.Context())
		*ai.RequestValidationInput.Request = *ai.RequestValidationInput.Request.WithContext(rCtx)
		return nil
	}, nil
}
