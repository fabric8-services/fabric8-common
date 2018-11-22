package auth

import (
	"context"

	"github.com/fabric8-services/fabric8-common/log"
	"github.com/pkg/errors"
)

type contextTMKey int

const (
	_ = iota
	//contextTokenManagerKey is a key that will be used to put and to get `tokenManager` from goa.context
	contextTokenManagerKey contextTMKey = iota
)

// ReadManagerFromContext extracts the token manager from the context and returns it
func ReadManagerFromContext(ctx context.Context) (Manager, error) {
	tm := ctx.Value(contextTokenManagerKey)
	if tm == nil {
		log.Error(ctx, map[string]interface{}{
			"token": tm,
		}, "missing token manager")

		return nil, errors.New("missing token manager")
	}
	return tm.(Manager), nil
}

// ContextWithTokenManager injects tokenManager in the context for every incoming request
// Accepts Token.Manager in order to make sure that correct object is set in the context.
// Only other possible value is nil
func ContextWithTokenManager(ctx context.Context, tm interface{}) context.Context {
	return context.WithValue(ctx, contextTokenManagerKey, tm)
}
