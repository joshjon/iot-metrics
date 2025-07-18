package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v4"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

// RestErrorResponse is a REST style error payload for JSON responses.
type RestErrorResponse struct {
	Error RestError `json:"error"`
}

// RestError represents a structured error for REST responses.
type RestError struct {
	Code    int      `json:"-"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

func (e RestError) Error() string {
	if len(e.Details) > 0 {
		return fmt.Sprintf("%s: %s", e.Message, strings.Join(e.Details, "; "))
	}
	return e.Message
}

// BadRequestError represents a bad request error with field level validation
// for both REST and Connect handlers.
type BadRequestError struct {
	FieldViolations map[string][]string
}

func (e *BadRequestError) Error() string {
	if len(e.FieldViolations) == 0 {
		return "bad request"
	}
	var fvErrs []error
	for f, v := range e.FieldViolations {
		fvErrs = append(fvErrs, fmt.Errorf("%s: %s", f, strings.Join(v, ", ")))
	}
	return fmt.Sprintf("bad request: %s", errors.Join(fvErrs...).Error())
}

// RestError converts a BadRequestError into a RestError.
func (e *BadRequestError) RestError() RestError {
	var fvErrs []string
	for f, v := range e.FieldViolations {
		fvErrs = append(fvErrs, fmt.Sprintf("%s: %s", f, strings.Join(v, ", ")))
	}
	return RestError{
		Code:    http.StatusBadRequest,
		Message: http.StatusText(http.StatusBadRequest),
		Details: fvErrs,
	}
}

// ConnectError converts a BadRequestError into a connect.Error.
func (e *BadRequestError) ConnectError() *connect.Error {
	cErr := connect.NewError(connect.CodeInvalidArgument, errors.New("bad request"))
	if len(e.FieldViolations) == 0 {
		return cErr
	}

	badReq := &errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{},
	}

	for f, v := range e.FieldViolations {
		badReq.FieldViolations = append(badReq.FieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       f,
			Description: strings.Join(v, ", "),
		})
	}

	detail, err := connect.NewErrorDetail(badReq)
	if err != nil {
		return cErr
	}

	cErr.AddDetail(detail)

	return cErr
}

// NewEchoErrorMiddleware returns an Echo middleware that transforms errors
// into structured responses.
func NewEchoErrorMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err == nil {
				return nil
			}

			restErr := func() RestError {
				var echoErr *echo.HTTPError
				if errors.As(err, &echoErr) {
					msg := http.StatusText(echoErr.Code)
					if echoErr.Message != nil {
						msg = fmt.Sprintf("%v", echoErr.Message)
					}
					return RestError{
						Code:    echoErr.Code,
						Message: msg,
					}
				}

				var brErr *BadRequestError
				if errors.As(err, &brErr) {
					return brErr.RestError()
				}

				return RestError{
					Code:    http.StatusInternalServerError,
					Message: http.StatusText(http.StatusInternalServerError),
				}
			}()

			return c.JSON(restErr.Code, RestErrorResponse{Error: restErr})
		}
	}
}

// NewConnectErrorInterceptor returns a Connect interceptor that transforms
// errors into structured connect.Errors.
func NewConnectErrorInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			res, err := next(ctx, req)
			if err == nil {
				return res, nil
			}

			var cerr *connect.Error
			if errors.As(err, &cerr) {
				return nil, cerr
			}

			var brErr *BadRequestError
			if errors.As(err, &brErr) {
				return nil, brErr.ConnectError()
			}

			return nil, connect.NewError(connect.CodeInternal, errors.New("internal server error"))
		}
	})
}
