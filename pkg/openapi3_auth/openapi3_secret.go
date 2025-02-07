package openapi3_auth

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3filter"
)

type OpenAPI3SecretConfig struct {
	AllowEmpty bool
}

func OpenAPI3Secret(secret string, conf OpenAPI3SecretConfig) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, ai *openapi3filter.AuthenticationInput) error {
		if ai.SecuritySchemeName != "Secret" {
			return fmt.Errorf("invalid security scheme: %s", ai.SecuritySchemeName)
		}

		var v string

		if username, _, ok := ai.RequestValidationInput.Request.BasicAuth(); ok {
			v = username
		} else {
			v = ai.RequestValidationInput.Request.Header.Get("Authorization")
		}

		if len(v) == 0 && conf.AllowEmpty {
			return nil
		}

		if len(v) == 0 || v != secret {
			return fmt.Errorf("invalid secret: %s", v)
		}

		return nil
	}
}
