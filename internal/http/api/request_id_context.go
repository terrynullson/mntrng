package api

import "context"

type requestIDContextKey struct{}

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func requestIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	value := ctx.Value(requestIDContextKey{})
	requestID, ok := value.(string)
	if !ok || requestID == "" {
		return "", false
	}
	return requestID, true
}
