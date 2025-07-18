package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Logger interface {
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
}

// NewEchoLogMiddleware returns an Echo middleware that logs handler requests.
func NewEchoLogMiddleware(logger Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Method == http.MethodOptions {
				return nil
			}

			logArgs := []any{
				"time", v.StartTime.UTC(),
				"http.method", v.Method,
				"http.uri", v.URI,
				"http.status", v.Status,
				"duration", v.Latency.String(),
			}

			level := slog.LevelInfo
			message := "http request"
			if v.Error != nil {
				level = slog.LevelError
				logArgs = append(logArgs, "error", v.Error)
			}

			logger.Log(c.Request().Context(), level, message, logArgs...)
			return v.Error
		},
		LogLatency: true,
		LogMethod:  true,
		LogURI:     true,
		LogStatus:  true,
		LogError:   true,
	})
}

// NewConnectLogInterceptor returns a Connect interceptor that logs handler requests.
func NewConnectLogInterceptor(logger Logger) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			keyVals := []any{
				"protocol", "connect",
			}
			procParts := strings.Split(req.Spec().Procedure, "/")
			if len(procParts) == 3 {
				keyVals = append(keyVals,
					"connect.service", strings.TrimSuffix(procParts[1], "/"),
					"connect.method", procParts[2],
				)
			} else {
				keyVals = append(keyVals, "connect.procedure", req.Spec().Procedure)
			}

			start := time.Now()
			keyVals = append(keyVals, "start_time", start.Format(time.RFC3339))

			level := slog.LevelInfo
			res, err := next(ctx, req)
			if err != nil {
				level = slog.LevelError
				var cErr *connect.Error
				if errors.As(err, &cErr) {
					keyVals = append(keyVals, "error_code", fmt.Sprintf("(%d) %s", cErr.Code(), cErr.Code().String()))
				}
				keyVals = append(keyVals, "error", err)
			}

			keyVals = append(keyVals, "duration", time.Since(start).String())

			logger.Log(ctx, level, "unary rpc called", keyVals...)
			return res, err
		}
	})
}
