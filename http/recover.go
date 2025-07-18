package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
)

// WithConnectRecover adds a Connect interceptor that recovers from panics.
func WithConnectRecover(logger Logger) connect.HandlerOption {
	return connect.WithRecover(func(ctx context.Context, spec connect.Spec, header http.Header, recovered any) error {
		logger.Log(ctx, slog.LevelError, "recovered from rpc handler panic", "procedure", spec.Procedure, "recovered", recovered)
		return connect.NewError(connect.CodeInternal, errors.New("internal Error"))
	})
}
