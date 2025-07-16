package rpc

import (
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

// RequestValidator validates RPC requests.
type RequestValidator struct {
	fieldViolations map[string][]string
}

// NewRequestValidator creates a new RequestValidator.
func NewRequestValidator() *RequestValidator {
	return &RequestValidator{
		fieldViolations: make(map[string][]string),
	}
}

// Field begins building a validation for the given request field name.
func (v *RequestValidator) Field(name string) *FieldValidator {
	return &FieldValidator{
		validator: v,
		field:     name,
	}
}

// Error returns a structured connect error if any violations exist.
func (v *RequestValidator) Error() error {
	if len(v.fieldViolations) == 0 {
		return nil
	}

	cErr := connect.NewError(connect.CodeInvalidArgument, errors.New("bad request"))

	badReq := &errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{},
	}

	for field, messages := range v.fieldViolations {
		badReq.FieldViolations = append(badReq.FieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       field,
			Description: strings.Join(messages, ", "),
		})
	}

	detail, err := connect.NewErrorDetail(badReq)
	if err != nil {
		return cErr
	}

	cErr.AddDetail(detail)

	return cErr
}

type FieldValidator struct {
	validator *RequestValidator
	field     string
	condition bool
}

// When sets the condition for the field.
func (f *FieldValidator) When(cond bool) *FieldCondition {
	return &FieldCondition{
		validator: f.validator,
		field:     f.field,
		condition: cond,
	}
}

type FieldCondition struct {
	validator *RequestValidator
	field     string
	condition bool
}

// Message adds a violation if the condition is true.
func (fc *FieldCondition) Message(msg string) *RequestValidator {
	if fc.condition {
		fc.validator.fieldViolations[fc.field] = append(
			fc.validator.fieldViolations[fc.field],
			msg,
		)
	}
	return fc.validator
}

// Messagef adds a formatted violation message if the condition is true.
func (fc *FieldCondition) Messagef(msg string, args ...any) *RequestValidator {
	return fc.Message(fmt.Sprintf(msg, args...))
}
