package http

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v4"
)

type RateLimiter interface {
	Wait(ctx context.Context, key string) error
}

type RateLimitEchoKeyGetter func(c echo.Context) (string, bool)

func NewEchoRateLimiterMiddleware(limiter RateLimiter, keyGetter RateLimitEchoKeyGetter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key, ok := keyGetter(c)
			if !ok {
				key = c.Request().RemoteAddr // default to client IP
			}
			if err := limiter.Wait(c.Request().Context(), key); err != nil {
				if errors.Is(err, context.Canceled) {
					return c.NoContent(http.StatusRequestTimeout)
				}
				return c.NoContent(http.StatusTooManyRequests)
			}
			return next(c)
		}
	}
}

type RateLimitConnectKeyGetter func(req connect.AnyRequest) (string, bool)

func NewConnectRateLimitInterceptor(limiter RateLimiter, keyGetter RateLimitConnectKeyGetter) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			key, ok := keyGetter(req)
			if !ok {
				key = req.Peer().Addr // default to client IP
			}
			if err := limiter.Wait(ctx, key); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil, connect.NewError(connect.CodeCanceled, errors.New("canceled"))
				}
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("resource exhausted"))
			}
			return next(ctx, req)
		}
	})
}
