package api

import (
	"context"
	"strings"

	"github.com/terrynullson/mntrng/internal/domain"
)

type authContextKey struct{}

func withAuthContext(ctx context.Context, value domain.AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey{}, value)
}

func authContextFromRequest(r interface{ Context() context.Context }) (domain.AuthContext, bool) {
	value := r.Context().Value(authContextKey{})
	authContext, ok := value.(domain.AuthContext)
	return authContext, ok
}

func bearerTokenFromHeader(headerValue string) string {
	trimmed := strings.TrimSpace(headerValue)
	if trimmed == "" {
		return ""
	}
	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
