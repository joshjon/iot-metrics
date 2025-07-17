package rpc

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

type Logger interface {
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
}

func NewLogInterceptor(logger Logger) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			level := slog.LevelInfo
			keyVals := initConnectLogKeyVals(req.Spec())
			start := time.Now()
			keyVals = append(keyVals, "connect.start_time", start.Format(time.RFC3339))

			res, err := next(ctx, req)
			if err != nil {
				level = slog.LevelError
				var cErr *connect.Error
				if errors.As(err, &cErr) && cErr.Code() == connect.CodeInvalidArgument {
					for _, d := range cErr.Details() {
						detail, dErr := d.Value()
						if dErr != nil {
							continue
						}
						if br, ok := detail.(*errdetails.BadRequest); ok && len(br.FieldViolations) > 0 {
							var fvStrs []string
							for _, fv := range br.FieldViolations {
								fvStrs = append(fvStrs, fv.Field+": "+fv.Description)
							}
							keyVals = append(keyVals, "connect.bad_request.field_violations", fvStrs)
						}
					}
				}
				keyVals = append(keyVals, "connect.Error", err)
			}

			keyVals = append(keyVals, "connect.duration", time.Since(start).String())

			logger.Log(ctx, level, "unary rpc called", keyVals...)
			return res, err
		}
	})
}

func initConnectLogKeyVals(spec connect.Spec) []any {
	keyVals := []any{
		"protocol", "connect",
	}

	procParts := strings.Split(spec.Procedure, "/")
	if len(procParts) == 3 {
		keyVals = append(keyVals,
			"connect.service", strings.TrimSuffix(procParts[1], "/"),
			"connect.method", procParts[2],
		)
	} else {
		keyVals = append(keyVals, "connect.procedure", spec.Procedure)
	}

	return keyVals
}
